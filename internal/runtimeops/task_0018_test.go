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

func TestTask0018LeaseSweepIsScopedToFleet(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	now := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	for _, item := range []struct{ id, tenant string }{{"job-a", "fleet-a"}, {"job-b", "fleet-b"}} {
		must(t, s.CreateJob(ctx, Job{ID: item.id, TenantID: item.tenant, State: "pending", Payload: "x", MaxAttempts: 3, AvailableAt: now}))
		jobs, err := s.ClaimJobs(ctx, item.tenant, now, 1, time.Minute)
		if err != nil || len(jobs) != 1 {
			t.Fatalf("claim %s=%+v err=%v", item.id, jobs, err)
		}
	}
	count, err := s.ReclaimJobs(ctx, "fleet-a", now.Add(2*time.Minute))
	if err != nil || count != 1 {
		t.Errorf("fleet-a reclaim count=%d err=%v", count, err)
	}
	a, err := s.Job(ctx, "fleet-a", "job-a")
	if err != nil || a.State != "failed" {
		t.Errorf("job-a=%+v err=%v", a, err)
	}
	b, err := s.Job(ctx, "fleet-b", "job-b")
	if err != nil || b.State != "running" || b.LeaseUntil == nil {
		t.Errorf("job-b=%+v err=%v", b, err)
	}
	count, err = s.ReclaimJobs(ctx, "fleet-b", now.Add(2*time.Minute))
	if err != nil || count != 1 {
		t.Errorf("fleet-b reclaim count=%d err=%v", count, err)
	}
}
