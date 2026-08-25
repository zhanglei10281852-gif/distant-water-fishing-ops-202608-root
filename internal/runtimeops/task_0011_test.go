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

func TestTask0011ForeignJobsCannotStarveFleetClaims(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	now := time.Date(2026, 8, 22, 2, 0, 0, 0, time.UTC)
	must(t, s.CreateJob(ctx, Job{ID: "foreign-first", TenantID: "fleet-b", State: "pending", Payload: "b", MaxAttempts: 3, AvailableAt: now.Add(-time.Hour)}))
	must(t, s.CreateJob(ctx, Job{ID: "local-ready", TenantID: "fleet-a", State: "pending", Payload: "a", MaxAttempts: 3, AvailableAt: now.Add(-time.Minute)}))
	jobs, err := s.ClaimJobs(ctx, "fleet-a", now, 1, time.Minute)
	if err != nil || len(jobs) != 1 || jobs[0].ID != "local-ready" || jobs[0].TenantID != "fleet-a" {
		t.Errorf("fleet-a claim=%+v err=%v", jobs, err)
	}
	foreign, err := s.Job(ctx, "fleet-b", "foreign-first")
	if err != nil || foreign.State != "pending" || foreign.Attempts != 0 {
		t.Errorf("foreign job=%+v err=%v", foreign, err)
	}
	must(t, s.CompleteJob(ctx, "fleet-a", "local-ready", 1))
}
