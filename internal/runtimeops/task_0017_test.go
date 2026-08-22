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

func TestTask0017CancelledLeaseSweepMakesNoChanges(t *testing.T) {
	base := context.Background()
	s := testStore(t)
	now := time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)
	must(t, s.CreateJob(base, Job{ID: "sweep", TenantID: "fleet-q", State: "pending", Payload: "x", MaxAttempts: 3, AvailableAt: now}))
	_, err := s.ClaimJobs(base, "fleet-q", now, 1, time.Minute)
	must(t, err)
	ctx, cancel := context.WithCancel(base)
	cancel()
	count, err := s.ReclaimJobs(ctx, "fleet-q", now.Add(2*time.Minute))
	if !errors.Is(err, context.Canceled) || count != 0 {
		t.Errorf("cancelled reclaim count=%d err=%v", count, err)
	}
	job, err := s.Job(base, "fleet-q", "sweep")
	if err != nil || job.State != "running" || job.LeaseUntil == nil {
		t.Errorf("job after cancelled sweep=%+v err=%v", job, err)
	}
	count, err = s.ReclaimJobs(base, "fleet-q", now.Add(2*time.Minute))
	if err != nil || count != 1 {
		t.Errorf("valid reclaim count=%d err=%v", count, err)
	}
	job, err = s.Job(base, "fleet-q", "sweep")
	if err != nil || job.State != "failed" {
		t.Errorf("reclaimed job=%+v err=%v", job, err)
	}
}
