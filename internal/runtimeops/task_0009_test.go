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

func TestTask0009TelemetryPageTotalMatchesTenantItems(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	must(t, s.CreateBatch(ctx, "fleet-a", "page-a", "collecting"))
	must(t, s.CreateBatch(ctx, "fleet-b", "page-b", "collecting"))
	must(t, s.RecordEvent(ctx, Event{ID: "a-one", TenantID: "fleet-a", BatchID: "page-a", Status: "classified", Magnitude: 1}))
	must(t, s.RecordEvent(ctx, Event{ID: "b-one", TenantID: "fleet-b", BatchID: "page-b", Status: "classified", Magnitude: 2}))
	must(t, s.RecordEvent(ctx, Event{ID: "b-two", TenantID: "fleet-b", BatchID: "page-b", Status: "unclassified", Magnitude: 3}))
	page, err := s.ListEvents(ctx, "fleet-a", "", 1, 10)
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].TenantID != "fleet-a" {
		t.Errorf("fleet-a page=%+v err=%v", page, err)
	}
	filtered, err := s.ListEvents(ctx, "fleet-b", "unclassified", 1, 10)
	if err != nil || filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].ID != "b-two" {
		t.Errorf("filtered page=%+v err=%v", filtered, err)
	}
}
