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

func TestTask0005TelemetryWriteRollsBackWhenBatchUpdateFails(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	must(t, s.CreateBatch(ctx, "fleet-e", "batch-e", "collecting"))
	_, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_batch_counter BEFORE UPDATE ON batches BEGIN SELECT RAISE(ABORT, 'batch locked'); END`)
	must(t, err)
	input := Event{ID: "event-failed", TenantID: "fleet-e", BatchID: "batch-e", Status: "unclassified", Magnitude: 7.4}
	if err = s.RecordEvent(ctx, input); err == nil {
		t.Error("event unexpectedly committed")
	}
	state, count, stateErr := s.BatchState(ctx, "fleet-e", "batch-e")
	if stateErr != nil || state != "collecting" || count != 0 {
		t.Errorf("batch=%s count=%d err=%v", state, count, stateErr)
	}
	page, pageErr := s.ListEvents(ctx, "fleet-e", "", 1, 10)
	if pageErr != nil || page.Total != 0 || len(page.Items) != 0 {
		t.Errorf("events=%+v err=%v", page, pageErr)
	}
	_, err = s.db.ExecContext(ctx, `DROP TRIGGER reject_batch_counter`)
	must(t, err)
	must(t, s.RecordEvent(ctx, Event{ID: "event-ok", TenantID: "fleet-e", BatchID: "batch-e", Status: "classified", Magnitude: 2.1}))
	state, count, stateErr = s.BatchState(ctx, "fleet-e", "batch-e")
	if stateErr != nil || state != "collecting" || count != 1 {
		t.Errorf("valid batch=%s count=%d err=%v", state, count, stateErr)
	}
}
