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

func TestTask0024FailedPipelineStepRunsAgainOnRetry(t *testing.T) {
	ctx := context.Background()
	pipeline := NewPipeline()
	calls := 0
	temporary := errors.New("satellite unavailable")
	steps := map[string]Step{"publish": func(context.Context) error {
		calls++
		if calls == 1 {
			return temporary
		}
		return nil
	}}
	if err := pipeline.Run(ctx, steps, []string{"publish"}); !errors.Is(err, temporary) {
		t.Errorf("first run error=%v", err)
	}
	if snapshot := pipeline.Snapshot(); snapshot["publish"] {
		t.Errorf("failed step marked complete: %+v", snapshot)
	}
	if err := pipeline.Run(ctx, steps, []string{"publish"}); err != nil {
		t.Errorf("retry failed: %v", err)
	}
	if calls != 2 {
		t.Errorf("step calls=%d", calls)
	}
	if !pipeline.Snapshot()["publish"] {
		t.Errorf("successful step not recorded: %+v", pipeline.Snapshot())
	}
}
