package store

import (
	"context"

	"task278-broadcastslot/internal/model"
)

// Stats 汇总各表行数。每条 COUNT 用 QueryRow，无需 rows.Close。
func (s *Store) Stats() (*model.Stats, error) {
	st := &model.Stats{}
	pairs := []struct {
		dst   *int
		query string
	}{
		{&st.Batches, `SELECT COUNT(*) FROM evidence_batches`},
		{&st.Entries, `SELECT COUNT(*) FROM program_entries`},
		{&st.Clips, `SELECT COUNT(*) FROM station_clips`},
		{&st.Ads, `SELECT COUNT(*) FROM newspaper_ads`},
		{&st.Citations, `SELECT COUNT(*) FROM source_citations`},
		{&st.Corrections, `SELECT COUNT(*) FROM clock_corrections`},
		{&st.Attributions, `SELECT COUNT(*) FROM slot_attributions`},
		{&st.Conflicts, `SELECT COUNT(*) FROM attribution_conflicts`},
		{&st.Verdicts, `SELECT COUNT(*) FROM slot_verdicts`},
		{&st.Versions, `SELECT COUNT(*) FROM schedule_versions`},
	}
	for _, p := range pairs {
		if err := s.db.QueryRowContext(context.Background(), p.query).Scan(p.dst); err != nil {
			return nil, err
		}
	}
	n, err := s.CountSealed()
	if err != nil {
		return nil, err
	}
	st.SealedBatches = n
	f, err := s.CountFrozenVersions()
	if err != nil {
		return nil, err
	}
	st.FrozenVersions = f
	return st, nil
}
