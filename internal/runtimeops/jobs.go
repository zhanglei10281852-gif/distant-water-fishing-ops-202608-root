package runtimeops

import (
	"context"
	"database/sql"
	"time"
)

func (s *Store) CreateJob(ctx context.Context, j Job) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO jobs(id,tenant_id,state,payload,attempts,max_attempts,available_at,lease_until) VALUES(?,?,?,?,?,?,?,NULL)`, j.ID, j.TenantID, j.State, j.Payload, j.Attempts, j.MaxAttempts, unix(j.AvailableAt))
	return err
}

func scanJob(row *sql.Row) (Job, error) {
	var j Job
	var available int64
	var lease *int64
	err := row.Scan(&j.ID, &j.TenantID, &j.State, &j.Payload, &j.Attempts, &j.MaxAttempts, &available, &lease)
	if lease != nil {
		t := fromUnix(*lease)
		j.LeaseUntil = &t
	}
	j.AvailableAt = fromUnix(available)
	return j, mapError(err)
}

func (s *Store) Job(ctx context.Context, tenant, id string) (Job, error) {
	return scanJob(s.db.QueryRowContext(ctx, `SELECT id,tenant_id,state,payload,attempts,max_attempts,available_at,lease_until FROM jobs WHERE tenant_id=? AND id=?`, tenant, id))
}

func (s *Store) ClaimJobs(ctx context.Context, tenant string, now time.Time, limit int, lease time.Duration) ([]Job, error) {
	if limit < 1 {
		limit = 1
	}
	out := []Job{}
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT id FROM jobs WHERE tenant_id=? AND state IN ('pending','failed') AND available_at<=? AND (lease_until IS NULL OR lease_until<=?) ORDER BY available_at,id LIMIT ?`, tenant, unix(now), unix(now), limit)
		if err != nil {
			return err
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		if err = rows.Close(); err != nil {
			return err
		}
		until := now.Add(lease)
		for _, id := range ids {
			result, err := tx.ExecContext(ctx, `UPDATE jobs SET state='running',attempts=attempts+1,lease_until=? WHERE tenant_id=? AND id=? AND state IN ('pending','failed') AND (lease_until IS NULL OR lease_until<=?)`, unix(until), tenant, id, unix(now))
			if err != nil {
				return err
			}
			n, _ := result.RowsAffected()
			if n != 1 {
				continue
			}
			var j Job
			var available int64
			var leaseValue *int64
			if err = tx.QueryRowContext(ctx, `SELECT id,tenant_id,state,payload,attempts,max_attempts,available_at,lease_until FROM jobs WHERE tenant_id=? AND id=?`, tenant, id).Scan(&j.ID, &j.TenantID, &j.State, &j.Payload, &j.Attempts, &j.MaxAttempts, &available, &leaseValue); err != nil {
				return err
			}
			j.AvailableAt = fromUnix(available)
			if leaseValue != nil {
				t := fromUnix(*leaseValue)
				j.LeaseUntil = &t
			}
			out = append(out, j)
		}
		return nil
	})
	return out, err
}

func (s *Store) CompleteJob(ctx context.Context, tenant, id string, attempt int) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE jobs SET state='done',lease_until=NULL WHERE tenant_id=? AND id=? AND state='running' AND attempts=?`, tenant, id, attempt)
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			return ErrConflict
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO outbox(tenant_id,entity_id,payload) VALUES(?,?,?)`, tenant, id, "job_done")
		return err
	})
}

func (s *Store) FailJob(ctx context.Context, tenant, id string, attempt int, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var max int
		if err := tx.QueryRowContext(ctx, `SELECT max_attempts FROM jobs WHERE tenant_id=? AND id=? AND state='running' AND attempts=?`, tenant, id, attempt).Scan(&max); err != nil {
			return mapError(err)
		}
		state := "failed"
		if attempt >= max {
			state = "dead"
		}
		next := now.Add(time.Duration(attempt) * time.Minute)
		result, err := tx.ExecContext(ctx, `UPDATE jobs SET state=?,available_at=?,lease_until=NULL WHERE tenant_id=? AND id=? AND state='running' AND attempts=?`, state, unix(next), tenant, id, attempt)
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			return ErrConflict
		}
		return nil
	})
}

func (s *Store) ReclaimJobs(ctx context.Context, tenant string, now time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE jobs SET state='failed',lease_until=NULL WHERE tenant_id=? AND state='running' AND lease_until<=?`, tenant, unix(now))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
