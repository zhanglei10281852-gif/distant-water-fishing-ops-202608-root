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

func TestTask0007ForeignTelemetryDoesNotBlockBatchClosure(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	must(t, s.CreateBatch(ctx, "fleet-a", "batch-a", "collecting"))
	must(t, s.CreateBatch(ctx, "fleet-b", "batch-b", "collecting"))
	must(t, s.RecordEvent(ctx, Event{ID: "foreign-open", TenantID: "fleet-b", BatchID: "batch-b", Status: "unclassified", Magnitude: 8}))
	if err := s.CloseBatch(ctx, "fleet-a", "batch-a"); err != nil {
		t.Errorf("unrelated telemetry blocked closure: %v", err)
	}
	state, _, err := s.BatchState(ctx, "fleet-a", "batch-a")
	if err != nil || state != "closed" {
		t.Errorf("fleet-a batch state=%s err=%v", state, err)
	}
	if err = s.CloseBatch(ctx, "fleet-b", "batch-b"); !IsConflict(err) {
		t.Errorf("local unresolved telemetry did not block closure: %v", err)
	}
	state, _, err = s.BatchState(ctx, "fleet-b", "batch-b")
	if err != nil || state != "collecting" {
		t.Errorf("fleet-b batch state=%s err=%v", state, err)
	}
	must(t, s.ClassifyEvent(ctx, "fleet-b", "foreign-open", "classified"))
	must(t, s.CloseBatch(ctx, "fleet-b", "batch-b"))
}
