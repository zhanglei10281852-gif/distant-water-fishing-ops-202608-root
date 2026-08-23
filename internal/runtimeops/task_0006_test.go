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

func TestTask0006DuplicateTelemetryDoesNotPolluteBatch(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	must(t, s.CreateBatch(ctx, "fleet-f", "batch-f", "collecting"))
	first := Event{ID: "event-shared", TenantID: "fleet-f", BatchID: "batch-f", Status: "classified", Magnitude: 3.5}
	must(t, s.RecordEvent(ctx, first))
	if err := s.RecordEvent(ctx, first); err == nil {
		t.Error("duplicate event unexpectedly succeeded")
	}
	state, count, err := s.BatchState(ctx, "fleet-f", "batch-f")
	if err != nil || state != "collecting" || count != 1 {
		t.Errorf("batch after duplicate=%s count=%d err=%v", state, count, err)
	}
	page, err := s.ListEvents(ctx, "fleet-f", "", 1, 10)
	if err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Errorf("events after duplicate=%+v err=%v", page, err)
	}
	must(t, s.RecordEvent(ctx, Event{ID: "event-distinct", TenantID: "fleet-f", BatchID: "batch-f", Status: "classified", Magnitude: 4.8}))
	_, count, err = s.BatchState(ctx, "fleet-f", "batch-f")
	if err != nil || count != 2 {
		t.Errorf("valid second event count=%d err=%v", count, err)
	}
}
