package store

import (
	"context"
	"database/sql"
	"fmt"

	"task278-broadcastslot/internal/model"
)

// InsertEntry 写入节目条目。指纹冲突包装 ErrDuplicateFingerprint。
func (s *Store) InsertEntry(e *model.ProgramEntry) error {
	res, err := s.db.ExecContext(context.Background(),
		`INSERT INTO program_entries(batch_id,fingerprint,title,callsign,printed_start_ms,printed_end_ms,page_id,transmitter,status)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		e.BatchID, e.Fingerprint, e.Title, e.Callsign, e.PrintedStartMS, e.PrintedEndMS, e.PageID, e.Transmitter, e.Status)
	if err != nil {
		if isUniqueErr(err) {
			return fmt.Errorf("store: duplicate fingerprint: %w", model.ErrDuplicateFingerprint)
		}
		return fmt.Errorf("store: insert entry: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = id
	return nil
}

// GetEntryByFingerprint 按批次+指纹取已有行；没有则 (nil, nil)。
func (s *Store) GetEntryByFingerprint(batchID int64, fp string) (*model.ProgramEntry, error) {
	row := s.db.QueryRowContext(context.Background(),
		`SELECT id,batch_id,fingerprint,title,callsign,printed_start_ms,printed_end_ms,page_id,transmitter,status
		 FROM program_entries WHERE batch_id=? AND fingerprint=?`, batchID, fp)
	e, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

func scanEntry(row *sql.Row) (*model.ProgramEntry, error) {
	var e model.ProgramEntry
	err := row.Scan(&e.ID, &e.BatchID, &e.Fingerprint, &e.Title, &e.Callsign,
		&e.PrintedStartMS, &e.PrintedEndMS, &e.PageID, &e.Transmitter, &e.Status)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ListEntries 列出批次内全部条目。
func (s *Store) ListEntries(batchID int64) ([]model.ProgramEntry, error) {
	return listEntries(s.db, batchID)
}

// ListEntriesTx 事务内列出条目。
func ListEntriesTx(tx *sql.Tx, batchID int64) ([]model.ProgramEntry, error) {
	return listEntries(tx, batchID)
}

func listEntries(ex execer, batchID int64) ([]model.ProgramEntry, error) {
	rows, err := ex.QueryContext(context.Background(),
		`SELECT id,batch_id,fingerprint,title,callsign,printed_start_ms,printed_end_ms,page_id,transmitter,status
		 FROM program_entries WHERE batch_id=? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ProgramEntry
	for rows.Next() {
		var e model.ProgramEntry
		if err := rows.Scan(&e.ID, &e.BatchID, &e.Fingerprint, &e.Title, &e.Callsign,
			&e.PrintedStartMS, &e.PrintedEndMS, &e.PageID, &e.Transmitter, &e.Status); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpdateEntryStatus 改条目状态。
func (s *Store) UpdateEntryStatus(entryID int64, status string) error {
	_, err := s.db.ExecContext(context.Background(),
		`UPDATE program_entries SET status=? WHERE id=?`, status, entryID)
	return err
}

// UpdateEntryStatusTx 事务内改条目状态。
func UpdateEntryStatusTx(tx *sql.Tx, entryID int64, status string) error {
	_, err := tx.ExecContext(context.Background(),
		`UPDATE program_entries SET status=? WHERE id=?`, status, entryID)
	return err
}

// UpdateEntryTitle 仅改标题（用于冻结后改 live 的测试与复核更正）。
func (s *Store) UpdateEntryTitle(entryID int64, title string) error {
	res, err := s.db.ExecContext(context.Background(),
		`UPDATE program_entries SET title=? WHERE id=?`, title, entryID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("entry %d not found", entryID)
	}
	return nil
}
