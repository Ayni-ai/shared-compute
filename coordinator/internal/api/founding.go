package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mcastroarroyo/shared-compute/coordinator/internal/auth"
)

// Founding members: the test-month program (docs/FOUNDING-MEMBERS.md). People enroll
// knowing their devices get test jobs, not regular paid workloads; the coordinator
// validates them from what it observes and numbers them 1 to 1,000. Validated members
// get a public badge and a share link that doubles as their invite link.

const (
	foundingMemberCap    = 1000
	foundingOnlineTarget = 100 // devices online 95% of the time (fleet floor), the program's second goal
	foundingTestMonth    = 30 * 24 * time.Hour
	fleetSampleEvery     = 5 * time.Minute
)

func (s *Server) foundingRules() auth.FoundingRules {
	h := s.cfg.FoundingMinOnlineHours
	if h <= 0 {
		h = 24
	}
	return auth.FoundingRules{MinOnlineMinutes: h * 60, Cap: foundingMemberCap}
}

// apiBase is this API's public origin, where badges and share pages are served.
func (s *Server) apiBase() string {
	if b := strings.TrimRight(s.cfg.AuthCallbackBase, "/"); b != "" {
		return b
	}
	return "https://api.ayni-ai.com"
}

// RunFoundingLoop samples the fleet every five minutes, credits connected devices with
// online time and validates founding members. It needs accounts (Postgres + OAuth).
func (s *Server) RunFoundingLoop(ctx context.Context) {
	if s.auth == nil {
		return
	}
	t := time.NewTicker(fleetSampleEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.foundingTick(ctx)
		}
	}
}

func (s *Server) foundingTick(ctx context.Context) {
	var pks []string
	phones, pcs := 0, 0
	for _, p := range s.reg.Snapshot() {
		pks = append(pks, base64.StdEncoding.EncodeToString(p.StaticPK[:]))
		switch deviceKind(p.Capabilities.Platform) {
		case "phone":
			phones++
		case "pc":
			pcs++
		}
	}
	if err := s.auth.RecordFleetSample(ctx, pks, phones, pcs, int(fleetSampleEvery/time.Minute)); err != nil {
		s.log.Warn("fleet sample failed", "err", err)
	}
	n, err := s.auth.ValidateFoundingMembers(ctx, s.foundingRules())
	if err != nil {
		s.log.Warn("founding validation failed", "err", err)
	} else if n > 0 {
		s.log.Info("founding members validated", "count", n)
	}
}

// --- signed-in ---

// GET /v1/me/founding
func (s *Server) handleMyFounding(w http.ResponseWriter, r *http.Request) {
	u, ok := s.foundingUser(w, r)
	if !ok {
		return
	}
	s.writeMyFounding(w, r, u.ID)
}

