package runtimeops

import (
	"context"
	"errors"
	"testing"
)

// ApprovePermit must atomically persist the state transition and the audit record.
// A rejected audit write must roll back the permit state and version so a dispatcher
// retry never observes an approved permit without an audit trail.

func TestApprovePermitLeavesNoPartialStateOnDuplicateAudit(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreatePermit(ctx, Permit{ID: "p", TenantID: "t", Slot: "s", State: "pending", Version: 1}); err != nil {
		t.Fatal(err)
	}

	// Seed an existing audit row so the unique index forces the audit INSERT to fail
	// inside ApprovePermit. This simulates the database rejecting the audit record.
	if _, err := s.db.ExecContext(ctx, `INSERT INTO audit(tenant_id,entity_id,action) VALUES(?,?,?)`, "t", "p", "permit_approved"); err != nil {
		t.Fatal(err)
	}

	// The approval must fail atomically: permit state and version stay untouched.
	if err := s.ApprovePermit(ctx, "t", "p", 1); !errors.Is(err, ErrConflict) && err == nil {
		t.Fatalf("expected approval to fail when audit is rejected, got %v", err)
	} else if err == nil {
		t.Fatalf("expected approval to fail when audit is rejected, but it succeeded")
	}

	p, err := s.Permit(ctx, "t", "p")
	if err != nil {
		t.Fatal(err)
	}
	if p.State != "pending" || p.Version != 1 {
		t.Fatalf("permit must remain pending@v1 after audit failure, got state=%s version=%d", p.State, p.Version)
	}

	// Dispatcher retry with the same expected version must still be able to observe the
	// original (unchanged) state without a spurious version conflict from a partial commit.
	if err := s.ApprovePermit(ctx, "t", "p", 1); err == nil {
		t.Fatalf("retry against pre-seeded audit row should still conflict, not succeed")
	}

	// Remove the seeded audit row and retry: the approval now succeeds and leaves exactly
	// one audit record (the pre-existing one remains, plus no new duplicate).
	if _, err := s.db.ExecContext(ctx, `DELETE FROM audit WHERE tenant_id=? AND entity_id=?`, "t", "p"); err != nil {
		t.Fatal(err)
	}
	if err := s.ApprovePermit(ctx, "t", "p", 1); err != nil {
		t.Fatalf("retry approval after clearing audit failed: %v", err)
	}
	n, err := s.AuditCount(ctx, "t", "p")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected exactly one audit record after successful approval, got %d", n)
	}
	p, err = s.Permit(ctx, "t", "p")
	if err != nil || p.State != "approved" || p.Version != 2 {
		t.Fatalf("permit must be approved@v2 after success, got %+v err=%v", p, err)
	}
}

func TestApprovePermitConflictKeepsOriginalState(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.CreatePermit(ctx, Permit{ID: "p", TenantID: "t", Slot: "s", State: "pending", Version: 1}); err != nil {
		t.Fatal(err)
	}
	// Wrong expected version -> conflict, no audit written, state unchanged.
	if err := s.ApprovePermit(ctx, "t", "p", 7); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	p, err := s.Permit(ctx, "t", "p")
	if err != nil || p.State != "pending" || p.Version != 1 {
		t.Fatalf("permit must remain pending@v1 on conflict, got %+v err=%v", p, err)
	}
	n, err := s.AuditCount(ctx, "t", "p")
	if err != nil || n != 0 {
		t.Fatalf("expected zero audit records on conflict, got %d err=%v", n, err)
	}
}
