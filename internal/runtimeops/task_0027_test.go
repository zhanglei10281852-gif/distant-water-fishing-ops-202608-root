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

func TestTask0027RetryOperationReceivesCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	sawCancel := make(chan struct{}, 1)
	operation := func(opCtx context.Context) error {
		select {
		case <-started:
		default:
			close(started)
		}
		select {
		case <-opCtx.Done():
			sawCancel <- struct{}{}
			return opCtx.Err()
		case <-time.After(250 * time.Millisecond):
			return errors.New("operation context stayed alive")
		}
	}
	go func() { <-started; cancel() }()
	before := time.Now()
	err := Retry(ctx, 3, time.Second, operation)
	elapsed := time.Since(before)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("retry error=%v", err)
	}
	if elapsed >= 150*time.Millisecond {
		t.Errorf("cancellation took %s", elapsed)
	}
	select {
	case <-sawCancel:
	default:
		t.Error("operation did not observe caller cancellation")
	}
	liveCalls := 0
	err = Retry(context.Background(), 2, time.Millisecond, func(context.Context) error {
		liveCalls++
		if liveCalls == 1 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil || liveCalls != 2 {
		t.Errorf("live retry calls=%d err=%v", liveCalls, err)
	}
}
