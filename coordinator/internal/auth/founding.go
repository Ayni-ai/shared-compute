package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Founding members (docs/FOUNDING-MEMBERS.md, migrations/0013_founding_members.sql).
//
// A person enrolls in the test month. The coordinator then validates them from what
// it observes, never from what they claim: one of their devices served a job, their
// devices were connected for MinOnlineMinutes in total, and they sent feedback. Each
// validated member gets the next number, 1 to Cap, in validation order, and keeps it.

// FoundingRules are the validation thresholds.
type FoundingRules struct {
	MinOnlineMinutes int // summed over the account's devices
	Cap              int // numbered founding members; the program closes at this count
}

const foundingSweepLockKey = 0x41594e49464d // "AYNIFM": one sweep at a time across instances

// Per-user conditions, shared by the sweep, the member's checklist and the funnel so
// the three never disagree. Each expects the users row aliased as u.
const (
	sqlServedJobs = `(SELECT count(*) FROM provider_earnings e JOIN provider_owners o ON o.static_pk = e.static_pk
	                  WHERE o.user_id = u.id AND e.state <> 'reversed')`
	sqlOnlineMinutes = `(SELECT COALESCE(sum(o.online_minutes), 0) FROM provider_owners o WHERE o.user_id = u.id)`
	sqlFeedbackCount = `(SELECT count(*) FROM feedback f WHERE f.kind <> 'tester_request'
	                     AND (f.user_id = u.id OR (f.email <> '' AND lower(f.email) = lower(u.email))))`
	sqlDeviceCount = `(SELECT count(*) FROM provider_owners o WHERE o.user_id = u.id)`
)

// FoundingStatus is one account's progress, for its own page.
type FoundingStatus struct {
	EnrolledAt    *time.Time `json:"enrolled_at"`
	ValidatedAt   *time.Time `json:"validated_at"`
	Number        int        `json:"number"` // 0 until validated
	Devices       int        `json:"devices"`
	JobsServed    int        `json:"jobs_served"`
	OnlineMinutes int        `json:"online_minutes"`
	Feedback      int        `json:"feedback"`
}

