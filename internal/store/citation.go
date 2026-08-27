package store

import (
	"context"
	"database/sql"

	"task278-broadcastslot/internal/model"
)

// InsertCitation 写入引用边。(batch_id, from_ref, to_ref) 唯一。
func (s *Store) InsertCitation(c *model.SourceCitation) error {
	res, err := s.db.ExecContext(context.Background(),
		`INSERT INTO source_citations(batch_id,from_ref,to_ref,kind) VALUES(?,?,?,?)`,
		c.BatchID, c.FromRef, c.ToRef, c.Kind)
	if err != nil {
		return wrapUnique(err, model.ErrSourceCycle)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = id
	return nil
}

// ListCitations 列出批次内引用边。
func (s *Store) ListCitations(batchID int64) ([]model.SourceCitation, error) {
	return listCitations(s.db, batchID)
}

// ListCitationsTx 事务内列出引用。
func ListCitationsTx(tx *sql.Tx, batchID int64) ([]model.SourceCitation, error) {
	return listCitations(tx, batchID)
}

func listCitations(ex execer, batchID int64) ([]model.SourceCitation, error) {
	rows, err := ex.QueryContext(context.Background(),
		`SELECT id,batch_id,from_ref,to_ref,kind FROM source_citations WHERE batch_id=? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var edges []model.SourceCitation
	for rows.Next() {
		var c model.SourceCitation
		if err := rows.Scan(&c.ID, &c.BatchID, &c.FromRef, &c.ToRef, &c.Kind); err != nil {
			return nil, err
		}
		edges = append(edges, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return edges, nil
}
