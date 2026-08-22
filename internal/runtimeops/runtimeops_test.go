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

func TestPipelineRestoreOwnsSnapshot(t *testing.T) {
	ctx := context.Background()
	var ran []string
	step := func(name string) Step {
		return func(context.Context) error { ran = append(ran, name); return nil }
	}

	// Snapshot with a completed step and an unfinished (false) step, then hand
	// it to a fresh pipeline.
	snapshot := map[string]bool{"prepare": true, "depart": false}
	p := NewPipeline()
	p.Restore(snapshot)

	// Caller keeps using the input map — setting "depart" to true externally
	// must not mark it as completed inside the pipeline, otherwise Run skips it.
	snapshot["depart"] = true
	snapshot["prepare"] = false // even mutating an already-true entry must not leak

	steps := map[string]Step{"prepare": step("prepare"), "depart": step("depart")}
	if err := p.Run(ctx, steps, []string{"prepare", "depart"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(ran) != 1 || ran[0] != "depart" {
		t.Fatalf("expected only depart to run, got %v", ran)
	}
	// "prepare" stays restored-as-done; "depart" was genuinely executed.
	got := p.Snapshot()
	if !got["prepare"] || !got["depart"] {
		t.Fatalf("snapshot=%v want prepare=true depart=true", got)
	}
}

func TestPipelineRestoreNilAndFalseSemantics(t *testing.T) {
	ctx := context.Background()
	var ran []string
	step := func(name string) Step {
		return func(context.Context) error { ran = append(ran, name); return nil }
	}

	p := NewPipeline()
	p.Restore(nil)

	snap := p.Snapshot()
	if len(snap) != 0 {
		t.Fatalf("expected empty snapshot after nil restore, got %v", snap)
	}

	// A false entry must not be treated as completed: the step still runs.
	p.Restore(map[string]bool{"depart": false})
	if err := p.Run(ctx, map[string]Step{"depart": step("depart")}, []string{"depart"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(ran) != 1 || ran[0] != "depart" {
		t.Fatalf("expected depart to run, got %v", ran)
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
