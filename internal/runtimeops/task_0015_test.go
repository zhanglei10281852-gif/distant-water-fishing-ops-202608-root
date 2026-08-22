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

func TestTask0015ExpiredWorkerCannotCompleteNewLease(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	now := time.Date(2026, 8, 22, 6, 0, 0, 0, time.UTC)
	must(t, s.CreateJob(ctx, Job{ID: "leased", TenantID: "fleet-o", State: "pending", Payload: "sample", MaxAttempts: 4, AvailableAt: now}))
	first, err := s.ClaimJobs(ctx, "fleet-o", now, 1, time.Minute)
	if err != nil || len(first) != 1 || first[0].Attempts != 1 {
		t.Fatalf("first claim=%+v err=%v", first, err)
	}
	count, err := s.ReclaimJobs(ctx, "fleet-o", now.Add(2*time.Minute))
	if err != nil || count != 1 {
		t.Fatalf("reclaim=%d err=%v", count, err)
	}
	second, err := s.ClaimJobs(ctx, "fleet-o", now.Add(2*time.Minute), 1, time.Minute)
	if err != nil || len(second) != 1 || second[0].Attempts != 2 {
		t.Fatalf("second claim=%+v err=%v", second, err)
	}
	if err = s.CompleteJob(ctx, "fleet-o", "leased", 1); !IsConflict(err) {
		t.Errorf("stale completion error=%v", err)
	}
	job, err := s.Job(ctx, "fleet-o", "leased")
	if err != nil || job.State != "running" || job.Attempts != 2 {
		t.Errorf("job after stale worker=%+v err=%v", job, err)
	}
	must(t, s.CompleteJob(ctx, "fleet-o", "leased", 2))
	job, err = s.Job(ctx, "fleet-o", "leased")
	if err != nil || job.State != "done" {
		t.Errorf("current worker completion=%+v err=%v", job, err)
	}
}
