package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"task278-broadcastslot/internal/model"
)

// InsertBatch 写入新批次。code 冲突包装为 ErrDuplicateCode。
func (s *Store) InsertBatch(b *model.EvidenceBatch) error {
	if b.CreatedAt == "" {
		b.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if b.Status == "" {
		b.Status = model.BatchOrganizing
	}
	res, err := s.db.ExecContext(context.Background(),
		`INSERT INTO evidence_batches(code,station,air_date,timezone,drift_ppm,status,created_at,sealed_at)
		 VALUES(?,?,?,?,?,?,?,?)`,
		b.Code, b.Station, b.AirDate, b.Timezone, b.DriftPPM, b.Status, b.CreatedAt, b.SealedAt)
	if err != nil {
		return wrapUnique(err, model.ErrDuplicateCode)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	b.ID = id
	return nil
}

// GetBatch 按主键取批次；缺失包装 ErrBatchNotFound。
func (s *Store) GetBatch(id int64) (*model.EvidenceBatch, error) {
	return scanBatch(s.db.QueryRowContext(context.Background(),
		`SELECT id,code,station,air_date,timezone,drift_ppm,status,created_at,sealed_at
		 FROM evidence_batches WHERE id=?`, id))
}

// GetBatchTx 在事务内取批次。
func GetBatchTx(tx *sql.Tx, id int64) (*model.EvidenceBatch, error) {
	return scanBatch(tx.QueryRowContext(context.Background(),
		`SELECT id,code,station,air_date,timezone,drift_ppm,status,created_at,sealed_at
		 FROM evidence_batches WHERE id=?`, id))
}

func scanBatch(row *sql.Row) (*model.EvidenceBatch, error) {
	var b model.EvidenceBatch
	err := row.Scan(&b.ID, &b.Code, &b.Station, &b.AirDate, &b.Timezone, &b.DriftPPM, &b.Status, &b.CreatedAt, &b.SealedAt)
	if err != nil {
		return nil, notFoundBatch(err)
	}
	return &b, nil
}

// ListBatches 列出全部批次，按 id 升序。
func (s *Store) ListBatches() ([]model.EvidenceBatch, error) {
	rows, err := s.db.QueryContext(context.Background(),
		`SELECT id,code,station,air_date,timezone,drift_ppm,status,created_at,sealed_at
		 FROM evidence_batches ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.EvidenceBatch
	for rows.Next() {
		var b model.EvidenceBatch
		if err := rows.Scan(&b.ID, &b.Code, &b.Station, &b.AirDate, &b.Timezone, &b.DriftPPM, &b.Status, &b.CreatedAt, &b.SealedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateBatchStatus 更新状态；封存时写入 sealed_at。
func (s *Store) UpdateBatchStatus(id int64, status, sealedAt string) error {
	res, err := s.db.ExecContext(context.Background(),
		`UPDATE evidence_batches SET status=?, sealed_at=? WHERE id=?`, status, sealedAt, id)
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

// UpdateBatchStatusTx 事务内更新状态。
func UpdateBatchStatusTx(tx *sql.Tx, id int64, status, sealedAt string) error {
	res, err := tx.ExecContext(context.Background(),
		`UPDATE evidence_batches SET status=?, sealed_at=? WHERE id=?`, status, sealedAt, id)
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

// CountSealed 统计封存批次数，供 Stats 使用。
func (s *Store) CountSealed() (int, error) {
	var n int
	err := s.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM evidence_batches WHERE status=?`, model.BatchSealed).Scan(&n)
	return n, err
}
