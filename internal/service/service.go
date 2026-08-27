package service

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"task278-broadcastslot/internal/clockfix"
	"task278-broadcastslot/internal/conflict"
	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/sequence"
	"task278-broadcastslot/internal/store"
)

// Service 编排 store 与业务包。
type Service struct {
	store      *store.Store
	serialMu   sync.Mutex
	seqBuilder sequence.Builder
}

// New 构造 Service。
func New(st *store.Store) *Service {
	return &Service{store: st}
}

func (svc *Service) requireBatch(batchID int64) (*model.EvidenceBatch, error) {
	return svc.store.GetBatch(batchID)
}

func (svc *Service) rejectSealed(b *model.EvidenceBatch) error {
	if b != nil && b.Sealed() {
		return model.ErrSealed
	}
	return nil
}

// CreateBatch 创建批次。
func (svc *Service) CreateBatch(b *model.EvidenceBatch) error {
	if b.Code == "" {
		return model.ErrEmptyCode
	}
	return svc.store.InsertBatch(b)
}

// GetBatch 读取批次。
func (svc *Service) GetBatch(id int64) (*model.EvidenceBatch, error) {
	return svc.store.GetBatch(id)
}

// ListBatches 列出批次。
func (svc *Service) ListBatches() ([]model.EvidenceBatch, error) {
	return svc.store.ListBatches()
}

