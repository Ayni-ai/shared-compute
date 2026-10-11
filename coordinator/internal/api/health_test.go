package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mcastroarroyo/shared-compute/coordinator/internal/catalog"
	"github.com/mcastroarroyo/shared-compute/coordinator/internal/config"
	"github.com/mcastroarroyo/shared-compute/coordinator/internal/jobs"
	"github.com/mcastroarroyo/shared-compute/coordinator/internal/registry"
	"github.com/mcastroarroyo/shared-compute/coordinator/internal/store"
)

// flakyStore is a memory store whose database "goes away" while down is set: every
// read the health and stats paths depend on fails like an unreachable Postgres.
type flakyStore struct {
	*store.Mem
	down atomic.Bool
}

var errConnRefused = errors.New("dial tcp 10.0.0.3:5432: connect: connection refused")

func (f *flakyStore) Ping(ctx context.Context) error {
	if f.down.Load() {
		return errConnRefused
	}
	return f.Mem.Ping(ctx)
}

func (f *flakyStore) NodeCapabilities(ctx context.Context) ([]store.NodeCapabilityRow, error) {
	if f.down.Load() {
		return nil, errConnRefused
	}
	return f.Mem.NodeCapabilities(ctx)
}

func (f *flakyStore) UsageSince(ctx context.Context, t time.Time) ([]store.UsageRow, error) {
	if f.down.Load() {
		return nil, errConnRefused
	}
	return f.Mem.UsageSince(ctx, t)
}

func (f *flakyStore) EarningsSince(ctx context.Context, t time.Time) ([]store.EarningRow, error) {
	if f.down.Load() {
		return nil, errConnRefused
	}
	return f.Mem.EarningsSince(ctx, t)
}

func (f *flakyStore) WaitlistSince(ctx context.Context, t time.Time) ([]store.WaitlistEntry, error) {
	if f.down.Load() {
		return nil, errConnRefused
	}
	return f.Mem.WaitlistSince(ctx, t)
}

func newFlakyServer(t *testing.T) (*Server, *flakyStore) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Config{
		HTTPAddr: ":0", HeartbeatSeconds: 15, PriceMultiplier: 1, MarketplaceMargin: 0.3,
		QuoteTTLSeconds: 600, JobDeadlineMS: 30000, AdminToken: testAdminToken,
		ConsumerAPIKeys:            map[string]struct{}{"consumer-key-1": {}},
		ProviderRegistrationTokens: map[string]struct{}{"dev-provider-token": {}},
	}
	st := &flakyStore{Mem: store.NewMem(cfg.ConsumerAPIKeys, cfg.ProviderRegistrationTokens)}
	cat, err := catalog.New("", "", log)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(cfg, registry.New(), jobs.New(), st, cat, log)
	_ = st.UpsertNodeCapability(t.Context(), store.NodeCapabilityRow{
		StaticPK: "pk-1", Model: "m", Backend: "llama", DecodeTPS: 20, UpdatedAt: time.Now(),
	})
	return srv, st
}

func get(t *testing.T, h http.Handler, path string, admin bool) (int, map[string]any, http.Header) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if admin {
		req.Header.Set("Authorization", "Bearer "+testAdminToken)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body, rec.Header()
}

// forget clears the server's cached probe and the package-level stats cache, standing
// in for the few seconds that pass between real polls.
func forget(srv *Server) {
	srv.db.mu.Lock()
	srv.db.at = time.Time{}
	srv.db.mu.Unlock()
	publicStatsCache.mu.Lock()
	publicStatsCache.at = time.Time{}
	publicStatsCache.mu.Unlock()
}

