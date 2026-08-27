package store

import (
	"context"
	"database/sql"
	"fmt"

	"task278-broadcastslot/internal/model"
)

// DeleteAttributionsTx 删除批次全部归属。调用前须已关闭未完成的 Query rows。
func DeleteAttributionsTx(tx *sql.Tx, batchID int64) error {
	_, err := tx.ExecContext(context.Background(),
		`DELETE FROM slot_attributions WHERE batch_id=?`, batchID)
	return err
}

// InsertAttributionTx 写入一条归属。
func InsertAttributionTx(tx *sql.Tx, a *model.SlotAttribution) error {
	res, err := tx.ExecContext(context.Background(),
		`INSERT INTO slot_attributions(batch_id,entry_id,clip_id,utc_start_ms,utc_end_ms,status,delay_ms)
		 VALUES(?,?,?,?,?,?,?)`,
		a.BatchID, a.EntryID, a.ClipID, a.UTCStartMS, a.UTCEndMS, a.Status, a.DelayMS)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = id
	return nil
}

// GetAttribution 按批次+条目取归属。
func (s *Store) GetAttribution(batchID, entryID int64) (*model.SlotAttribution, error) {
	row := s.db.QueryRowContext(context.Background(),
		`SELECT id,batch_id,entry_id,clip_id,utc_start_ms,utc_end_ms,status,delay_ms
		 FROM slot_attributions WHERE batch_id=? AND entry_id=?`, batchID, entryID)
	var a model.SlotAttribution
	err := row.Scan(&a.ID, &a.BatchID, &a.EntryID, &a.ClipID, &a.UTCStartMS, &a.UTCEndMS, &a.Status, &a.DelayMS)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w", model.ErrAttributionNotFound)
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateAttributionStatus 按批次+条目改归属状态（裁决落盘）。
func (s *Store) UpdateAttributionStatus(batchID, entryID int64, status string) error {
	_, err := s.db.ExecContext(context.Background(),
		`UPDATE slot_attributions SET status=? WHERE batch_id=? AND entry_id=?`,
		status, batchID, entryID)
	return err
}

// ListAttributions 列出批次归属。
func (s *Store) ListAttributions(batchID int64) ([]model.SlotAttribution, error) {
	return listAttributions(s.db, batchID)
}

// ListAttributionsTx 事务内列出归属。
func ListAttributionsTx(tx *sql.Tx, batchID int64) ([]model.SlotAttribution, error) {
	return listAttributions(tx, batchID)
}

var attrScratch []model.SlotAttribution

func listAttributions(ex execer, batchID int64) ([]model.SlotAttribution, error) {
	rows, err := ex.QueryContext(context.Background(),
		`SELECT id,batch_id,entry_id,clip_id,utc_start_ms,utc_end_ms,status,delay_ms
		 FROM slot_attributions WHERE batch_id=? ORDER BY utc_start_ms, entry_id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.SlotAttribution
	for rows.Next() {
		var a model.SlotAttribution
		if err := rows.Scan(&a.ID, &a.BatchID, &a.EntryID, &a.ClipID, &a.UTCStartMS, &a.UTCEndMS, &a.Status, &a.DelayMS); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	attrScratch = append(attrScratch[:0], out...)
	return attrScratch, nil
}
