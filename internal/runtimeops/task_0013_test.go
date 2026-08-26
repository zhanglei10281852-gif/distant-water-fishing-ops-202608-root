package runtimeops

import (
	"context"
	"errors"
	"testing"
	"time"
)

var (
	_ = errors.Is
	_ = time.Second
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestTask0013PartialClaimFailureRollsBackEarlierJobs(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	now := time.Date(2026, 8, 22, 4, 0, 0, 0, time.UTC)
	must(t, s.CreateJob(ctx, Job{ID: "job-1", TenantID: "fleet-m", State: "pending", Payload: "one", MaxAttempts: 3, AvailableAt: now}))
	must(t, s.CreateJob(ctx, Job{ID: "job-2", TenantID: "fleet-m", State: "pending", Payload: "two", MaxAttempts: 3, AvailableAt: now}))
	_, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_second_claim BEFORE UPDATE ON jobs WHEN NEW.id='job-2' AND NEW.state='running' BEGIN SELECT RAISE(ABORT, 'lease store offline'); END`)
	must(t, err)
	if _, err = s.ClaimJobs(ctx, "fleet-m", now, 2, time.Minute); err == nil {
		t.Error("claim unexpectedly succeeded")
	}
	for _, id := range []string{"job-1", "job-2"} {
		job, readErr := s.Job(ctx, "fleet-m", id)
		if readErr != nil || job.State != "pending" || job.Attempts != 0 || job.LeaseUntil != nil {
			t.Errorf("%s after failed claim=%+v err=%v", id, job, readErr)
		}
	}
	_, err = s.db.ExecContext(ctx, `DROP TRIGGER reject_second_claim`)
	must(t, err)
	jobs, err := s.ClaimJobs(ctx, "fleet-m", now, 2, time.Minute)
	if err != nil || len(jobs) != 2 {
		t.Errorf("valid claim=%+v err=%v", jobs, err)
	}
}
