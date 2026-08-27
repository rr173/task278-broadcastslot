package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"task278-broadcastslot/internal/model"
)
func (s *Store) NextVersionNo(batchID int64) (int64, error) {
	var max sql.NullInt64
	err := s.db.QueryRowContext(context.Background(),
		`SELECT MAX(version) FROM schedule_versions WHERE batch_id=?`, batchID).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 1, nil
	}
	return max.Int64 + 1, nil
}

// LatestVersionNo 当前最大版本号；没有版本返回 0。
func (s *Store) LatestVersionNo(batchID int64) (int64, error) {
	var max sql.NullInt64
	err := s.db.QueryRowContext(context.Background(),
		`SELECT MAX(version) FROM schedule_versions WHERE batch_id=?`, batchID).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 0, nil
	}
	return max.Int64, nil
}

// InsertVersion 写入播出表版本行。
func (s *Store) InsertVersion(v *model.ScheduleVersion) error {
	if v.CreatedAt == "" {
		v.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	sealed := 0
	if v.Sealed {
		sealed = 1
	}
	res, err := s.db.ExecContext(context.Background(),
		`INSERT INTO schedule_versions(batch_id,version,status,sealed,payload,content_hash,created_at)
		 VALUES(?,?,?,?,?,?,?)`,
		v.BatchID, v.Version, v.Status, sealed, v.Payload, v.ContentHash, v.CreatedAt)
	if err != nil {
		return wrapUnique(err, model.ErrVersionConflict)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	v.ID = id
	return nil
}

// GetVersionByNo 按批次+版本序号取存库行，原样返回 payload。
func (s *Store) GetVersionByNo(batchID, versionNo int64) (*model.ScheduleVersion, error) {
	v, err := scanVersion(s.db.QueryRowContext(context.Background(),
		`SELECT id,batch_id,version,status,sealed,payload,content_hash,created_at
		 FROM schedule_versions WHERE batch_id=? AND version=?`, batchID, versionNo))
	if err != nil {
		if errors.Is(err, model.ErrBatchNotFound) {
			return nil, fmt.Errorf("%w", model.ErrVersionNotFound)
		}
		return nil, err
	}
	return v, nil
}

// GetVersion 按主键取版本，原样返回存库 payload，不重算。
func (s *Store) GetVersion(id int64) (*model.ScheduleVersion, error) {
	return scanVersion(s.db.QueryRowContext(context.Background(),
		`SELECT id,batch_id,version,status,sealed,payload,content_hash,created_at
		 FROM schedule_versions WHERE id=?`, id))
}

// GetVersionInBatch 按批次+版本行主键取版本，防止跨批次读取。
func (s *Store) GetVersionInBatch(batchID, versionID int64) (*model.ScheduleVersion, error) {
	return scanVersion(s.db.QueryRowContext(context.Background(),
		`SELECT id,batch_id,version,status,sealed,payload,content_hash,created_at
		 FROM schedule_versions WHERE id=? AND batch_id=?`, versionID, batchID))
}

// GetVersionInBatchViaTx 事务内按批次+主键取版本。
func GetVersionInBatchViaTx(tx *sql.Tx, batchID, versionID int64) (*model.ScheduleVersion, error) {
	return scanVersion(tx.QueryRowContext(context.Background(),
		`SELECT id,batch_id,version,status,sealed,payload,content_hash,created_at
		 FROM schedule_versions WHERE id=? AND batch_id=?`, versionID, batchID))
}

func scanVersion(row *sql.Row) (*model.ScheduleVersion, error) {
	var v model.ScheduleVersion
	var sealed int
	err := row.Scan(&v.ID, &v.BatchID, &v.Version, &v.Status, &sealed, &v.Payload, &v.ContentHash, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w", model.ErrBatchNotFound)
	}
	if err != nil {
		return nil, err
	}
	v.Sealed = sealed != 0
	return &v, nil
}

// ListVersions 列出批次版本（含 payload，Get 依赖存库字节）。
func (s *Store) ListVersions(batchID int64) ([]model.ScheduleVersion, error) {
	rows, err := s.db.QueryContext(context.Background(),
		`SELECT id,batch_id,version,status,sealed,payload,content_hash,created_at
		 FROM schedule_versions WHERE batch_id=? ORDER BY version`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ScheduleVersion
	for rows.Next() {
		var v model.ScheduleVersion
		var sealed int
		if err := rows.Scan(&v.ID, &v.BatchID, &v.Version, &v.Status, &sealed, &v.Payload, &v.ContentHash, &v.CreatedAt); err != nil {
			return nil, err
		}
		v.Sealed = sealed != 0
		out = append(out, v)
	}
	return out, rows.Err()
}

// FreezeVersionTx 把指定版本写成冻结 payload，并把同批旧 frozen 标为 superseded。
func FreezeVersionTx(tx *sql.Tx, batchID, versionID int64, payload, hash string) error {
	if _, err := tx.ExecContext(context.Background(),
		`UPDATE schedule_versions SET status=? WHERE batch_id=? AND status=?`,
		model.VersionSuperseded, batchID, model.VersionFrozen); err != nil {
		return err
	}
	res, err := tx.ExecContext(context.Background(),
		`UPDATE schedule_versions SET status=?, sealed=1, payload=?, content_hash=? WHERE id=? AND batch_id=?`,
		model.VersionFrozen, payload, hash, versionID, batchID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w", model.ErrBatchNotFound)
	}
	return nil
}

// CountFrozenVersions 统计冻结版本数。
func (s *Store) CountFrozenVersions() (int, error) {
	var n int
	err := s.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM schedule_versions WHERE status=?`, model.VersionFrozen).Scan(&n)
	return n, err
}
