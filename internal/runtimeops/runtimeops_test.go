package runtimeops

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "runtime.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestPermitHappyPath(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.CreatePermit(ctx, Permit{ID: "p", TenantID: "t", Slot: "s", State: "pending", Version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.ApprovePermit(ctx, "t", "p", 1); err != nil {
		t.Fatal(err)
	}
	if err := s.ExecutePermit(ctx, "t", "p", 2); err != nil {
		t.Fatal(err)
	}
	p, err := s.Permit(ctx, "t", "p")
	if err != nil || p.State != "executing" {
		t.Fatalf("permit=%+v err=%v", p, err)
	}
}

func TestEventHappyPath(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.CreateBatch(ctx, "t", "b", "collecting"); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordEvent(ctx, Event{ID: "e", TenantID: "t", BatchID: "b", Status: "unclassified", Magnitude: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.ClassifyEvent(ctx, "t", "e", "classified"); err != nil {
		t.Fatal(err)
	}
	if err := s.CloseBatch(ctx, "t", "b"); err != nil {
		t.Fatal(err)
	}
}

func TestJobHappyPath(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()
	if err := s.CreateJob(ctx, Job{ID: "j", TenantID: "t", State: "pending", Payload: "x", MaxAttempts: 3, AvailableAt: now}); err != nil {
		t.Fatal(err)
	}
	jobs, err := s.ClaimJobs(ctx, "t", now, 1, time.Minute)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("jobs=%+v err=%v", jobs, err)
	}
	if err = s.CompleteJob(ctx, "t", "j", 1); err != nil {
		t.Fatal(err)
	}
}

func TestCommandHappyPath(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.SaveCommand(ctx, Command{TenantID: "t", Method: "POST", Path: "/permits", Key: "k", RequestHash: "h", Response: []byte("ok")}); err != nil {
		t.Fatal(err)
	}
	body, err := s.ReplayCommand(ctx, "t", "POST", "/permits", "k", "h")
	if err != nil || string(body) != "ok" {
		t.Fatalf("body=%q err=%v", body, err)
	}
}

func TestPermitCancelVersionGuard(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.CreatePermit(ctx, Permit{ID: "p", TenantID: "t", Slot: "s", State: "approved", Version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.CancelPermit(ctx, "t", "p", 99); !IsConflict(err) {
		t.Fatalf("expected conflict on stale version, got %v", err)
	}
	p, err := s.Permit(ctx, "t", "p")
	if err != nil || p.State != "approved" || p.Version != 1 {
		t.Fatalf("state must be unchanged on guard failure: %+v err=%v", p, err)
	}
	if n, _ := s.OutboxCount(ctx, "t", "p"); n != 0 {
		t.Fatalf("no notice must be written on guard failure: outbox=%d", n)
	}
}

func TestPermitCancelAtomicNotice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.CreatePermit(ctx, Permit{ID: "p", TenantID: "t", Slot: "s", State: "approved", Version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.CancelPermit(ctx, "t", "p", 1); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	p, err := s.Permit(ctx, "t", "p")
	if err != nil || p.State != "cancelled" || p.Version != 2 {
		t.Fatalf("expected cancelled v2: %+v err=%v", p, err)
	}
	if n, _ := s.OutboxCount(ctx, "t", "p"); n != 1 {
		t.Fatalf("exactly one notice must accompany cancellation: outbox=%d", n)
	}
	if err := s.CancelPermit(ctx, "t", "p", 2); !IsConflict(err) {
		t.Fatalf("cancelling an already-cancelled permit must conflict, got %v", err)
	}
	if n, _ := s.OutboxCount(ctx, "t", "p"); n != 1 {
		t.Fatalf("duplicate cancellation must not add notices: outbox=%d", n)
	}
}
