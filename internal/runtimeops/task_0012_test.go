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

func TestTask0012UnownedJobIsNotReturnedToWorker(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	now := time.Date(2026, 8, 22, 3, 0, 0, 0, time.UTC)
	must(t, s.CreateJob(ctx, Job{ID: "lease-job", TenantID: "fleet-l", State: "pending", Payload: "telemetry", MaxAttempts: 4, AvailableAt: now}))
	_, err := s.db.ExecContext(ctx, `CREATE TRIGGER ignore_lease_claim BEFORE UPDATE ON jobs WHEN NEW.id='lease-job' AND NEW.state='running' BEGIN SELECT RAISE(IGNORE); END`)
	must(t, err)
	jobs, err := s.ClaimJobs(ctx, "fleet-l", now, 1, 5*time.Minute)
	if err != nil || len(jobs) != 0 {
		t.Errorf("claim without ownership=%+v err=%v", jobs, err)
	}
	stored, err := s.Job(ctx, "fleet-l", "lease-job")
	if err != nil || stored.State != "pending" || stored.Attempts != 0 || stored.LeaseUntil != nil {
		t.Errorf("unowned stored job=%+v err=%v", stored, err)
	}
	_, err = s.db.ExecContext(ctx, `DROP TRIGGER ignore_lease_claim`)
	must(t, err)
	jobs, err = s.ClaimJobs(ctx, "fleet-l", now, 1, 5*time.Minute)
	if err != nil || len(jobs) != 1 || jobs[0].State != "running" || jobs[0].Attempts != 1 {
		t.Errorf("valid claim=%+v err=%v", jobs, err)
	}
	must(t, s.CompleteJob(ctx, "fleet-l", "lease-job", 1))
	stored, err = s.Job(ctx, "fleet-l", "lease-job")
	if err != nil || stored.State != "done" {
		t.Errorf("completed job=%+v err=%v", stored, err)
	}
}
