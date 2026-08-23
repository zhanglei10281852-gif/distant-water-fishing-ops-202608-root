package runtimeops

import (
	"context"
	"database/sql"
	"errors"
)

func (s *Store) SaveCommand(ctx context.Context, c Command) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var hash string
		var response []byte
		err := tx.QueryRowContext(ctx, `SELECT request_hash,response FROM commands WHERE tenant_id=? AND method=? AND path=? AND key=?`, c.TenantID, c.Method, c.Path, c.Key).Scan(&hash, &response)
		if err == nil {
			if hash != c.RequestHash {
				return ErrConflict
			}
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		copyResponse := append([]byte(nil), c.Response...)
		_, err = tx.ExecContext(ctx, `INSERT INTO commands(tenant_id,method,path,key,request_hash,response) VALUES(?,?,?,?,?,?)`, c.TenantID, c.Method, c.Path, c.Key, c.RequestHash, copyResponse)
		return err
	})
}

func (s *Store) ReplayCommand(ctx context.Context, tenant, method, path, key, hash string) ([]byte, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	var stored, response string
	err = tx.QueryRowContext(ctx, `SELECT request_hash,response FROM commands WHERE tenant_id=? AND method=? AND path=? AND key=?`, tenant, method, path, key).Scan(&stored, &response)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapError(err)
	}
	if stored != hash {
		_ = tx.Rollback()
		return nil, ErrConflict
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	copyResponse := append([]byte(nil), response...)
	return copyResponse, nil
}

func (s *Store) SaveCheckpoint(ctx context.Context, c Checkpoint) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var generation int64
		err := tx.QueryRowContext(ctx, `SELECT generation FROM checkpoints WHERE tenant_id=? AND stream=?`, c.TenantID, c.Stream).Scan(&generation)
		if errors.Is(err, sql.ErrNoRows) {
			_, err = tx.ExecContext(ctx, `INSERT INTO checkpoints(tenant_id,stream,generation,payload) VALUES(?,?,?,?)`, c.TenantID, c.Stream, c.Generation, append([]byte(nil), c.Payload...))
			return err
		}
		if err != nil {
			return err
		}
		if c.Generation <= generation {
			return ErrConflict
		}
		result, err := tx.ExecContext(ctx, `UPDATE checkpoints SET generation=?,payload=? WHERE tenant_id=? AND stream=? AND generation=?`, c.Generation, append([]byte(nil), c.Payload...), c.TenantID, c.Stream, generation)
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

func (s *Store) Checkpoint(ctx context.Context, tenant, stream string) (Checkpoint, error) {
	var c Checkpoint
	c.TenantID = tenant
	c.Stream = stream
	err := s.db.QueryRowContext(ctx, `SELECT generation,payload FROM checkpoints WHERE tenant_id=? AND stream=?`, tenant, stream).Scan(&c.Generation, &c.Payload)
	c.Payload = append([]byte(nil), c.Payload...)
	return c, mapError(err)
}
