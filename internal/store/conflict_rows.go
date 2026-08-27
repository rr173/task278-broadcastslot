package store

import (
	"context"
	"database/sql"

	"task278-broadcastslot/internal/model"
)

// DeleteConflictsTx 删除批次全部冲突行。须在写之前关闭未完成的 rows。
func DeleteConflictsTx(tx *sql.Tx, batchID int64) error {
	_, err := tx.ExecContext(context.Background(),
		`DELETE FROM attribution_conflicts WHERE batch_id=?`, batchID)
	return err
}

// InsertConflictTx 写入一条冲突。
func InsertConflictTx(tx *sql.Tx, c *model.AttributionConflict) error {
	res, err := tx.ExecContext(context.Background(),
		`INSERT INTO attribution_conflicts(batch_id,left_entry_id,right_entry_id,kind,detail)
		 VALUES(?,?,?,?,?)`,
		c.BatchID, c.LeftEntryID, c.RightEntryID, c.Kind, c.Detail)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = id
	return nil
}

// ListConflicts 列出批次冲突。
func (s *Store) ListConflicts(batchID int64) ([]model.AttributionConflict, error) {
	return listConflicts(s.db, batchID)
}

// ListConflictsTx 事务内列出冲突。
func ListConflictsTx(tx *sql.Tx, batchID int64) ([]model.AttributionConflict, error) {
	return listConflicts(tx, batchID)
}

func listConflicts(ex execer, batchID int64) ([]model.AttributionConflict, error) {
	rows, err := ex.QueryContext(context.Background(),
		`SELECT id,batch_id,left_entry_id,right_entry_id,kind,detail
		 FROM attribution_conflicts WHERE batch_id=? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	found := make([]model.AttributionConflict, 0)
	for rows.Next() {
		var c model.AttributionConflict
		if err := rows.Scan(&c.ID, &c.BatchID, &c.LeftEntryID, &c.RightEntryID, &c.Kind, &c.Detail); err != nil {
			return nil, err
		}
		found = append(found, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return found, nil
}
