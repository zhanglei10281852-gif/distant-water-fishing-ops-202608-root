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

func TestCommandTenantIsolation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	fleetA := Command{TenantID: "fleet-a", Method: "POST", Path: "/missions", Key: "shared-key", RequestHash: "hash-a", Response: []byte("resp-a")}
	fleetB := Command{TenantID: "fleet-b", Method: "POST", Path: "/missions", Key: "shared-key", RequestHash: "hash-b", Response: []byte("resp-b")}
	if err := s.SaveCommand(ctx, fleetA); err != nil {
		t.Fatalf("save fleet-a: %v", err)
	}
	// Same idempotency key, different tenant and payload must not collide with fleet A.
	if err := s.SaveCommand(ctx, fleetB); err != nil {
		t.Fatalf("save fleet-b: %v", err)
	}
	bodyA, err := s.ReplayCommand(ctx, fleetA.TenantID, fleetA.Method, fleetA.Path, fleetA.Key, fleetA.RequestHash)
	if err != nil || string(bodyA) != "resp-a" {
		t.Fatalf("fleet-a replay body=%q err=%v", bodyA, err)
	}
	bodyB, err := s.ReplayCommand(ctx, fleetB.TenantID, fleetB.Method, fleetB.Path, fleetB.Key, fleetB.RequestHash)
	if err != nil || string(bodyB) != "resp-b" {
		t.Fatalf("fleet-b replay body=%q err=%v", bodyB, err)
	}
	// Replaying fleet A's record with fleet B's hash must still be a payload conflict within fleet A.
	if _, err := s.ReplayCommand(ctx, fleetA.TenantID, fleetA.Method, fleetA.Path, fleetA.Key, fleetB.RequestHash); err != ErrConflict {
		t.Fatalf("fleet-a cross-payload replay err=%v want ErrConflict", err)
	}
}

func TestCommandSameTenantDifferentPayloadConflicts(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	original := Command{TenantID: "fleet-a", Method: "POST", Path: "/missions", Key: "shared-key", RequestHash: "hash-a", Response: []byte("resp-a")}
	if err := s.SaveCommand(ctx, original); err != nil {
		t.Fatalf("save original: %v", err)
	}
	// Same tenant, same operation, same key, but a different payload must be rejected.
	diverged := original
	diverged.RequestHash = "hash-a-prime"
	diverged.Response = []byte("resp-prime")
	if err := s.SaveCommand(ctx, diverged); err != ErrConflict {
		t.Fatalf("diverged payload err=%v want ErrConflict", err)
	}
	// The original response is untouched.
	body, err := s.ReplayCommand(ctx, original.TenantID, original.Method, original.Path, original.Key, original.RequestHash)
	if err != nil || string(body) != "resp-a" {
		t.Fatalf("replay body=%q err=%v", body, err)
	}
}