// POST /v1/me/founding/enroll {"accept": true}
func (s *Server) handleFoundingEnroll(w http.ResponseWriter, r *http.Request) {
	u, ok := s.foundingUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Accept bool `json:"accept"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "could not parse body")
		return
	}
	if !body.Accept {
		writeError(w, http.StatusBadRequest, "accept_required", "accept the test-month terms to enroll")
		return
	}
	if validated, _, err := s.auth.FoundingCounts(r.Context()); err == nil && validated >= foundingMemberCap {
		writeError(w, http.StatusConflict, "program_full", "all 1,000 founding member places are taken")
		return
	}
	if err := s.auth.EnrollFounding(r.Context(), u.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "enroll_failed", "could not enroll")
		return
	}
	s.log.Info("founding member enrolled")
	s.writeMyFounding(w, r, u.ID)
}

func (s *Server) foundingUser(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	if s.auth == nil {
		writeError(w, http.StatusNotFound, "accounts_disabled", "accounts are not enabled")
		return auth.User{}, false
	}
	u, ok := s.auth.SessionUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "not signed in")
	}
	return u, ok
}

func (s *Server) writeMyFounding(w http.ResponseWriter, r *http.Request, userID string) {
	ctx := r.Context()
	st, err := s.auth.FoundingStatusFor(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "founding_failed", "could not read founding status")
		return
	}
	code, err := s.auth.ReferralCode(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "founding_failed", "could not read invite code")
		return
	}
	validated, _, _ := s.auth.FoundingCounts(ctx)
	rules := s.foundingRules()
	out := map[string]any{
		"enrolled":         st.EnrolledAt != nil,
		"validated":        st.Number > 0,
		"number":           st.Number,
		"enrolled_at":      st.EnrolledAt,
		"validated_at":     st.ValidatedAt,
		"devices":          st.Devices,
		"jobs_served":      st.JobsServed,
		"online_minutes":   st.OnlineMinutes,
		"feedback":         st.Feedback,
		"min_online_hours": rules.MinOnlineMinutes / 60,
		"members":          validated,
		"cap":              foundingMemberCap,
		"open":             validated < foundingMemberCap,
		"code":             code,
		"join_link":        s.joinBase() + "/join/?c=" + code,
		"member_page":      s.joinBase() + "/member/?c=" + code,
		"share_url":        s.apiBase() + "/m/" + code,
		"badge_url":        s.apiBase() + "/v1/members/" + code + "/badge.svg",
		"card_url":         s.apiBase() + "/v1/members/" + code + "/badge.svg?style=card",
	}
	if st.EnrolledAt != nil {
		out["test_month_ends"] = st.EnrolledAt.Add(foundingTestMonth).UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, out)
}

// --- public ---

var foundingPublicCache struct {
	mu   sync.Mutex
	at   time.Time
	body map[string]any
}

// GET /v1/founding: progress toward 1,000 members and 100 devices online at any time.
// Counts only.
func (s *Server) handleFoundingPublic(w http.ResponseWriter, r *http.Request) {
	foundingPublicCache.mu.Lock()
	defer foundingPublicCache.mu.Unlock()
	if time.Since(foundingPublicCache.at) < 30*time.Second && foundingPublicCache.body != nil {
		w.Header().Set("Cache-Control", "public, max-age=30")
		writeJSON(w, http.StatusOK, foundingPublicCache.body)
		return
	}
	rules := s.foundingRules()
	body := map[string]any{
		"members":          0,
		"enrolled":         0,
		"cap":              foundingMemberCap,
		"online_now":       s.reg.Count(),
		"online_target":    foundingOnlineTarget,
		"online_floor_7d":  0,
		"online_median_7d": 0,
		"online_peak_7d":   0,
		"covered_hours":    0.0,
		"min_online_hours": rules.MinOnlineMinutes / 60,
		"generated_at":     time.Now().UTC().Format(time.RFC3339),
	}
	if s.auth != nil {
		// Counts come from the database; if it does not answer, say so (503, not
		// cached) instead of publishing zero members and a zero floor.
		v, e, err := s.auth.FoundingCounts(r.Context())
		if err != nil {
			s.log.Warn("founding counts failed", "err", err)
			writeDBUnavailable(w)
			return
		}
		body["members"], body["enrolled"] = v, e
		fw, err := s.auth.FleetSince(r.Context(), time.Now().Add(-7*24*time.Hour))
		if err != nil {
			s.log.Warn("founding fleet window failed", "err", err)
			writeDBUnavailable(w)
			return
		}
		body["online_floor_7d"], body["online_median_7d"], body["online_peak_7d"] = fw.Floor, fw.Median, fw.Peak
		body["covered_hours"] = fw.CoveredHrs
	}
	foundingPublicCache.at, foundingPublicCache.body = time.Now(), body
	w.Header().Set("Cache-Control", "public, max-age=30")
	writeJSON(w, http.StatusOK, body)
}

var memberCodeRE = regexp.MustCompile(`^[a-z0-9]{4,16}$`)

// memberCache keeps badge and share-page lookups off the database: a badge in a
// README is fetched on every page view.
var memberCache struct {
	mu sync.Mutex
	m  map[string]memberCacheEntry
}

type memberCacheEntry struct {
	at time.Time
	m  auth.PublicMember
	ok bool
}

const memberCacheTTL = 5 * time.Minute

func (s *Server) lookupMember(r *http.Request) (string, auth.PublicMember, bool, error) {
	code := strings.ToLower(strings.TrimSpace(r.PathValue("code")))
	if !memberCodeRE.MatchString(code) || s.auth == nil {
		return code, auth.PublicMember{}, false, nil
	}
	memberCache.mu.Lock()
	if e, hit := memberCache.m[code]; hit && time.Since(e.at) < memberCacheTTL {
		memberCache.mu.Unlock()
		return code, e.m, e.ok, nil
	}
	memberCache.mu.Unlock()
	if allowed, _ := s.memberRL.Allow("member:" + clientIP(r)); !allowed {
		return code, auth.PublicMember{}, false, errRateLimited
	}
	m, ok, err := s.auth.MemberByCode(r.Context(), code)
	if err != nil {
		return code, m, false, err
	}
	memberCache.mu.Lock()
	if memberCache.m == nil || len(memberCache.m) > 10000 {
		memberCache.m = map[string]memberCacheEntry{}
	}
	memberCache.m[code] = memberCacheEntry{at: time.Now(), m: m, ok: ok}
	memberCache.mu.Unlock()
	return code, m, ok, nil
}

var errRateLimited = errors.New("rate limited")

// GET /v1/members/{code}: verify a founding member.
func (s *Server) handleMember(w http.ResponseWriter, r *http.Request) {
	_, m, ok, err := s.lookupMember(r)
	switch {
	case errors.Is(err, errRateLimited):
		writeError(w, http.StatusTooManyRequests, "rate_limited", "slow down")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "member_failed", "could not read member")
	case !ok:
		writeError(w, http.StatusNotFound, "not_a_member", "no validated founding member with that code")
	default:
		w.Header().Set("Cache-Control", "public, max-age=300")
		writeJSON(w, http.StatusOK, map[string]any{
			"number": m.Number, "name": m.Name, "cap": foundingMemberCap,
			"validated_at": m.ValidatedAt.UTC().Format(time.RFC3339),
		})
	}
}

// GET /v1/members/{code}/badge.svg[?style=card]: the shareable badge. The compact
// style is for READMEs and profiles; the card is 1200x630 for social posts.
func (s *Server) handleMemberBadge(w http.ResponseWriter, r *http.Request) {
	_, m, ok, err := s.lookupMember(r)
	if errors.Is(err, errRateLimited) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "slow down")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "member_failed", "could not read member")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "not_a_member", "no validated founding member with that code")
		return
	}
	var svg string
	if r.URL.Query().Get("style") == "card" {
		svg = badgeCardSVG(m)
	} else {
		svg = badgeCompactSVG(m)
	}
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(svg))
}

// badgeCompactSVG is a shields-style two-part badge.
func badgeCompactSVG(m auth.PublicMember) string {
	left, right := "ayni", fmt.Sprintf("founding member #%d", m.Number)
	lw, rw := textWidth11(left)+12, textWidth11(right)+12
	total := lw + rw
	label := html.EscapeString("Ayni: " + right)
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s">`+
		`<title>%s</title>`+
		`<linearGradient id="s" x2="0" y2="100%%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>`+
		`<clipPath id="r"><rect width="%d" height="20" rx="3" fill="#fff"/></clipPath>`+
		`<g clip-path="url(#r)"><rect width="%d" height="20" fill="#1b2a63"/><rect x="%d" width="%d" height="20" fill="#0f6b63"/><rect width="%d" height="20" fill="url(#s)"/></g>`+
		`<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">`+
		`<text x="%d" y="14">%s</text><text x="%d" y="14">%s</text></g></svg>`,
		total, label, label, total, lw, lw, rw, total, lw/2, left, lw+rw/2, html.EscapeString(right))
}

