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

func TestTask0002VersionConflictLeavesNoApprovalAudit(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	must(t, s.CreatePermit(ctx, Permit{ID: "permit-v", TenantID: "fleet-v", Slot: "dock-v", State: "pending", Version: 9}))
	if err := s.ApprovePermit(ctx, "fleet-v", "permit-v", 8); !IsConflict(err) {
		t.Errorf("stale approval error=%v", err)
	}
	permit, err := s.Permit(ctx, "fleet-v", "permit-v")
	if err != nil || permit.State != "pending" || permit.Version != 9 {
		t.Errorf("permit=%+v err=%v", permit, err)
	}
	count, err := s.AuditCount(ctx, "fleet-v", "permit-v")
	if err != nil || count != 0 {
		t.Errorf("stale approval audit count=%d err=%v", count, err)
	}
	must(t, s.ApprovePermit(ctx, "fleet-v", "permit-v", 9))
	permit, err = s.Permit(ctx, "fleet-v", "permit-v")
	if err != nil || permit.State != "approved" || permit.Version != 10 {
		t.Errorf("valid approval permit=%+v err=%v", permit, err)
	}
	count, err = s.AuditCount(ctx, "fleet-v", "permit-v")
	if err != nil || count != 1 {
		t.Errorf("valid approval audit count=%d err=%v", count, err)
	}
}