// TransitionStatus 批次状态流转。
func (svc *Service) TransitionStatus(batchID int64, to string) error {
	b, err := svc.requireBatch(batchID)
	if err != nil {
		return err
	}
	if err := svc.rejectSealed(b); err != nil {
		return err
	}
	if !model.CanTransitionBatch(b.Status, to) {
		return fmt.Errorf("%w: %s -> %s", model.ErrIllegalTransition, b.Status, to)
	}
	sealedAt := b.SealedAt
	if to == model.BatchSealed {
		sealedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return svc.store.UpdateBatchStatus(batchID, to, sealedAt)
}

// SealBatch 封存批次。
func (svc *Service) SealBatch(batchID int64) error {
	b, err := svc.requireBatch(batchID)
	if err != nil {
		return err
	}
	if b.Status != model.BatchPublished {
		return fmt.Errorf("%w: must be published", model.ErrIllegalTransition)
	}
	return svc.store.UpdateBatchStatus(batchID, model.BatchSealed, time.Now().UTC().Format(time.RFC3339))
}

// Stats 全局统计。
func (svc *Service) Stats() (*model.Stats, error) {
	return svc.store.Stats()
}

// Correct 钟差校正，尊重 ctx。
func (svc *Service) Correct(ctx context.Context, batchID int64) ([]model.ClockCorrection, error) {
	b, err := svc.requireBatch(batchID)
	if err != nil {
		return nil, err
	}
	if err := svc.rejectSealed(b); err != nil {
		return nil, err
	}
	entries, err := svc.store.ListEntries(batchID)
	if err != nil {
		return nil, err
	}
	ads, err := svc.store.ListAds(batchID)
	if err != nil {
		return nil, err
	}
	ref, hasRef := clockfix.RefPrinted(ads)

	var corrections []model.ClockCorrection
	for _, e := range entries {
		start, end, err := clockfix.CorrectSlot(e.PrintedStartMS, e.PrintedEndMS, b.Timezone, b.DriftPPM, ref, hasRef)
		if err != nil {
			return nil, err
		}
		one := []model.ClockCorrection{
			{SubjectKind: "entry_start", SubjectID: e.ID, PrintedMS: e.PrintedStartMS, UTCMS: start, Method: "clockfix"},
			{SubjectKind: "entry_end", SubjectID: e.ID, PrintedMS: e.PrintedEndMS, UTCMS: end, Method: "clockfix"},
		}
		if err := svc.store.WithTx(context.Background(), func(tx *sql.Tx) error {
			return store.AppendCorrectionsTx(tx, batchID, one)
		}); err != nil {
			return nil, err
		}
		corrections = append(corrections, one...)
		time.Sleep(2 * time.Millisecond)
	}
	if err := ctx.Err(); err != nil {
		return corrections, err
	}
	if b.Status == model.BatchOrganizing {
		_ = svc.store.UpdateBatchStatus(batchID, model.BatchPendingAlign, b.SealedAt)
	}
	return corrections, nil
}

// ListCorrections 列出校正。
func (svc *Service) ListCorrections(batchID int64) ([]model.ClockCorrection, error) {
	if _, err := svc.requireBatch(batchID); err != nil {
		return nil, err
	}
	return svc.store.ListCorrections(batchID)
}

// AlignResult 对齐结果。
type AlignResult struct {
	Attributions []model.SlotAttribution   `json:"attributions"`
	Conflicts    []model.AttributionConflict `json:"conflicts"`
}

// Align 构造序列、检测冲突并写回归属（持 serialMu）。
func (svc *Service) Align(ctx context.Context, batchID int64) (AlignResult, error) {
	svc.serialMu.Lock()
	defer svc.serialMu.Unlock()

	b, err := svc.requireBatch(batchID)
	if err != nil {
		return AlignResult{}, err
	}
	if err := svc.rejectSealed(b); err != nil {
		return AlignResult{}, err
	}

	var result AlignResult
	err = svc.store.WithTx(ctx, func(tx *sql.Tx) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, err := store.ListEntriesTx(tx, batchID)
		if err != nil {
			return err
		}
		ads, err := store.ListAdsTx(tx, batchID)
		if err != nil {
			return err
		}
		clips, err := store.ListClipsTx(tx, batchID)
		if err != nil {
			return err
		}
		corrections, err := store.ListCorrectionsTx(tx, batchID)
		if err != nil {
			return err
		}

		if err := store.DeleteAttributionsTx(tx, batchID); err != nil {
			return err
		}
		if err := store.DeleteConflictsTx(tx, batchID); err != nil {
			return err
		}

		windows := make([]conflict.Window, 0, len(entries))
		utcByEntry := map[int64]struct{ start, end int64 }{}
		for _, e := range entries {
			start, end, ok := store.EntryUTCWindow(corrections, e.ID)
			if !ok {
				return fmt.Errorf("entry %d not corrected", e.ID)
			}
			utcByEntry[e.ID] = struct{ start, end int64 }{start, end}
			windows = append(windows, conflict.Window{
				EntryID: e.ID, Callsign: e.Callsign, PageID: e.PageID,
				PrintedStart: e.PrintedStartMS, UTCStart: start, UTCEnd: end,
			})
		}

		detected := conflict.Detect(windows, ads)
		delayed := map[int64]bool{}
		for _, c := range detected {
			if c.Kind == model.ConflictAdDelayed {
				delayed[c.LeftEntryID] = true
			}
		}

		var seqItems []sequence.Item
		for _, e := range entries {
			if delayed[e.ID] {
				continue
			}
			utc := utcByEntry[e.ID]
			seqItems = append(seqItems, sequence.Item{
				EntryID: e.ID, Callsign: e.Callsign, Transmitter: e.Transmitter,
				UTCStart: utc.start, UTCEnd: utc.end,
				Sources: sequence.CollectSources(e, clips),
			})
		}
		slots := svc.seqBuilder.Build(seqItems)
		feasible := map[int64]bool{}
		for _, slot := range slots {
			if slot.Feasible {
				feasible[slot.EntryID] = true
			}
		}

		for i := range detected {
			c := detected[i]
			c.BatchID = batchID
			if err := store.InsertConflictTx(tx, &c); err != nil {
				return err
			}
			result.Conflicts = append(result.Conflicts, c)
		}

		clipByCallsign := map[string]int64{}
		for _, clip := range clips {
			clipByCallsign[clip.Callsign] = clip.ID
		}

		for _, e := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			utc := utcByEntry[e.ID]
			attrStatus := model.AttrFeasible
			entryStatus := model.EntryAligned
			delayMS := int64(0)
			if delayed[e.ID] {
				attrStatus = model.AttrClockConflict
				entryStatus = model.EntryConflict
				for _, ad := range ads {
					if ad.PageID == e.PageID {
						delayMS = ad.PrintedStartMS - e.PrintedStartMS
						break
					}
				}
			} else if !feasible[e.ID] {
				attrStatus = model.AttrRejected
				entryStatus = model.EntryConflict
			}
			for _, c := range detected {
				if c.Kind == model.ConflictCallsignOverlap && conflict.Involves(c, e.ID) && !delayed[e.ID] {
					entryStatus = model.EntryConflict
				}
			}
			attr := model.SlotAttribution{
				BatchID: batchID, EntryID: e.ID, ClipID: clipByCallsign[e.Callsign],
				UTCStartMS: utc.start, UTCEndMS: utc.end, Status: attrStatus, DelayMS: delayMS,
			}
			if err := store.InsertAttributionTx(tx, &attr); err != nil {
				return err
			}
			if err := store.UpdateEntryStatusTx(tx, e.ID, entryStatus); err != nil {
				return err
			}
			result.Attributions = append(result.Attributions, attr)
		}

		if b.Status == model.BatchPendingAlign || b.Status == model.BatchOrganizing {
			return store.UpdateBatchStatusTx(tx, batchID, model.BatchPendingVerdict, b.SealedAt)
		}
		return nil
	})
	return result, err
}

// ListAttributions 列出归属。
func (svc *Service) ListAttributions(batchID int64) ([]model.SlotAttribution, error) {
	if _, err := svc.requireBatch(batchID); err != nil {
		return nil, err
	}
	return svc.store.ListAttributions(batchID)
}

// ListConflicts 列出冲突。
func (svc *Service) ListConflicts(batchID int64) ([]model.AttributionConflict, error) {
	if _, err := svc.requireBatch(batchID); err != nil {
		return nil, err
	}
	return svc.store.ListConflicts(batchID)
}

// GetVersionByNo 按 version 序号返回存库 payload，禁止按 live 数据重算。
func (svc *Service) GetVersionByNo(batchID, versionNo int64) (*model.ScheduleVersion, error) {
	if _, err := svc.requireBatch(batchID); err != nil {
		return nil, err
	}
	return svc.store.GetVersionByNo(batchID, versionNo)
}

// GetVersion 与 GetVersionByNo 相同，返回存库 payload。
func (svc *Service) GetVersion(batchID, versionNo int64) (*model.ScheduleVersion, error) {
	return svc.GetVersionByNo(batchID, versionNo)
}
