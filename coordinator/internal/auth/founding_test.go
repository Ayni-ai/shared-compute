package auth

import (
	"context"
	"testing"
	"time"
)

func foundingFixture(t *testing.T) (*Auth, context.Context) {
	t.Helper()
	a, _ := testAuth(t)
	ctx := context.Background()
	for _, tbl := range []string{"feedback_replies", "feedback", "fleet_samples"} {
		if _, err := a.pool.Exec(ctx, "DELETE FROM "+tbl); err != nil {
			t.Fatalf("clean %s: %v", tbl, err)
		}
	}
	return a, ctx
}

// member creates an account; device, job, minutes and feedback set what the
// coordinator would have observed for it.
func member(t *testing.T, a *Auth, ctx context.Context, name string, job bool, minutes int, feedback bool) User {
	t.Helper()
	u, err := a.upsertUser(ctx, "github", "gh-"+name, name+"@example.com", name, "")
	if err != nil {
		t.Fatal(err)
	}
	pk := "pk-" + name
	if _, err := a.pool.Exec(ctx, `INSERT INTO provider_owners (static_pk, user_id, online_minutes) VALUES ($1,$2,$3)`,
		pk, u.ID, minutes); err != nil {
		t.Fatal(err)
	}
	if job {
		if _, err := a.pool.Exec(ctx, `INSERT INTO provider_earnings (job_id, static_pk, provider_micros) VALUES ($1,$2,10)`,
			"job-"+name, pk); err != nil {
			t.Fatal(err)
		}
	}
	if feedback {
		if _, err := a.pool.Exec(ctx, `INSERT INTO feedback (kind, message, user_id) VALUES ('idea', 'pairing was easy', $1)`, u.ID); err != nil {
			t.Fatal(err)
		}
	}
	return u
}

func TestFoundingValidationNeedsAllThreeAndNumbersInEnrollmentOrder(t *testing.T) {
	a, ctx := foundingFixture(t)
	rules := FoundingRules{MinOnlineMinutes: 24 * 60, Cap: 1000}

	full1 := member(t, a, ctx, "ana", true, 1500, true)
	noJob := member(t, a, ctx, "ben", false, 1500, true)
	short := member(t, a, ctx, "cai", true, 600, true)
	silent := member(t, a, ctx, "dee", true, 1500, false)
	notEnrolled := member(t, a, ctx, "eve", true, 1500, true)
	full2 := member(t, a, ctx, "fay", true, 2000, true)

	// Enrollment order decides numbers: fay enrolls before ana.
	for _, u := range []User{full2, full1, noJob, short, silent} {
		if err := a.EnrollFounding(ctx, u.ID); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond)
	}
	// Enrolling again keeps the first time.
	before, _ := a.FoundingStatusFor(ctx, full2.ID)
	_ = a.EnrollFounding(ctx, full2.ID)
	after, _ := a.FoundingStatusFor(ctx, full2.ID)
	if !before.EnrolledAt.Equal(*after.EnrolledAt) {
		t.Errorf("re-enrolling moved enrolled_at")
	}

	n, err := a.ValidateFoundingMembers(ctx, rules)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("validated %d, want 2 (only ana and fay meet all three)", n)
	}
	want := map[string]int{full2.ID: 1, full1.ID: 2, noJob.ID: 0, short.ID: 0, silent.ID: 0, notEnrolled.ID: 0}
	for id, num := range want {
		st, err := a.FoundingStatusFor(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if st.Number != num {
			t.Errorf("user %s number = %d, want %d (status %+v)", id, st.Number, num, st)
		}
	}

	// Short-on-hours member catches up; the next sweep gives them #3 and keeps #1-2.
	if _, err := a.pool.Exec(ctx, `UPDATE provider_owners SET online_minutes = 1440 WHERE user_id = $1`, short.ID); err != nil {
		t.Fatal(err)
	}
	if n, _ := a.ValidateFoundingMembers(ctx, rules); n != 1 {
		t.Fatalf("second sweep validated %d, want 1", n)
	}
	if st, _ := a.FoundingStatusFor(ctx, short.ID); st.Number != 3 || st.ValidatedAt == nil {
		t.Errorf("caught-up member = %+v, want #3 validated", st)
	}
	if n, _ := a.ValidateFoundingMembers(ctx, rules); n != 0 {
		t.Errorf("idempotent sweep validated %d more", n)
	}

	f, err := a.FoundingFunnelCounts(ctx, rules)
	if err != nil {
		t.Fatal(err)
	}
	if f.Users != 6 || f.Enrolled != 5 || f.WithDevice != 5 || f.ServedJob != 4 || f.OnlineEnough != 5 || f.GaveFeedback != 4 || f.Validated != 3 {
		t.Errorf("funnel = %+v", f)
	}
	if v, e, _ := a.FoundingCounts(ctx); v != 3 || e != 5 {
		t.Errorf("counts = %d validated, %d enrolled; want 3, 5", v, e)
	}
}

