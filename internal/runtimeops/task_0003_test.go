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

func TestTask0003DispatchFailureDoesNotConsumePermit(t *testing.T) {
	ctx := context.Background()
	t.Run("notification rejection preserves approved permit", func(t *testing.T) {
		s := testStore(t)
		must(t, s.CreatePermit(ctx, Permit{ID: "dispatch-a", TenantID: "fleet-d", Slot: "slot-d", State: "approved", Version: 2}))
		_, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_dispatch_outbox BEFORE INSERT ON outbox BEGIN SELECT RAISE(ABORT, 'outbox unavailable'); END`)
		must(t, err)
		if err = s.ExecutePermit(ctx, "fleet-d", "dispatch-a", 2); err == nil {
			t.Error("dispatch unexpectedly succeeded")
		}
		permit, readErr := s.Permit(ctx, "fleet-d", "dispatch-a")
		if readErr != nil || permit.State != "approved" || permit.Version != 2 {
			t.Errorf("permit=%+v err=%v", permit, readErr)
		}
		count, countErr := s.OutboxCount(ctx, "fleet-d", "dispatch-a")
		if countErr != nil || count != 0 {
			t.Errorf("outbox=%d err=%v", count, countErr)
		}
	})
	t.Run("dispatch succeeds when notification is durable", func(t *testing.T) {
		s := testStore(t)
		must(t, s.CreatePermit(ctx, Permit{ID: "dispatch-ok", TenantID: "fleet-d", Slot: "slot-ok", State: "approved", Version: 5}))
		must(t, s.ExecutePermit(ctx, "fleet-d", "dispatch-ok", 5))
		permit, err := s.Permit(ctx, "fleet-d", "dispatch-ok")
		if err != nil || permit.State != "executing" || permit.Version != 6 {
			t.Errorf("permit=%+v err=%v", permit, err)
		}
		count, err := s.OutboxCount(ctx, "fleet-d", "dispatch-ok")
		if err != nil || count != 1 {
			t.Errorf("outbox=%d err=%v", count, err)
		}
	})
}
