package store

import (
	"context"
	"database/sql"

	"task278-broadcastslot/internal/model"
)

// InsertClip 写入录音台呼。(batch_id, clip_no) 唯一。
func (s *Store) InsertClip(c *model.StationClip) error {
	if c.Status == "" {
		c.Status = "raw"
	}
	res, err := s.db.ExecContext(context.Background(),
		`INSERT INTO station_clips(batch_id,clip_no,callsign,offset_ms,source,status)
		 VALUES(?,?,?,?,?,?)`,
		c.BatchID, c.ClipNo, c.Callsign, c.OffsetMS, c.Source, c.Status)
	if err != nil {
		return wrapUnique(err, model.ErrDuplicateCode)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = id
	return nil
}

// ListClips 列出批次内台呼片段。
func (s *Store) ListClips(batchID int64) ([]model.StationClip, error) {
	return listClips(s.db, batchID)
}

// ListClipsTx 事务内列出台呼片段。
func ListClipsTx(tx *sql.Tx, batchID int64) ([]model.StationClip, error) {
	return listClips(tx, batchID)
}

func listClips(ex execer, batchID int64) ([]model.StationClip, error) {
	rows, err := ex.QueryContext(context.Background(),
		`SELECT id,batch_id,clip_no,callsign,offset_ms,source,status
		 FROM station_clips WHERE batch_id=? ORDER BY clip_no`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.StationClip, 0)
	for rows.Next() {
		var c model.StationClip
		if err := rows.Scan(&c.ID, &c.BatchID, &c.ClipNo, &c.Callsign, &c.OffsetMS, &c.Source, &c.Status); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
