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

func TestTask0029ProcessorHandlerSharesRequestLifetime(t *testing.T) {
	base := context.Background()
	s := testStore(t)
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	must(t, s.CreateJob(base, Job{ID: "processor-cancel", TenantID: "fleet-u", State: "pending", Payload: "x", MaxAttempts: 3, AvailableAt: now}))
	processor := Processor{Store: s, Clock: func() time.Time { return now }, Lease: time.Minute}
	ctx, cancel := context.WithCancel(base)
	started := make(chan struct{})
	sawCancel := make(chan struct{}, 1)
	handler := func(handlerCtx context.Context, job Job) error {
		close(started)
		select {
		case <-handlerCtx.Done():
			sawCancel <- struct{}{}
			return handlerCtx.Err()
		case <-time.After(250 * time.Millisecond):
			return errors.New("handler context stayed alive")
		}
	}
	go func() { <-started; cancel() }()
	before := time.Now()
	err := processor.RunOnce(ctx, "fleet-u", handler)
	elapsed := time.Since(before)
	if elapsed >= 150*time.Millisecond {
		t.Errorf("processor cancellation took %s", elapsed)
	}
	if err == nil {
		t.Error("cancelled processor returned nil")
	}
	select {
	case <-sawCancel:
	default:
		t.Error("handler did not receive request cancellation")
	}
	job, readErr := s.Job(base, "fleet-u", "processor-cancel")
	if readErr != nil || job.State == "done" {
		t.Errorf("job after cancellation=%+v err=%v", job, readErr)
	}
}
