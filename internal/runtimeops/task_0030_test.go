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

func TestTask0030HandlerFailureCannotMarkJobDone(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	now := time.Date(2026, 8, 22, 11, 0, 0, 0, time.UTC)
	processor := Processor{Store: s, Clock: func() time.Time { return now }, Lease: time.Minute}
	must(t, s.CreateJob(ctx, Job{ID: "handler-fail", TenantID: "fleet-v", State: "pending", Payload: "x", MaxAttempts: 3, AvailableAt: now}))
	handlerErr := errors.New("payload rejected")
	if err := processor.RunOnce(ctx, "fleet-v", func(context.Context, Job) error { return handlerErr }); !errors.Is(err, handlerErr) {
		t.Errorf("handler error=%v", err)
	}
	job, err := s.Job(ctx, "fleet-v", "handler-fail")
	if err != nil || job.State != "failed" {
		t.Errorf("failed job=%+v err=%v", job, err)
	}
	count, err := s.OutboxCount(ctx, "fleet-v", "handler-fail")
	if err != nil || count != 0 {
		t.Errorf("failed outbox count=%d err=%v", count, err)
	}
	must(t, s.CreateJob(ctx, Job{ID: "handler-ok", TenantID: "fleet-v", State: "pending", Payload: "y", MaxAttempts: 3, AvailableAt: now}))
	must(t, processor.RunOnce(ctx, "fleet-v", func(context.Context, Job) error { return nil }))
	job, err = s.Job(ctx, "fleet-v", "handler-ok")
	if err != nil || job.State != "done" {
		t.Errorf("successful job=%+v err=%v", job, err)
	}
	count, err = s.OutboxCount(ctx, "fleet-v", "handler-ok")
	if err != nil || count != 1 {
		t.Errorf("successful outbox=%d err=%v", count, err)
	}
}
