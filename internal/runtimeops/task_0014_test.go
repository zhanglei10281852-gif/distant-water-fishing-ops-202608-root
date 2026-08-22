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

func TestTask0014CompletionFailureKeepsJobRunning(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	now := time.Date(2026, 8, 22, 5, 0, 0, 0, time.UTC)
	must(t, s.CreateJob(ctx, Job{ID: "complete-a", TenantID: "fleet-n", State: "pending", Payload: "result", MaxAttempts: 3, AvailableAt: now}))
	_, err := s.ClaimJobs(ctx, "fleet-n", now, 1, time.Minute)
	must(t, err)
	_, err = s.db.ExecContext(ctx, `CREATE TRIGGER reject_done_notice BEFORE INSERT ON outbox BEGIN SELECT RAISE(ABORT, 'outbox offline'); END`)
	must(t, err)
	if err = s.CompleteJob(ctx, "fleet-n", "complete-a", 1); err == nil {
		t.Error("completion unexpectedly succeeded")
	}
	job, err := s.Job(ctx, "fleet-n", "complete-a")
	if err != nil || job.State != "running" || job.Attempts != 1 || job.LeaseUntil == nil {
		t.Errorf("job after failure=%+v err=%v", job, err)
	}
	count, err := s.OutboxCount(ctx, "fleet-n", "complete-a")
	if err != nil || count != 0 {
		t.Errorf("outbox=%d err=%v", count, err)
	}
	_, err = s.db.ExecContext(ctx, `DROP TRIGGER reject_done_notice`)
	must(t, err)
	must(t, s.CompleteJob(ctx, "fleet-n", "complete-a", 1))
	job, err = s.Job(ctx, "fleet-n", "complete-a")
	if err != nil || job.State != "done" {
		t.Errorf("valid completion=%+v err=%v", job, err)
	}
}
