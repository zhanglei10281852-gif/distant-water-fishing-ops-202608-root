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

func TestTask0023CheckpointReadsCannotCrossFleetBoundary(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	must(t, s.SaveCheckpoint(ctx, Checkpoint{TenantID: "fleet-a", Stream: "position-feed", Generation: 3, Payload: []byte("alpha")}))
	must(t, s.SaveCheckpoint(ctx, Checkpoint{TenantID: "fleet-b", Stream: "position-feed", Generation: 8, Payload: []byte("bravo")}))
	for _, item := range []struct {
		tenant     string
		generation int64
		payload    string
	}{{"fleet-a", 3, "alpha"}, {"fleet-b", 8, "bravo"}} {
		checkpoint, err := s.Checkpoint(ctx, item.tenant, "position-feed")
		if err != nil || checkpoint.TenantID != item.tenant || checkpoint.Generation != item.generation || string(checkpoint.Payload) != item.payload {
			t.Errorf("checkpoint %s=%+v err=%v", item.tenant, checkpoint, err)
		}
	}
	_, err := s.Checkpoint(ctx, "fleet-c", "position-feed")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown fleet read error=%v", err)
	}
}
