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

func TestTask0010TelemetryRowsStayInsideFleetBoundary(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	must(t, s.CreateBatch(ctx, "fleet-a", "items-a", "collecting"))
	must(t, s.CreateBatch(ctx, "fleet-b", "items-b", "collecting"))
	must(t, s.RecordEvent(ctx, Event{ID: "a-event", TenantID: "fleet-a", BatchID: "items-a", Status: "classified", Magnitude: 1}))
	must(t, s.RecordEvent(ctx, Event{ID: "b-event", TenantID: "fleet-b", BatchID: "items-b", Status: "classified", Magnitude: 2}))
	page, err := s.ListEvents(ctx, "fleet-a", "classified", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Errorf("fleet-a page=%+v", page)
	}
	for _, event := range page.Items {
		if event.TenantID != "fleet-a" || event.ID != "a-event" {
			t.Errorf("foreign event leaked: %+v", event)
		}
	}
	empty, err := s.ListEvents(ctx, "fleet-a", "unclassified", 1, 20)
	if err != nil || empty.Total != 0 || len(empty.Items) != 0 {
		t.Errorf("empty legal filter=%+v err=%v", empty, err)
	}
}
