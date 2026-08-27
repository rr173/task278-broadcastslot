package service

import (
	"fmt"

	"task278-broadcastslot/internal/conflict"
	"task278-broadcastslot/internal/evidence"
	"task278-broadcastslot/internal/model"
)

// AddEntry 导入节目条目：计算指纹，同指纹同字段幂等返回已有行。
func (svc *Service) AddEntry(batchID int64, title, callsign string, printedStart, printedEnd int64, pageID, transmitter string) (*model.ProgramEntry, error) {
	b, err := svc.requireBatch(batchID)
	if err != nil {
		return nil, err
	}
	if err := svc.rejectSealed(b); err != nil {
		return nil, err
	}
	incoming := model.ProgramEntry{
		BatchID: batchID, Title: title, Callsign: callsign,
		PrintedStartMS: printedStart, PrintedEndMS: printedEnd,
		PageID: pageID, Transmitter: transmitter, Status: model.EntryRaw,
	}
	if err := evidence.Fill(&incoming); err != nil {
		return nil, fmt.Errorf("service: fill entry: %w", err)
	}
	existing, err := svc.store.GetEntryByFingerprint(batchID, incoming.Fingerprint)
	if err != nil {
		return nil, fmt.Errorf("service: lookup fingerprint: %w", err)
	}
	resolved, err := evidence.Resolve(existing, incoming)
	if err != nil {
		return nil, fmt.Errorf("service: resolve entry: %w", err)
	}
	if existing != nil && resolved.ID == existing.ID {
		return existing, nil
	}
	if err := svc.store.InsertEntry(resolved); err != nil {
		return nil, fmt.Errorf("service: add entry: %w", err)
	}
	return resolved, nil
}

// ListEntries 列出节目条目。
func (svc *Service) ListEntries(batchID int64) ([]model.ProgramEntry, error) {
	if _, err := svc.requireBatch(batchID); err != nil {
		return nil, err
	}
	return svc.store.ListEntries(batchID)
}

// AddClip 导入录音台呼片段。
func (svc *Service) AddClip(batchID, clipNo int64, callsign string, offsetMS int64, source string) (*model.StationClip, error) {
	b, err := svc.requireBatch(batchID)
	if err != nil {
		return nil, err
	}
	if err := svc.rejectSealed(b); err != nil {
		return nil, err
	}
	c := &model.StationClip{
		BatchID: batchID, ClipNo: clipNo, Callsign: callsign,
		OffsetMS: offsetMS, Source: source, Status: model.EntryRaw,
	}
	if err := svc.store.InsertClip(c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListClips 列出台呼片段。
func (svc *Service) ListClips(batchID int64) ([]model.StationClip, error) {
	if _, err := svc.requireBatch(batchID); err != nil {
		return nil, err
	}
	return svc.store.ListClips(batchID)
}

// AddAd 导入报纸广告时刻。
func (svc *Service) AddAd(batchID, adNo, printedStart int64, pageID, edition, note string) (*model.NewspaperAd, error) {
	b, err := svc.requireBatch(batchID)
	if err != nil {
		return nil, err
	}
	if err := svc.rejectSealed(b); err != nil {
		return nil, err
	}
	a := &model.NewspaperAd{
		BatchID: batchID, AdNo: adNo, PrintedStartMS: printedStart,
		PageID: pageID, Edition: edition, Note: note,
	}
	if err := svc.store.InsertAd(a); err != nil {
		return nil, err
	}
	return a, nil
}

// ListAds 列出报纸广告。
func (svc *Service) ListAds(batchID int64) ([]model.NewspaperAd, error) {
	if _, err := svc.requireBatch(batchID); err != nil {
		return nil, err
	}
	return svc.store.ListAds(batchID)
}

// AddCitation 登记来源互引；成环返回 ErrSourceCycle。
func (svc *Service) AddCitation(batchID int64, fromRef, toRef, kind string) (*model.SourceCitation, error) {
	b, err := svc.requireBatch(batchID)
	if err != nil {
		return nil, err
	}
	if err := svc.rejectSealed(b); err != nil {
		return nil, err
	}
	if _, _, err := model.ParseRef(fromRef); err != nil {
		return nil, err
	}
	if _, _, err := model.ParseRef(toRef); err != nil {
		return nil, err
	}
	existing, err := svc.store.ListCitations(batchID)
	if err != nil {
		return nil, err
	}
	if err := conflict.DetectCycle(existing, fromRef, toRef); err != nil {
		return nil, err
	}
	c := &model.SourceCitation{BatchID: batchID, FromRef: fromRef, ToRef: toRef, Kind: kind}
	if err := svc.store.InsertCitation(c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListCitations 列出引用边。
func (svc *Service) ListCitations(batchID int64) ([]model.SourceCitation, error) {
	if _, err := svc.requireBatch(batchID); err != nil {
		return nil, err
	}
	return svc.store.ListCitations(batchID)
}

// UpdateEntryTitle 改 live 条目标题（冻结版本 payload 不受影响）。
func (svc *Service) UpdateEntryTitle(batchID, entryID int64, title string) error {
	b, err := svc.requireBatch(batchID)
	if err != nil {
		return err
	}
	if err := svc.rejectSealed(b); err != nil {
		return err
	}
	return svc.store.UpdateEntryTitle(entryID, title)
}
