package runtimeops

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (s *Store) CreatePermit(ctx context.Context, p Permit) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO permits(id,tenant_id,slot,state,version) VALUES(?,?,?,?,?)`, p.ID, p.TenantID, p.Slot, p.State, p.Version)
	return err
}

func scanPermit(row *sql.Row) (Permit, error) {
	var p Permit
	err := row.Scan(&p.ID, &p.TenantID, &p.Slot, &p.State, &p.Version)
	return p, mapError(err)
}

func (s *Store) Permit(ctx context.Context, tenant, id string) (Permit, error) {
	return scanPermit(s.db.QueryRowContext(ctx, `SELECT id,tenant_id,slot,state,version FROM permits WHERE tenant_id=? AND id=?`, tenant, id))
}

func (s *Store) ApprovePermit(ctx context.Context, tenant, id string, expected int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE permits SET state='approved',version=version+1 WHERE tenant_id=? AND id=? AND state='pending' AND version=?`, tenant, id, expected)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return ErrConflict
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO audit(tenant_id,entity_id,action) VALUES(?,?,?)`, tenant, id, "permit_approved"); err != nil {
			return fmt.Errorf("write approval audit: %w", err)
		}
		return nil
	})
}

func (s *Store) ExecutePermit(ctx context.Context, tenant, id string, expected int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var slot, state string
		if err := tx.QueryRowContext(ctx, `SELECT slot,state FROM permits WHERE tenant_id=? AND id=?`, tenant, id).Scan(&slot, &state); err != nil {
			return mapError(err)
		}
		if state != "approved" {
			return ErrConflict
		}
		var occupied int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM permits WHERE tenant_id=? AND slot=? AND state='executing'`, tenant, slot).Scan(&occupied); err != nil {
			return err
		}
		if occupied > 0 {
			return ErrConflict
		}
		result, err := tx.ExecContext(ctx, `UPDATE permits SET state='executing',version=version+1 WHERE tenant_id=? AND id=? AND version=?`, tenant, id, expected)
		if err != nil {
			return err
		}
		rows, _ := result.RowsAffected()
		if rows != 1 {
			return ErrConflict
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO outbox(tenant_id,entity_id,payload) VALUES(?,?,?)`, tenant, id, "permit_executing")
		return err
	})
}

func (s *Store) CancelPermit(ctx context.Context, tenant, id string, expected int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE permits SET state='cancelled',version=version+1 WHERE tenant_id=? AND id=? AND state IN ('pending','approved') AND version=?`, tenant, id, expected)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return ErrConflict
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		_, err = tx.ExecContext(ctx, `INSERT INTO outbox(tenant_id,entity_id,payload) VALUES(?,?,?)`, tenant, id, "permit_cancelled")
		return err
	})
}

func (s *Store) ListPermits(ctx context.Context, tenant string) ([]Permit, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,tenant_id,slot,state,version FROM permits WHERE tenant_id=? ORDER BY id`, tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Permit{}
	for rows.Next() {
		var p Permit
		if err = rows.Scan(&p.ID, &p.TenantID, &p.Slot, &p.State, &p.Version); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) AuditCount(ctx context.Context, tenant, entity string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit WHERE tenant_id=? AND entity_id=?`, tenant, entity).Scan(&n)
	return n, err
}

func (s *Store) OutboxCount(ctx context.Context, tenant, entity string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox WHERE tenant_id=? AND entity_id=?`, tenant, entity).Scan(&n)
	return n, err
}

func IsConflict(err error) bool { return errors.Is(err, ErrConflict) }
