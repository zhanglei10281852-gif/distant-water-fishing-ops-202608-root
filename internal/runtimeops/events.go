package runtimeops

import (
	"context"
	"database/sql"
)

func (s *Store) CreateBatch(ctx context.Context, tenant, id, state string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO batches(id,tenant_id,state,event_count) VALUES(?,?,?,0)`, id, tenant, state)
	return err
}

func (s *Store) RecordEvent(ctx context.Context, event Event) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM batches WHERE id=? AND tenant_id=?`, event.BatchID, event.TenantID).Scan(&state); err != nil {
			return mapError(err)
		}
		if state != "collecting" {
			return ErrConflict
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO events(id,tenant_id,batch_id,status,magnitude) VALUES(?,?,?,?,?)`, event.ID, event.TenantID, event.BatchID, event.Status, event.Magnitude); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE batches SET event_count=event_count+1 WHERE id=? AND tenant_id=? AND state='collecting'`, event.BatchID, event.TenantID)
		if err != nil {
			return err
		}
		rows, _ := result.RowsAffected()
		if rows != 1 {
			return ErrConflict
		}
		return nil
	})
}

func (s *Store) CloseBatch(ctx context.Context, tenant, id string) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var unresolved int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM events WHERE tenant_id=? AND batch_id=? AND status='unclassified'`, tenant, id).Scan(&unresolved); err != nil {
			return err
		}
		if unresolved > 0 {
			return ErrConflict
		}
		result, err := tx.ExecContext(ctx, `UPDATE batches SET state='closed' WHERE id=? AND tenant_id=? AND state='collecting'`, id, tenant)
		if err != nil {
			return err
		}
		rows, _ := result.RowsAffected()
		if rows != 1 {
			return ErrConflict
		}
		return nil
	})
}

func (s *Store) ClassifyEvent(ctx context.Context, tenant, id, status string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE events SET status=? WHERE tenant_id=? AND id=? AND status='unclassified'`, status, tenant, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return ErrConflict
	}
	return nil
}

func (s *Store) BatchState(ctx context.Context, tenant, id string) (string, int, error) {
	var state string
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT state,event_count FROM batches WHERE tenant_id=? AND id=?`, tenant, id).Scan(&state, &count)
	return state, count, mapError(err)
}

func (s *Store) ListEvents(ctx context.Context, tenant, status string, page, size int) (EventPage, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	where := `tenant_id=?`
	args := []any{tenant}
	if status != "" {
		where += ` AND status=?`
		args = append(args, status)
	}
	countWhere := `1=1`
	countArgs := []any{}
	if status != "" {
		countWhere += ` AND status=?`
		countArgs = append(countArgs, status)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events WHERE `+countWhere, countArgs...).Scan(&total); err != nil {
		return EventPage{}, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.QueryContext(ctx, `SELECT id,tenant_id,batch_id,status,magnitude FROM events WHERE `+where+` ORDER BY id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return EventPage{}, err
	}
	defer rows.Close()
	result := EventPage{Items: []Event{}, Total: total}
	for rows.Next() {
		var event Event
		if err = rows.Scan(&event.ID, &event.TenantID, &event.BatchID, &event.Status, &event.Magnitude); err != nil {
			return EventPage{}, err
		}
		result.Items = append(result.Items, event)
	}
	return result, rows.Err()
}
