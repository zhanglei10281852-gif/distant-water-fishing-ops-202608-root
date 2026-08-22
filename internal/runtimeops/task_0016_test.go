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

func TestTask0016CancelledFailureReportCannotMutateJob(t *testing.T) {
	base := context.Background()
	s := testStore(t)
	now := time.Date(2026, 8, 22, 7, 0, 0, 0, time.UTC)
	must(t, s.CreateJob(base, Job{ID: "cancel-fail", TenantID: "fleet-p", State: "pending", Payload: "x", MaxAttempts: 3, AvailableAt: now}))
	_, err := s.ClaimJobs(base, "fleet-p", now, 1, time.Minute)
	must(t, err)
	ctx, cancel := context.WithCancel(base)
	cancel()
	if err = s.FailJob(ctx, "fleet-p", "cancel-fail", 1, now); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled failure error=%v", err)
	}
	job, err := s.Job(base, "fleet-p", "cancel-fail")
	if err != nil || job.State != "running" || job.LeaseUntil == nil {
		t.Errorf("job after cancellation=%+v err=%v", job, err)
	}
	must(t, s.FailJob(base, "fleet-p", "cancel-fail", 1, now))
	job, err = s.Job(base, "fleet-p", "cancel-fail")
	if err != nil || job.State != "failed" || job.LeaseUntil != nil {
		t.Errorf("valid failure=%+v err=%v", job, err)
	}
}
