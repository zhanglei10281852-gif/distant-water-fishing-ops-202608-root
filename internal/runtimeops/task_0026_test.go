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

func TestTask0026RestoreDoesNotRetainCallerOwnedMap(t *testing.T) {
	ctx := context.Background()
	pipeline := NewPipeline()
	external := map[string]bool{"sync-manifest": true}
	pipeline.Restore(external)
	external["sync-manifest"] = false
	external["foreign-step"] = true
	snapshot := pipeline.Snapshot()
	if !snapshot["sync-manifest"] || snapshot["foreign-step"] {
		t.Errorf("restored state followed caller mutation: %+v", snapshot)
	}
	calls := 0
	must(t, pipeline.Run(ctx, map[string]Step{"sync-manifest": func(context.Context) error { calls++; return nil }}, []string{"sync-manifest"}))
	if calls != 0 {
		t.Errorf("restored completed step reran %d times", calls)
	}
	clean := map[string]bool{"new-step": false}
	pipeline.Restore(clean)
	if pipeline.Snapshot()["new-step"] {
		t.Errorf("valid restore changed false state: %+v", pipeline.Snapshot())
	}
}
