package api

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

// errDBUnavailable is returned by handlers that cannot answer without the database.
var errDBUnavailable = errors.New("database unavailable")

// dbProbe caches the result of a database ping for a few seconds, so health checks
// and public pages polling every few seconds cost at most one ping per window.
type dbProbe struct {
	mu   sync.Mutex
	at   time.Time
	err  error
	down bool // last logged state, to log transitions once instead of every probe
}

const (
	dbProbeTTL     = 5 * time.Second
	dbProbeTimeout = 2 * time.Second
)

// dbErr reports whether the store and the accounts database answer. nil means healthy
// (or no database configured: the memory store always answers).
func (s *Server) dbErr(ctx context.Context) error {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	if !s.db.at.IsZero() && time.Since(s.db.at) < dbProbeTTL {
		return s.db.err
	}
	c, cancel := context.WithTimeout(context.WithoutCancel(ctx), dbProbeTimeout)
	defer cancel()
	var err error
	if s.store != nil {
		err = s.store.Ping(c)
	}
	if err == nil && s.auth != nil {
		err = s.auth.Ping(c)
	}
	s.db.at, s.db.err = time.Now(), err
	if down := err != nil; down != s.db.down && s.log != nil {
		s.db.down = down
		if down {
			s.log.Error("database unreachable", "err", err)
		} else {
			s.log.Info("database reachable again")
		}
	}
	return err
}

// GET /health, /healthz: 200 when the coordinator and its database answer; 503 with
// "db":"down" when the database does not, so uptime checks and the status page see
// an outage instead of a healthy service that silently returns zeros.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	body := map[string]any{"status": "ok", "db": "ok", "providers": s.reg.Count()}
	if err := s.dbErr(r.Context()); err != nil {
		body["status"], body["db"] = "degraded", "down"
		writeJSON(w, http.StatusServiceUnavailable, body)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// GET /livez: the process is up and serving. No database check.
func (s *Server) handleLive(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "providers": s.reg.Count()})
}

// writeDBUnavailable answers 503 for a read that needed the database. Callers must
// not cache the response, and never substitute zeros for the missing numbers.
func writeDBUnavailable(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", "30")
	writeError(w, http.StatusServiceUnavailable, "database_unavailable",
		"the coordinator cannot reach its database right now; try again shortly")
}