func TestFoundingCapStopsNumbering(t *testing.T) {
	a, ctx := foundingFixture(t)
	for _, name := range []string{"gus", "hal", "ivy"} {
		u := member(t, a, ctx, name, true, 2000, true)
		if err := a.EnrollFounding(ctx, u.ID); err != nil {
			t.Fatal(err)
		}
	}
	rules := FoundingRules{MinOnlineMinutes: 60, Cap: 2}
	if n, _ := a.ValidateFoundingMembers(ctx, rules); n != 2 {
		t.Fatalf("validated %d with cap 2", n)
	}
	if n, _ := a.ValidateFoundingMembers(ctx, rules); n != 0 {
		t.Fatalf("validated %d past the cap", n)
	}
}

func TestMemberByCodeShowsNameOnlyWhenOptedIn(t *testing.T) {
	a, ctx := foundingFixture(t)
	u := member(t, a, ctx, "jo", true, 2000, true)
	code, err := a.ReferralCode(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := a.MemberByCode(ctx, code); ok {
		t.Fatal("not-yet-validated member is public")
	}
	_ = a.EnrollFounding(ctx, u.ID)
	if _, err := a.ValidateFoundingMembers(ctx, FoundingRules{MinOnlineMinutes: 60, Cap: 1000}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetProfile(ctx, u.ID, Profile{DisplayName: "Jo R.", LeaderboardOptIn: false}); err != nil {
		t.Fatal(err)
	}
	m, ok, err := a.MemberByCode(ctx, code)
	if err != nil || !ok || m.Number != 1 || m.Name != "" {
		t.Fatalf("member = %+v ok=%v err=%v; want #1 with no name", m, ok, err)
	}
	_ = a.SetProfile(ctx, u.ID, Profile{DisplayName: "Jo R.", LeaderboardOptIn: true})
	if m, _, _ := a.MemberByCode(ctx, code); m.Name != "Jo R." {
		t.Errorf("opted-in name = %q", m.Name)
	}
	if _, ok, _ := a.MemberByCode(ctx, "nosuchcode"); ok {
		t.Error("unknown code resolved")
	}
}

func TestFleetSamplesCreditMinutesAndGiveAFloor(t *testing.T) {
	a, ctx := foundingFixture(t)
	u := member(t, a, ctx, "kai", false, 0, false)
	// 40 samples: 39 with 100 to 109 online, one dip to 3 (a deploy). The 5th
	// percentile ignores a single dip in forty.
	for i := 0; i < 40; i++ {
		online := []string{"pk-kai"}
		for j := 0; j < 99+i%10; j++ {
			online = append(online, "other")
		}
		if i == 7 {
			online = online[:3]
		}
		if err := a.RecordFleetSample(ctx, online, 1, len(online)-1, 5); err != nil {
			t.Fatal(err)
		}
	}
	if st, _ := a.FoundingStatusFor(ctx, u.ID); st.OnlineMinutes != 200 {
		t.Errorf("online minutes = %d, want 200 (40 samples x 5)", st.OnlineMinutes)
	}
	fw, err := a.FleetSince(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if fw.Samples != 40 || fw.Peak != 109 || fw.Floor != 100 {
		t.Errorf("fleet window = %+v, want 40 samples, peak 109, floor 100", fw)
	}
}
