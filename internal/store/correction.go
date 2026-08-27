package store

import (
	"context"
	"database/sql"
	"time"

	"task278-broadcastslot/internal/model"
)

// ReplaceCorrectionsTx 先关查询再删旧校正、写入新校正。调用方须已持有事务。
func ReplaceCorrectionsTx(tx *sql.Tx, batchID int64, rows []model.ClockCorrection) error {
	if _, err := tx.ExecContext(context.Background(),
		`DELETE FROM clock_corrections WHERE batch_id=?`, batchID); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range rows {
		if rows[i].AppliedAt == "" {
			rows[i].AppliedAt = now
		}
		res, err := tx.ExecContext(context.Background(),
			`INSERT INTO clock_corrections(batch_id,subject_kind,subject_id,printed_ms,utc_ms,method,applied_at)
			 VALUES(?,?,?,?,?,?,?)`,
			batchID, rows[i].SubjectKind, rows[i].SubjectID, rows[i].PrintedMS, rows[i].UTCMS, rows[i].Method, rows[i].AppliedAt)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		rows[i].ID = id
		rows[i].BatchID = batchID
	}
	return nil
}

// AppendCorrectionsTx 立刻追加校正行，不删除旧行。
func AppendCorrectionsTx(tx *sql.Tx, batchID int64, rows []model.ClockCorrection) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range rows {
		if rows[i].AppliedAt == "" {
			rows[i].AppliedAt = now
		}
		res, err := tx.ExecContext(context.Background(),
			`INSERT INTO clock_corrections(batch_id,subject_kind,subject_id,printed_ms,utc_ms,method,applied_at)
			 VALUES(?,?,?,?,?,?,?)`,
			batchID, rows[i].SubjectKind, rows[i].SubjectID, rows[i].PrintedMS, rows[i].UTCMS, rows[i].Method, rows[i].AppliedAt)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		rows[i].ID = id
		rows[i].BatchID = batchID
	}
	return nil
}

// ListCorrections 列出批次内全部校正。
func (s *Store) ListCorrections(batchID int64) ([]model.ClockCorrection, error) {
	return listCorrections(s.db, batchID)
}

// ListCorrectionsTx 事务内列出校正。
func ListCorrectionsTx(tx *sql.Tx, batchID int64) ([]model.ClockCorrection, error) {
	return listCorrections(tx, batchID)
}

func listCorrections(ex execer, batchID int64) ([]model.ClockCorrection, error) {
	q, err := ex.QueryContext(context.Background(),
		`SELECT id,batch_id,subject_kind,subject_id,printed_ms,utc_ms,method,applied_at
		 FROM clock_corrections WHERE batch_id=? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer q.Close()
	var out []model.ClockCorrection
	for q.Next() {
		var c model.ClockCorrection
		if err := q.Scan(&c.ID, &c.BatchID, &c.SubjectKind, &c.SubjectID, &c.PrintedMS, &c.UTCMS, &c.Method, &c.AppliedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := q.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// EntryUTCWindow 从 entry_start / entry_end 校正行取出播出 UTC 窗口。
func EntryUTCWindow(rows []model.ClockCorrection, entryID int64) (start, end int64, ok bool) {
	var gotStart, gotEnd bool
	for _, r := range rows {
		if r.SubjectID != entryID {
			continue
		}
		switch r.SubjectKind {
		case "entry_start":
			start = r.UTCMS
			gotStart = true
		case "entry_end":
			end = r.UTCMS
			gotEnd = true
		}
	}
	return start, end, gotStart && gotEnd
}