// EnrollFounding records that the user accepted the test-month terms. Enrolling twice
// keeps the first time.
func (a *Auth) EnrollFounding(ctx context.Context, userID string) error {
	tag, err := a.pool.Exec(ctx,
		`UPDATE users SET founding_enrolled_at = COALESCE(founding_enrolled_at, now()) WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("enroll founding: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return errors.New("enroll founding: no such user")
	}
	return nil
}

// FoundingStatusFor returns the account's checklist.
func (a *Auth) FoundingStatusFor(ctx context.Context, userID string) (FoundingStatus, error) {
	var s FoundingStatus
	var num *int
	err := a.pool.QueryRow(ctx, `
		SELECT u.founding_enrolled_at, u.founding_validated_at, u.founding_number,
		       `+sqlDeviceCount+`, `+sqlServedJobs+`, `+sqlOnlineMinutes+`, `+sqlFeedbackCount+`
		FROM users u WHERE u.id = $1`, userID).
		Scan(&s.EnrolledAt, &s.ValidatedAt, &num, &s.Devices, &s.JobsServed, &s.OnlineMinutes, &s.Feedback)
	if err != nil {
		return s, fmt.Errorf("founding status: %w", err)
	}
	if num != nil {
		s.Number = *num
	}
	return s, nil
}

// ValidateFoundingMembers numbers every enrolled account that now meets the rules, in
// enrollment order, until the cap. It returns how many were newly validated.
func (a *Auth) ValidateFoundingMembers(ctx context.Context, rules FoundingRules) (int, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("founding sweep: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(foundingSweepLockKey)); err != nil {
		return 0, fmt.Errorf("founding sweep lock: %w", err)
	}
	var base int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(max(founding_number), 0) FROM users`).Scan(&base); err != nil {
		return 0, fmt.Errorf("founding sweep base: %w", err)
	}
	remaining := rules.Cap - base
	if remaining <= 0 {
		return 0, nil
	}
	tag, err := tx.Exec(ctx, `
		WITH eligible AS (
			SELECT u.id, u.founding_enrolled_at FROM users u
			WHERE u.founding_enrolled_at IS NOT NULL AND u.founding_number IS NULL
			  AND `+sqlServedJobs+` > 0
			  AND `+sqlOnlineMinutes+` >= $1
			  AND `+sqlFeedbackCount+` > 0
			ORDER BY u.founding_enrolled_at, u.id
			LIMIT $2
		), numbered AS (
			SELECT id, row_number() OVER (ORDER BY founding_enrolled_at, id) AS rn FROM eligible
		)
		UPDATE users u SET founding_number = $3 + numbered.rn, founding_validated_at = now()
		FROM numbered WHERE u.id = numbered.id`,
		rules.MinOnlineMinutes, remaining, base)
	if err != nil {
		return 0, fmt.Errorf("founding sweep: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("founding sweep commit: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// PublicMember is what anyone holding a member's link may see: the number, the month
// it was earned and, only if the member opted in, their display name.
type PublicMember struct {
	Number      int       `json:"number"`
	Name        string    `json:"name"`
	ValidatedAt time.Time `json:"validated_at"`
}

// MemberByCode looks up a validated member by their invite code. ok is false for an
// unknown code or a member who is not validated yet.
func (a *Auth) MemberByCode(ctx context.Context, code string) (PublicMember, bool, error) {
	var m PublicMember
	err := a.pool.QueryRow(ctx, `
		SELECT founding_number, founding_validated_at,
		       CASE WHEN leaderboard_opt_in THEN display_name ELSE '' END
		FROM users WHERE referral_code = $1 AND founding_number IS NOT NULL`, code).
		Scan(&m.Number, &m.ValidatedAt, &m.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, false, nil
	}
	if err != nil {
		return m, false, fmt.Errorf("member by code: %w", err)
	}
	return m, true, nil
}

// FoundingFunnel counts each step of the program, for the operator.
type FoundingFunnel struct {
	Users            int `json:"users"`
	Enrolled         int `json:"enrolled"`
	WithDevice       int `json:"with_device"`
	ServedJob        int `json:"served_job"`
	OnlineEnough     int `json:"online_enough"`
	GaveFeedback     int `json:"gave_feedback"`
	Validated        int `json:"validated"`
	Enrolled7d       int `json:"enrolled_7d"`
	Validated7d      int `json:"validated_7d"`
	ReferredEnrolled int `json:"referred_enrolled"` // enrolled members who arrived through someone's link
}

func (a *Auth) FoundingFunnelCounts(ctx context.Context, rules FoundingRules) (FoundingFunnel, error) {
	var f FoundingFunnel
	err := a.pool.QueryRow(ctx, `
		WITH m AS (
			SELECT u.founding_enrolled_at, u.founding_validated_at, u.founding_number, u.referred_by,
			       `+sqlDeviceCount+` AS devices, `+sqlServedJobs+` AS jobs,
			       `+sqlOnlineMinutes+` AS mins, `+sqlFeedbackCount+` AS fb
			FROM users u WHERE u.founding_enrolled_at IS NOT NULL
		)
		SELECT (SELECT count(*) FROM users), count(*),
		       count(*) FILTER (WHERE devices > 0), count(*) FILTER (WHERE jobs > 0),
		       count(*) FILTER (WHERE mins >= $1), count(*) FILTER (WHERE fb > 0),
		       count(*) FILTER (WHERE founding_number IS NOT NULL),
		       count(*) FILTER (WHERE founding_enrolled_at >= now() - interval '7 days'),
		       count(*) FILTER (WHERE founding_validated_at >= now() - interval '7 days'),
		       count(*) FILTER (WHERE referred_by <> '')
		FROM m`, rules.MinOnlineMinutes).
		Scan(&f.Users, &f.Enrolled, &f.WithDevice, &f.ServedJob, &f.OnlineEnough, &f.GaveFeedback,
			&f.Validated, &f.Enrolled7d, &f.Validated7d, &f.ReferredEnrolled)
	if err != nil {
		return f, fmt.Errorf("founding funnel: %w", err)
	}
	return f, nil
}

// FoundingCounts is the public pair: validated members and people in the test month.
func (a *Auth) FoundingCounts(ctx context.Context) (validated, enrolled int, err error) {
	err = a.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE founding_number IS NOT NULL), count(*) FILTER (WHERE founding_enrolled_at IS NOT NULL)
		FROM users`).Scan(&validated, &enrolled)
	if err != nil {
		err = fmt.Errorf("founding counts: %w", err)
	}
	return
}

// RecordFleetSample stores one fleet-size sample and credits every connected device
// with the minutes since the previous sample. Samples older than 90 days are pruned.
func (a *Auth) RecordFleetSample(ctx context.Context, onlinePKs []string, phones, pcs, minutes int) error {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("fleet sample: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := tx.Exec(ctx, `INSERT INTO fleet_samples (online, phones, pcs) VALUES ($1, $2, $3)`,
		len(onlinePKs), phones, pcs); err != nil {
		return fmt.Errorf("fleet sample insert: %w", err)
	}
	if len(onlinePKs) > 0 && minutes > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE provider_owners SET online_minutes = online_minutes + $2, last_seen = now()
			WHERE static_pk = ANY($1)`, onlinePKs, minutes); err != nil {
			return fmt.Errorf("fleet sample minutes: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM fleet_samples WHERE at < now() - interval '90 days'`); err != nil {
		return fmt.Errorf("fleet sample prune: %w", err)
	}
	return tx.Commit(ctx)
}

// FleetWindow summarizes the samples since a time. Floor is the 5th percentile: the
// fleet was at least this large 95% of the time.
type FleetWindow struct {
	Samples     int     `json:"samples"`
	Floor       int     `json:"floor"`
	Median      int     `json:"median"`
	Peak        int     `json:"peak"`
	CoveredHrs  float64 `json:"covered_hours"` // first sample in the window to now
	WindowHours float64 `json:"window_hours"`
}

func (a *Auth) FleetSince(ctx context.Context, since time.Time) (FleetWindow, error) {
	w := FleetWindow{WindowHours: time.Since(since).Hours()}
	var first *time.Time
	err := a.pool.QueryRow(ctx, `
		SELECT count(*),
		       COALESCE(percentile_disc(0.05) WITHIN GROUP (ORDER BY online), 0),
		       COALESCE(percentile_disc(0.5) WITHIN GROUP (ORDER BY online), 0),
		       COALESCE(max(online), 0), min(at)
		FROM fleet_samples WHERE at >= $1`, since).
		Scan(&w.Samples, &w.Floor, &w.Median, &w.Peak, &first)
	if err != nil {
		return w, fmt.Errorf("fleet window: %w", err)
	}
	if first != nil {
		w.CoveredHrs = float64(int(time.Since(*first).Hours()*10)) / 10
	}
	w.WindowHours = float64(int(w.WindowHours*10)) / 10
	return w, nil
}
