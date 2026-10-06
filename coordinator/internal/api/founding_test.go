package api

import (
	"encoding/xml"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mcastroarroyo/shared-compute/coordinator/internal/auth"
)

// Without accounts the public progress endpoint still answers with the targets, and
// the member routes say there is nobody to show instead of failing.
func TestFoundingRoutesWithoutAccounts(t *testing.T) {
	f := newConsoleFixture(t)
	code, out, _ := f.do(t, "GET", "/v1/founding", "", nil)
	if code != 200 {
		t.Fatalf("GET /v1/founding = %d", code)
	}
	if num(out, "cap") != 1000 || num(out, "online_target") != 100 || num(out, "min_online_hours") != 24 {
		t.Errorf("targets = %v", out)
	}
	for _, p := range []string{"/v1/me/founding"} {
		if code, _, _ := f.do(t, "GET", p, "", nil); code != 404 {
			t.Errorf("%s without accounts = %d, want 404", p, code)
		}
	}
	if code, _, _ := f.do(t, "POST", "/v1/me/founding/enroll", "", map[string]any{"accept": true}); code != 404 {
		t.Errorf("enroll without accounts = %d, want 404", code)
	}
	for _, p := range []string{"/v1/members/abcd2345", "/v1/members/abcd2345/badge.svg", "/v1/members/BAD..code"} {
		if code, _, _ := f.do(t, "GET", p, "", nil); code != 404 {
			t.Errorf("%s = %d, want 404", p, code)
		}
	}

	req := httptest.NewRequest("GET", "/m/abcd2345", nil)
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	body, _ := io.ReadAll(rec.Body)
	if rec.Code != 404 || !strings.Contains(string(body), "Ayni founding members") || !strings.Contains(string(body), "/founders/") {
		t.Errorf("/m/ unknown = %d %s", rec.Code, body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("/m/ content type = %q", ct)
	}

	if code, _, _ := f.do(t, "GET", "/admin/founding", "", nil); code != 401 {
		t.Errorf("/admin/founding without token = %d, want 401", code)
	}
	code, out, _ = f.do(t, "GET", "/admin/founding", testAdminToken, nil)
	if code != 200 || num(out, "targets", "members") != 1000 || num(out, "targets", "online_floor") != 100 {
		t.Errorf("/admin/founding = %d %v", code, out)
	}
}

// A display name is chosen by the member: the badge must stay well-formed XML and
// carry no markup from it.
func TestBadgesEscapeTheDisplayName(t *testing.T) {
	m := auth.PublicMember{Number: 42, Name: `<script>alert(1)</script> & "Jo"`, ValidatedAt: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)}
	for name, svg := range map[string]string{"compact": badgeCompactSVG(m), "card": badgeCardSVG(m)} {
		if strings.Contains(svg, "<script") {
			t.Errorf("%s badge carries raw markup from the name", name)
		}
		d := xml.NewDecoder(strings.NewReader(svg))
		for {
			_, err := d.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("%s badge is not well-formed XML: %v\n%s", name, err, svg)
			}
		}
		if !strings.Contains(svg, "#42") {
			t.Errorf("%s badge has no number", name)
		}
	}
	if !strings.Contains(badgeCardSVG(m), "October 2026") || !strings.Contains(badgeCardSVG(m), "1,000") {
		t.Errorf("card misses the month or the cap")
	}
	if fmtThousands(1000) != "1,000" || fmtThousands(999) != "999" || fmtThousands(1234567) != "1,234,567" {
		t.Errorf("fmtThousands")
	}
}