// textWidth11 estimates the rendered width of s in 11px Verdana, which is wide enough
// that the estimate errs on the roomy side for other sans fonts.
func textWidth11(s string) int {
	w := 0.0
	for _, r := range s {
		switch {
		case r == ' ':
			w += 3.9
		case r == '#':
			w += 9
		case r >= '0' && r <= '9':
			w += 7
		case strings.ContainsRune("ilj.:", r):
			w += 3.5
		case strings.ContainsRune("mw", r):
			w += 10.7
		default:
			w += 6.9
		}
	}
	return int(w + 0.5)
}

// badgeCardSVG is the 1200x630 social card.
func badgeCardSVG(m auth.PublicMember) string {
	name := strings.TrimSpace(m.Name)
	nameLine := ""
	if name != "" {
		size := 40 // display names are capped at 40 characters; long ones shrink to stay clear of the seal
		if n := len([]rune(name)); n > 30 {
			size = 26
		} else if n > 22 {
			size = 32
		}
		nameLine = fmt.Sprintf(`<text x="80" y="452" font-size="%d" font-weight="600" fill="#ffffff">%s</text>`, size, html.EscapeString(name))
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630" role="img" aria-label="Ayni founding member #%d">`+
		`<title>Ayni founding member #%d</title>`+
		`<defs><linearGradient id="bg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#14204d"/><stop offset="1" stop-color="#0a1230"/></linearGradient>`+
		`<path id="ring" d="M 960 315 m -150 0 a 150 150 0 1 1 300 0 a 150 150 0 1 1 -300 0"/></defs>`+
		`<rect width="1200" height="630" fill="url(#bg)"/>`+
		`<circle cx="960" cy="315" r="190" fill="none" stroke="#1f9e94" stroke-opacity=".25" stroke-width="2"/>`+
		`<circle cx="960" cy="315" r="122" fill="#0f6b63"/>`+
		`<circle cx="960" cy="315" r="122" fill="none" stroke="#7fd3c9" stroke-width="3"/>`+
		`<text font-family="Inter,Helvetica Neue,Helvetica,Arial,sans-serif" font-size="19" font-weight="700" letter-spacing="6" fill="#7fd3c9">`+
		`<textPath href="#ring">FOUNDING MEMBER · TEST MONTH · VALIDATED · AYNI ·</textPath></text>`+
		`<text x="960" y="300" text-anchor="middle" font-family="Inter,Helvetica Neue,Helvetica,Arial,sans-serif" font-size="30" font-weight="700" fill="#ffffff">#%d</text>`+
		`<text x="960" y="340" text-anchor="middle" font-family="Inter,Helvetica Neue,Helvetica,Arial,sans-serif" font-size="20" fill="#cfe9e6">of %s</text>`+
		`<g font-family="Inter,Helvetica Neue,Helvetica,Arial,sans-serif">`+
		`<text x="80" y="118" font-size="34" font-weight="800" letter-spacing="10" fill="#ffffff">AYNI</text>`+
		`<text x="80" y="236" font-size="26" font-weight="700" letter-spacing="5" fill="#7fd3c9">FOUNDING MEMBER</text>`+
		`<text x="76" y="380" font-size="150" font-weight="800" fill="#ffffff">#%d</text>`+
		`%s`+
		`<text x="80" y="526" font-size="24" fill="#cfd6ea">One of the first %s people whose devices power</text>`+
		`<text x="80" y="560" font-size="24" fill="#cfd6ea">private AI on idle phones and laptops. Validated %s.</text>`+
		`</g></svg>`,
		m.Number, m.Number, m.Number, fmtThousands(foundingMemberCap), m.Number, nameLine,
		fmtThousands(foundingMemberCap), m.ValidatedAt.UTC().Format("January 2006"))
}

