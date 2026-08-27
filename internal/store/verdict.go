package store

import (
	"context"
	"database/sql"

	"task278-broadcastslot/internal/model"
)

// UpsertVerdict 按 (batch_id, entry_id) 写入或覆盖裁决。
func (s *Store) UpsertVerdict(v *model.SlotVerdict) error {
	res, err := s.db.ExecContext(context.Background(),
		`INSERT INTO slot_verdicts(batch_id,entry_id,decision,reviewer,note,expected_version)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(batch_id, entry_id) DO UPDATE SET
		   decision=excluded.decision,
		   reviewer=excluded.reviewer,
		   note=excluded.note,
		   expected_version=excluded.expected_version`,
		v.BatchID, v.EntryID, v.Decision, v.Reviewer, v.Note, v.ExpectedVersion)
	if err != nil {
		return err
	}
	if v.ID == 0 {
		id, err := res.LastInsertId()
		if err == nil {
			v.ID = id
		}
	}
	return nil
}

// ListVerdicts 列出批次裁决。
func (s *Store) ListVerdicts(batchID int64) ([]model.SlotVerdict, error) {
	return listVerdicts(s.db, batchID)
}

// ListVerdictsTx 事务内列出裁决。
func ListVerdictsTx(tx *sql.Tx, batchID int64) ([]model.SlotVerdict, error) {
	return listVerdicts(tx, batchID)
}

func listVerdicts(ex execer, batchID int64) ([]model.SlotVerdict, error) {
	rows, err := ex.QueryContext(context.Background(),
		`SELECT id,batch_id,entry_id,decision,reviewer,note,expected_version
		 FROM slot_verdicts WHERE batch_id=? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.SlotVerdict
	for rows.Next() {
		var v model.SlotVerdict
		if err := rows.Scan(&v.ID, &v.BatchID, &v.EntryID, &v.Decision, &v.Reviewer, &v.Note, &v.ExpectedVersion); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
