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

func TestTask0025SnapshotMutationCannotChangePipelineState(t *testing.T) {
	ctx := context.Background()
	pipeline := NewPipeline()
	calls := 0
	steps := map[string]Step{"archive": func(context.Context) error { calls++; return nil }}
	must(t, pipeline.Run(ctx, steps, []string{"archive"}))
	snapshot := pipeline.Snapshot()
	snapshot["archive"] = false
	snapshot["invented"] = true
	current := pipeline.Snapshot()
	if !current["archive"] || current["invented"] {
		t.Errorf("pipeline polluted by snapshot mutation: %+v", current)
	}
	must(t, pipeline.Run(ctx, steps, []string{"archive"}))
	if calls != 1 {
		t.Errorf("completed step reran %d times", calls)
	}
}
