package runtimeops

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrExpired  = errors.New("expired")
)

type Store struct{ db *sql.DB }

func Open(ctx context.Context, path string) (*Store, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
	}
	dsn := path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	if path == ":memory:" {
		dsn = "file:runtimeops?mode=memory&cache=shared&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open runtime store: %w", err)
	}
	db.SetMaxOpenConns(8)
	s := &Store{db: db}
	if err = s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS permits(id TEXT PRIMARY KEY,tenant_id TEXT NOT NULL,slot TEXT NOT NULL,state TEXT NOT NULL,version INTEGER NOT NULL)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS active_slot ON permits(tenant_id,slot) WHERE state='executing'`,
		`CREATE TABLE IF NOT EXISTS audit(id INTEGER PRIMARY KEY AUTOINCREMENT,tenant_id TEXT NOT NULL,entity_id TEXT NOT NULL,action TEXT NOT NULL)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS audit_entity_action ON audit(tenant_id,entity_id,action)`,
		`CREATE TABLE IF NOT EXISTS outbox(id INTEGER PRIMARY KEY AUTOINCREMENT,tenant_id TEXT NOT NULL,entity_id TEXT NOT NULL,payload TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS batches(id TEXT PRIMARY KEY,tenant_id TEXT NOT NULL,state TEXT NOT NULL,event_count INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS events(id TEXT NOT NULL,tenant_id TEXT NOT NULL,batch_id TEXT NOT NULL,status TEXT NOT NULL,magnitude REAL NOT NULL,PRIMARY KEY(tenant_id,id),FOREIGN KEY(batch_id) REFERENCES batches(id))`,
		`CREATE TABLE IF NOT EXISTS jobs(id TEXT PRIMARY KEY,tenant_id TEXT NOT NULL,state TEXT NOT NULL,payload TEXT NOT NULL,attempts INTEGER NOT NULL,max_attempts INTEGER NOT NULL,available_at INTEGER NOT NULL,lease_until INTEGER)`,
		`CREATE TABLE IF NOT EXISTS commands(tenant_id TEXT NOT NULL,method TEXT NOT NULL,path TEXT NOT NULL,key TEXT NOT NULL,request_hash TEXT NOT NULL,response BLOB NOT NULL,PRIMARY KEY(tenant_id,method,path,key))`,
		`CREATE TABLE IF NOT EXISTS checkpoints(tenant_id TEXT NOT NULL,stream TEXT NOT NULL,generation INTEGER NOT NULL,payload BLOB NOT NULL,PRIMARY KEY(tenant_id,stream))`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate runtime store: %w", err)
		}
	}
	return nil
}

func withTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
	}()
	if err = fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func mapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func unix(t time.Time) int64     { return t.UTC().UnixNano() }
func fromUnix(v int64) time.Time { return time.Unix(0, v).UTC() }
