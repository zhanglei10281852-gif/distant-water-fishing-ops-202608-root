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

func TestTask0004CancellationAndNoticeRemainAtomic(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	must(t, s.CreatePermit(ctx, Permit{ID: "cancel-a", TenantID: "fleet-c", Slot: "slot-c", State: "approved", Version: 3}))
	_, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_cancel_notice BEFORE INSERT ON outbox BEGIN SELECT RAISE(ABORT, 'notice offline'); END`)
	must(t, err)
	if err = s.CancelPermit(ctx, "fleet-c", "cancel-a", 3); err == nil {
		t.Error("cancellation unexpectedly succeeded")
	}
	permit, err := s.Permit(ctx, "fleet-c", "cancel-a")
	if err != nil || permit.State != "approved" || permit.Version != 3 {
		t.Errorf("permit after rejected notice=%+v err=%v", permit, err)
	}
	count, err := s.OutboxCount(ctx, "fleet-c", "cancel-a")
	if err != nil || count != 0 {
		t.Errorf("notice count=%d err=%v", count, err)
	}
	_, err = s.db.ExecContext(ctx, `DROP TRIGGER reject_cancel_notice`)
	must(t, err)
	must(t, s.CancelPermit(ctx, "fleet-c", "cancel-a", 3))
	permit, err = s.Permit(ctx, "fleet-c", "cancel-a")
	if err != nil || permit.State != "cancelled" || permit.Version != 4 {
		t.Errorf("valid cancellation=%+v err=%v", permit, err)
	}
	count, err = s.OutboxCount(ctx, "fleet-c", "cancel-a")
	if err != nil || count != 1 {
		t.Errorf("valid notice count=%d err=%v", count, err)
	}
}
