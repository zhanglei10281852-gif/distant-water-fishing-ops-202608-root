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

func TestTask0022StaleCheckpointCannotOverwriteNewerGeneration(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	must(t, s.SaveCheckpoint(ctx, Checkpoint{TenantID: "fleet-t", Stream: "telemetry", Generation: 5, Payload: []byte("generation-five")}))
	if err := s.SaveCheckpoint(ctx, Checkpoint{TenantID: "fleet-t", Stream: "telemetry", Generation: 4, Payload: []byte("stale")}); !IsConflict(err) {
		t.Errorf("stale checkpoint error=%v", err)
	}
	checkpoint, err := s.Checkpoint(ctx, "fleet-t", "telemetry")
	if err != nil || checkpoint.Generation != 5 || string(checkpoint.Payload) != "generation-five" {
		t.Errorf("checkpoint after stale write=%+v err=%v", checkpoint, err)
	}
	must(t, s.SaveCheckpoint(ctx, Checkpoint{TenantID: "fleet-t", Stream: "telemetry", Generation: 6, Payload: []byte("generation-six")}))
	checkpoint, err = s.Checkpoint(ctx, "fleet-t", "telemetry")
	if err != nil || checkpoint.Generation != 6 || string(checkpoint.Payload) != "generation-six" {
		t.Errorf("new checkpoint=%+v err=%v", checkpoint, err)
	}
}