func fmtThousands(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

var memberShareTmpl = template.Must(template.New("share").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Title}}</title>
<meta name="description" content="{{.Desc}}">
<meta property="og:type" content="website">
<meta property="og:site_name" content="Ayni">
<meta property="og:title" content="{{.Title}}">
<meta property="og:description" content="{{.Desc}}">
<meta property="og:url" content="{{.ShareURL}}">
<meta property="og:image" content="{{.Image}}">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="{{.Title}}">
<meta name="twitter:description" content="{{.Desc}}">
<meta name="twitter:image" content="{{.Image}}">
<meta http-equiv="refresh" content="0;url={{.Target}}">
</head><body><p><a href="{{.Target}}">{{.Title}}</a></p></body></html>
`))

// GET /m/{code}: the link members share. Link previews (X, LinkedIn, Slack, chat
// apps) read the per-member title from here; people are sent on to the member page.
func (s *Server) handleMemberShare(w http.ResponseWriter, r *http.Request) {
	code, m, ok, err := s.lookupMember(r)
	if errors.Is(err, errRateLimited) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "slow down")
		return
	}
	data := struct{ Title, Desc, ShareURL, Image, Target string }{
		Title:    "Ayni founding members",
		Desc:     fmt.Sprintf("The first %s people whose devices power Ayni, the open-source network for private AI on idle phones and laptops.", fmtThousands(foundingMemberCap)),
		ShareURL: s.apiBase() + "/m/" + code,
		Image:    s.joinBase() + "/founding-og.png",
		Target:   s.joinBase() + "/founders/",
	}
	status := http.StatusNotFound
	if err == nil && ok {
		status = http.StatusOK
		data.Title = fmt.Sprintf("Founding member #%d of Ayni", m.Number)
		if m.Name != "" {
			data.Title = fmt.Sprintf("%s is founding member #%d of Ayni", m.Name, m.Number)
		}
		data.Desc = fmt.Sprintf("One of the first %s people whose devices power Ayni, the open-source network for private AI on idle phones and laptops. Join through this link.", fmtThousands(foundingMemberCap))
		data.Target = s.joinBase() + "/member/?c=" + code
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	h.Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(status)
	_ = memberShareTmpl.Execute(w, data)
}

// --- operator ---

// GET /admin/founding: the program funnel and the fleet floor, for the weekly report
// (scripts/founding-report.py).
func (s *Server) adminFounding(w http.ResponseWriter, r *http.Request) {
	rules := s.foundingRules()
	out := map[string]any{
		"targets":    map[string]int{"members": foundingMemberCap, "online_floor": foundingOnlineTarget},
		"rules":      map[string]int{"min_online_hours": rules.MinOnlineMinutes / 60, "cap": rules.Cap},
		"online_now": s.reg.Count(),
	}
	if s.auth == nil {
		writeJSON(w, http.StatusOK, out)
		return
	}
	ctx := r.Context()
	// A read that fails is named in "errors" with "degraded": true; the weekly report
	// refuses to record a degraded snapshot (scripts/founding-report.py).
	failed := []string{}
	if f, err := s.auth.FoundingFunnelCounts(ctx, rules); err == nil {
		out["funnel"] = f
	} else {
		failed = append(failed, "funnel")
		s.log.Warn("founding funnel failed", "err", err)
	}
	for _, k := range []struct {
		key string
		d   time.Duration
	}{{"fleet_24h", 24 * time.Hour}, {"fleet_7d", 7 * 24 * time.Hour}} {
		if fw, err := s.auth.FleetSince(ctx, time.Now().Add(-k.d)); err == nil {
			out[k.key] = fw
		} else {
			failed = append(failed, k.key)
			s.log.Warn("founding fleet window failed", "window", k.key, "err", err)
		}
	}
	out["degraded"], out["errors"] = len(failed) > 0, failed
	writeJSON(w, http.StatusOK, out)
}
