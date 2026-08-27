package store

import (
	"context"
	"database/sql"

	"task278-broadcastslot/internal/model"
)

// InsertAd 写入报纸广告。(batch_id, ad_no) 唯一。
func (s *Store) InsertAd(a *model.NewspaperAd) error {
	res, err := s.db.ExecContext(context.Background(),
		`INSERT INTO newspaper_ads(batch_id,ad_no,printed_start_ms,page_id,edition,note)
		 VALUES(?,?,?,?,?,?)`,
		a.BatchID, a.AdNo, a.PrintedStartMS, a.PageID, a.Edition, a.Note)
	if err != nil {
		return wrapUnique(err, model.ErrDuplicateCode)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = id
	return nil
}

// ListAds 列出批次内广告，按 printed_start 升序以便取最早参考钟。
func (s *Store) ListAds(batchID int64) ([]model.NewspaperAd, error) {
	return listAds(s.db, batchID)
}

// ListAdsTx 事务内列出广告。
func ListAdsTx(tx *sql.Tx, batchID int64) ([]model.NewspaperAd, error) {
	return listAds(tx, batchID)
}

func listAds(ex execer, batchID int64) ([]model.NewspaperAd, error) {
	rows, err := ex.QueryContext(context.Background(),
		`SELECT id,batch_id,ad_no,printed_start_ms,page_id,edition,note
		 FROM newspaper_ads WHERE batch_id=? ORDER BY printed_start_ms, ad_no`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]model.NewspaperAd, 0)
	for rows.Next() {
		var a model.NewspaperAd
		if err := rows.Scan(&a.ID, &a.BatchID, &a.AdNo, &a.PrintedStartMS, &a.PageID, &a.Edition, &a.Note); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}
