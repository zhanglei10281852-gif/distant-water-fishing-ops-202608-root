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

func TestTask0028CancellationInterruptsRetryBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempted := make(chan struct{})
	calls := 0
	operation := func(context.Context) error {
		calls++
		if calls == 1 {
			close(attempted)
		}
		return errors.New("uplink unavailable")
	}
	go func() { <-attempted; cancel() }()
	before := time.Now()
	err := Retry(ctx, 5, 300*time.Millisecond, operation)
	elapsed := time.Since(before)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("retry error=%v", err)
	}
	if elapsed >= 150*time.Millisecond {
		t.Errorf("backoff ignored cancellation for %s", elapsed)
	}
	if calls != 1 {
		t.Errorf("calls after cancellation=%d", calls)
	}
	liveCalls := 0
	err = Retry(context.Background(), 2, time.Millisecond, func(context.Context) error {
		liveCalls++
		if liveCalls == 1 {
			return errors.New("once")
		}
		return nil
	})
	if err != nil || liveCalls != 2 {
		t.Errorf("live retry calls=%d err=%v", liveCalls, err)
	}
}
