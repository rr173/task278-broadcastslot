package smoke

import (
	"context"
	"fmt"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/service"
	"task278-broadcastslot/internal/store"
)

const msHour = int64(3600000)

// Run 执行端到端 smoke 场景并校验关库重开。
func Run(dbPath string) error {
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	svc := service.New(st)

	batch := model.EvidenceBatch{
		Code: "R-1952-11-08", Station: "上海台", AirDate: "1952-11-08",
		Timezone: "Asia/Shanghai", DriftPPM: 120,
	}
	if err := svc.CreateBatch(&batch); err != nil {
		st.Close()
		return err
	}

	entryA, err := svc.AddEntry(batch.ID, "晚间新闻", "SH-RADIO", 20*msHour, 20*msHour+30*60000, "page-A", "TX-1")
	if err != nil {
		st.Close()
		return err
	}
	entryB, err := svc.AddEntry(batch.ID, "晚间新闻", "SH-RADIO", 20*msHour+2*60000, 20*msHour+32*60000, "page-B", "TX-1")
	if err != nil {
		st.Close()
		return err
	}
	_ = entryB

	if _, err := svc.AddClip(batch.ID, 1, "SH-RADIO", 0, "archive-reel-7"); err != nil {
		st.Close()
		return err
	}
	if _, err := svc.AddAd(batch.ID, 1, 20*msHour, "page-A", "", ""); err != nil {
		st.Close()
		return err
	}
	if _, err := svc.AddAd(batch.ID, 2, 20*msHour+14*60000, "page-B", "", ""); err != nil {
		st.Close()
		return err
	}

	if _, err := svc.AddCitation(batch.ID, model.RefClip(1), model.RefEntry(entryA.ID), "audio"); err != nil {
		st.Close()
		return err
	}
	if _, err := svc.AddCitation(batch.ID, model.RefClip(1), model.RefEntry(entryB.ID), "audio"); err != nil {
		st.Close()
		return err
	}

	ctx := context.Background()
	if _, err := svc.Correct(ctx, batch.ID); err != nil {
		st.Close()
		return fmt.Errorf("correct: %w", err)
	}
	alignRes, err := svc.Align(ctx, batch.ID)
	if err != nil {
		st.Close()
		return fmt.Errorf("align: %w", err)
	}

	adDelayedCount := 0
	for _, c := range alignRes.Conflicts {
		if c.Kind == model.ConflictAdDelayed {
			adDelayedCount++
		}
	}
	if adDelayedCount < 1 {
		st.Close()
		return fmt.Errorf("expected at least 1 ad_delayed conflict, got %d", adDelayedCount)
	}

	pageAFeasible := false
	for _, a := range alignRes.Attributions {
		if a.EntryID == entryA.ID && a.Status == model.AttrFeasible {
			pageAFeasible = true
		}
	}
	if !pageAFeasible {
		st.Close()
		return fmt.Errorf("page A entry should have feasible attribution")
	}

	if _, err := svc.RecordVerdict(batch.ID, entryA.ID, model.DecisionConfirmed, "smoke", "", 0); err != nil {
		st.Close()
		return fmt.Errorf("verdict: %w", err)
	}

	draft, err := svc.BuildVersion(batch.ID)
	if err != nil {
		st.Close()
		return fmt.Errorf("build version: %w", err)
	}
	frozen, err := svc.PublishVersion(ctx, batch.ID, draft.Version)
	if err != nil {
		st.Close()
		return fmt.Errorf("publish: %w", err)
	}
	if frozen.ContentHash == "" {
		st.Close()
		return fmt.Errorf("frozen content_hash empty")
	}
	savedHash := frozen.ContentHash
	conflictCount := len(alignRes.Conflicts)

	if err := st.Close(); err != nil {
		return err
	}

	st2, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st2.Close()
	svc2 := service.New(st2)

	v, err := svc2.GetVersionByNo(batch.ID, draft.Version)
	if err != nil {
		return fmt.Errorf("reopen get version: %w", err)
	}
	if v.ContentHash != savedHash {
		return fmt.Errorf("hash changed after reopen: %s vs %s", savedHash, v.ContentHash)
	}
	conflicts, err := svc2.ListConflicts(batch.ID)
	if err != nil {
		return err
	}
	if len(conflicts) < conflictCount {
		return fmt.Errorf("conflicts lost after reopen")
	}

	fmt.Println("SMOKE-TEST PASSED")
	return nil
}
