// Package store 用 SQLite 持久化证据批次、校正、归属、裁决与冻结播出表。
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"

	"task278-broadcastslot/internal/model"
)

// execer 覆盖 *sql.DB 与 *sql.Tx 的查询面，便于事务内外共用扫描逻辑。
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store 封装单写连接。mu 覆盖 WithTx 的 Begin→Commit，避免事务被切开。
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// Open 打开（或创建）数据库：MaxOpenConns=1、WAL、foreign_keys=ON，然后迁移。
func Open(path string) (*Store, error) {
	if path == "" {
		path = ":memory:"
	}
	if path != ":memory:" {
		if dir := filepath.Dir(path); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("store: mkdir: %w", err)
			}
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: pragma: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close 关闭底层连接。
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// WithTx 在同一把锁内 Begin→fn→Commit；ctx 取消则回滚。
func (s *Store) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit: %w", err)
	}
	return nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS evidence_batches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			station TEXT NOT NULL DEFAULT '',
			air_date TEXT NOT NULL DEFAULT '',
			timezone TEXT NOT NULL DEFAULT '',
			drift_ppm REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			sealed_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS program_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL,
			fingerprint TEXT NOT NULL,
			title TEXT NOT NULL,
			callsign TEXT NOT NULL DEFAULT '',
			printed_start_ms INTEGER NOT NULL,
			printed_end_ms INTEGER NOT NULL,
			page_id TEXT NOT NULL DEFAULT '',
			transmitter TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			UNIQUE(batch_id, fingerprint),
			FOREIGN KEY(batch_id) REFERENCES evidence_batches(id)
		)`,
		`CREATE TABLE IF NOT EXISTS station_clips (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL,
			clip_no INTEGER NOT NULL,
			callsign TEXT NOT NULL DEFAULT '',
			offset_ms INTEGER NOT NULL DEFAULT 0,
			source TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'raw',
			UNIQUE(batch_id, clip_no),
			FOREIGN KEY(batch_id) REFERENCES evidence_batches(id)
		)`,
		`CREATE TABLE IF NOT EXISTS newspaper_ads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL,
			ad_no INTEGER NOT NULL,
			printed_start_ms INTEGER NOT NULL,
			page_id TEXT NOT NULL DEFAULT '',
			edition TEXT NOT NULL DEFAULT '',
			note TEXT NOT NULL DEFAULT '',
			UNIQUE(batch_id, ad_no),
			FOREIGN KEY(batch_id) REFERENCES evidence_batches(id)
		)`,
		`CREATE TABLE IF NOT EXISTS source_citations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL,
			from_ref TEXT NOT NULL,
			to_ref TEXT NOT NULL,
			kind TEXT NOT NULL DEFAULT '',
			UNIQUE(batch_id, from_ref, to_ref),
			FOREIGN KEY(batch_id) REFERENCES evidence_batches(id)
		)`,
		`CREATE TABLE IF NOT EXISTS clock_corrections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL,
			subject_kind TEXT NOT NULL,
			subject_id INTEGER NOT NULL,
			printed_ms INTEGER NOT NULL,
			utc_ms INTEGER NOT NULL,
			method TEXT NOT NULL DEFAULT '',
			applied_at TEXT NOT NULL,
			FOREIGN KEY(batch_id) REFERENCES evidence_batches(id)
		)`,
		`CREATE TABLE IF NOT EXISTS slot_attributions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL,
			entry_id INTEGER NOT NULL,
			clip_id INTEGER NOT NULL DEFAULT 0,
			utc_start_ms INTEGER NOT NULL,
			utc_end_ms INTEGER NOT NULL,
			status TEXT NOT NULL,
			delay_ms INTEGER NOT NULL DEFAULT 0,
			UNIQUE(batch_id, entry_id),
			FOREIGN KEY(batch_id) REFERENCES evidence_batches(id)
		)`,
		`CREATE TABLE IF NOT EXISTS attribution_conflicts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL,
			left_entry_id INTEGER NOT NULL,
			right_entry_id INTEGER NOT NULL DEFAULT 0,
			kind TEXT NOT NULL,
			detail TEXT NOT NULL DEFAULT '',
			FOREIGN KEY(batch_id) REFERENCES evidence_batches(id)
		)`,
		`CREATE TABLE IF NOT EXISTS slot_verdicts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL,
			entry_id INTEGER NOT NULL,
			decision TEXT NOT NULL,
			reviewer TEXT NOT NULL DEFAULT '',
			note TEXT NOT NULL DEFAULT '',
			expected_version INTEGER NOT NULL DEFAULT 0,
			UNIQUE(batch_id, entry_id),
			FOREIGN KEY(batch_id) REFERENCES evidence_batches(id)
		)`,
		`CREATE TABLE IF NOT EXISTS schedule_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL,
			version INTEGER NOT NULL,
			status TEXT NOT NULL,
			sealed INTEGER NOT NULL DEFAULT 0,
			payload TEXT NOT NULL DEFAULT '',
			content_hash TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(batch_id, version),
			FOREIGN KEY(batch_id) REFERENCES evidence_batches(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_entries_batch ON program_entries(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_clips_batch ON station_clips(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ads_batch ON newspaper_ads(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cite_batch ON source_citations(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_corr_batch ON clock_corrections(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_attr_batch ON slot_attributions(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_conf_batch ON attribution_conflicts(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_verd_batch ON slot_verdicts(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ver_batch ON schedule_versions(batch_id)`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(st); err != nil {
			return fmt.Errorf("store: migrate: %w", err)
		}
	}
	return nil
}

func isUniqueErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func wrapUnique(err error, sentinel error) error {
	if isUniqueErr(err) {
		return fmt.Errorf("%w", sentinel)
	}
	return err
}

func notFoundBatch(err error) error {
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w", model.ErrBatchNotFound)
	}
	return err
}