func TestHealthReportsDatabaseOutage(t *testing.T) {
	srv, st := newFlakyServer(t)
	h := srv.Handler()

	forget(srv)
	for _, p := range []string{"/health", "/healthz"} {
		code, body, _ := get(t, h, p, false)
		if code != 200 || body["status"] != "ok" || body["db"] != "ok" {
			t.Fatalf("%s with database up = %d %v, want 200 ok/db ok", p, code, body)
		}
	}

	st.down.Store(true)
	forget(srv)
	for _, p := range []string{"/health", "/healthz"} {
		code, body, _ := get(t, h, p, false)
		if code != http.StatusServiceUnavailable || body["status"] != "degraded" || body["db"] != "down" {
			t.Fatalf("%s with database down = %d %v, want 503 degraded/db down", p, code, body)
		}
		if _, ok := body["providers"]; !ok {
			t.Errorf("%s dropped the live providers count; the site falls back to it", p)
		}
	}
	// Liveness never depends on the database: an outage must not restart the instance.
	if code, _, _ := get(t, h, "/livez", false); code != 200 {
		t.Fatalf("/livez with database down = %d, want 200", code)
	}

	st.down.Store(false)
	forget(srv)
	if code, _, _ := get(t, h, "/health", false); code != 200 {
		t.Fatalf("/health after recovery = %d, want 200", code)
	}
}

func TestHealthProbeIsCached(t *testing.T) {
	srv, st := newFlakyServer(t)
	h := srv.Handler()
	forget(srv)
	if code, _, _ := get(t, h, "/health", false); code != 200 {
		t.Fatalf("first /health = %d", code)
	}
	st.down.Store(true)
	// Within the probe window the cached healthy result stands: one ping per window.
	if code, _, _ := get(t, h, "/health", false); code != 200 {
		t.Fatalf("/health inside the probe window = %d, want the cached 200", code)
	}
}

func TestPublicStatsRefusesZerosWhenDatabaseDown(t *testing.T) {
	srv, st := newFlakyServer(t)
	h := srv.Handler()

	forget(srv)
	code, good, _ := get(t, h, "/v1/stats", false)
	if code != 200 || good["devices_known"] != float64(1) {
		t.Fatalf("/v1/stats with database up = %d %v, want 200 with devices_known 1", code, good)
	}

	st.down.Store(true)
	forget(srv)
	code, body, hdr := get(t, h, "/v1/stats", false)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("/v1/stats with database down = %d %v, want 503", code, body)
	}
	if _, has := body["devices_known"]; has {
		t.Errorf("/v1/stats published numbers while the database was down: %v", body)
	}
	if hdr.Get("Cache-Control") != "no-store" {
		t.Errorf("503 Cache-Control = %q, want no-store", hdr.Get("Cache-Control"))
	}

	// The failure must not have been cached: recovery serves real numbers at once.
	st.down.Store(false)
	publicStatsCache.mu.Lock()
	cachedAt := publicStatsCache.at
	publicStatsCache.mu.Unlock()
	if !cachedAt.IsZero() {
		t.Fatalf("a failed stats read was cached")
	}
	if code, body, _ := get(t, h, "/v1/stats", false); code != 200 || body["devices_known"] != float64(1) {
		t.Fatalf("/v1/stats after recovery = %d %v", code, body)
	}
}

func TestAdminOverviewMarksDegradedReads(t *testing.T) {
	srv, st := newFlakyServer(t)
	h := srv.Handler()

	code, body, _ := get(t, h, "/admin/overview", true)
	if code != 200 || body["degraded"] != false {
		t.Fatalf("/admin/overview with database up = %d degraded=%v, want 200 false", code, body["degraded"])
	}
	if errs, _ := body["errors"].([]any); len(errs) != 0 {
		t.Fatalf("errors with database up = %v, want []", errs)
	}

	st.down.Store(true)
	code, body, _ = get(t, h, "/admin/overview", true)
	if code != 200 || body["degraded"] != true {
		t.Fatalf("/admin/overview with database down = %d degraded=%v, want 200 true", code, body["degraded"])
	}
	got := map[string]bool{}
	errs, _ := body["errors"].([]any)
	for _, e := range errs {
		got[e.(string)] = true
	}
	for _, want := range []string{"node_capabilities", "usage_24h", "earnings_7d", "waitlist"} {
		if !got[want] {
			t.Errorf("errors = %v, missing %q", errs, want)
		}
	}
}
