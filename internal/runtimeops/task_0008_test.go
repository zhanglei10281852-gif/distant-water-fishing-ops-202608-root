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

func TestTask0008RejectedClosureKeepsBatchCollecting(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	must(t, s.CreateBatch(ctx, "fleet-g", "batch-g", "collecting"))
	must(t, s.RecordEvent(ctx, Event{ID: "event-open", TenantID: "fleet-g", BatchID: "batch-g", Status: "unclassified", Magnitude: 6}))
	if err := s.CloseBatch(ctx, "fleet-g", "batch-g"); !IsConflict(err) {
		t.Errorf("closure error=%v", err)
	}
	state, count, err := s.BatchState(ctx, "fleet-g", "batch-g")
	if err != nil || state != "collecting" || count != 1 {
		t.Errorf("rejected closure state=%s count=%d err=%v", state, count, err)
	}
	must(t, s.ClassifyEvent(ctx, "fleet-g", "event-open", "classified"))
	must(t, s.CloseBatch(ctx, "fleet-g", "batch-g"))
	state, count, err = s.BatchState(ctx, "fleet-g", "batch-g")
	if err != nil || state != "closed" || count != 1 {
		t.Errorf("valid closure state=%s count=%d err=%v", state, count, err)
	}
}
