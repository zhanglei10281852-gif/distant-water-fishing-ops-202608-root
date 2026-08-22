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

func TestTask0001ApprovalFailureKeepsPermitPending(t *testing.T) {
	ctx := context.Background()
	t.Run("audit rejection rolls back state", func(t *testing.T) {
		s := testStore(t)
		must(t, s.CreatePermit(ctx, Permit{ID: "permit-a", TenantID: "fleet-a", Slot: "berth-7", State: "pending", Version: 4}))
		_, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_permit_audit BEFORE INSERT ON audit BEGIN SELECT RAISE(ABORT, 'audit offline'); END`)
		must(t, err)
		if err = s.ApprovePermit(ctx, "fleet-a", "permit-a", 4); err == nil {
			t.Error("approval unexpectedly succeeded")
		}
		permit, readErr := s.Permit(ctx, "fleet-a", "permit-a")
		if readErr != nil || permit.State != "pending" || permit.Version != 4 {
			t.Errorf("permit after failure=%+v err=%v", permit, readErr)
		}
		count, countErr := s.AuditCount(ctx, "fleet-a", "permit-a")
		if countErr != nil || count != 0 {
			t.Errorf("audit count=%d err=%v", count, countErr)
		}
	})
	t.Run("normal approval persists state and audit", func(t *testing.T) {
		s := testStore(t)
		must(t, s.CreatePermit(ctx, Permit{ID: "permit-ok", TenantID: "fleet-a", Slot: "berth-8", State: "pending", Version: 1}))
		must(t, s.ApprovePermit(ctx, "fleet-a", "permit-ok", 1))
		permit, err := s.Permit(ctx, "fleet-a", "permit-ok")
		if err != nil || permit.State != "approved" || permit.Version != 2 {
			t.Errorf("permit=%+v err=%v", permit, err)
		}
		count, err := s.AuditCount(ctx, "fleet-a", "permit-ok")
		if err != nil || count != 1 {
			t.Errorf("audit count=%d err=%v", count, err)
		}
	})
}
