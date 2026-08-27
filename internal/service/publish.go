package service

import (
	"context"
	"database/sql"
	"time"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/snapshot"
	"task278-broadcastslot/internal/store"
	"task278-broadcastslot/internal/verdict"
)

// RecordVerdict 持 serialMu，按 expected_version 乐观锁确认或否决归属。
func (svc *Service) RecordVerdict(batchID, entryID int64, decision, reviewer, note string, expectedVersion int64) (*model.SlotVerdict, error) {
	svc.serialMu.Lock()
	defer svc.serialMu.Unlock()
	b, err := svc.requireBatch(batchID)
	if err != nil {
		return nil, err
	}
	if err := svc.rejectSealed(b); err != nil {
		return nil, err
	}
	current, err := svc.store.LatestVersionNo(batchID)
	if err != nil {
		return nil, err
	}
	if err := verdict.Apply(current, expectedVersion, decision); err != nil {
		return nil, err
	}
	next, err := verdict.NextAttrStatus(decision)
	if err != nil {
		return nil, err
	}
	v := &model.SlotVerdict{
		BatchID: batchID, EntryID: entryID, Decision: decision,
		Reviewer: reviewer, Note: note, ExpectedVersion: expectedVersion,
	}
	if err := svc.store.UpsertVerdict(v); err != nil {
		return nil, err
	}
	if err := svc.store.UpdateAttributionStatus(batchID, entryID, next); err != nil {
		return nil, err
	}
	if decision == model.DecisionConfirmed {
		_ = svc.store.UpdateEntryStatus(entryID, model.EntryAligned)
	} else {
		_ = svc.store.UpdateEntryStatus(entryID, model.EntryExcluded)
	}
	return v, nil
}

// ListVerdicts 列出裁决。
func (svc *Service) ListVerdicts(batchID int64) ([]model.SlotVerdict, error) {
	if _, err := svc.requireBatch(batchID); err != nil {
		return nil, err
	}
	return svc.store.ListVerdicts(batchID)
}

// BuildVersion 持 serialMu，创建草稿版本。
func (svc *Service) BuildVersion(batchID int64) (*model.ScheduleVersion, error) {
	svc.serialMu.Lock()
	defer svc.serialMu.Unlock()
	b, err := svc.requireBatch(batchID)
	if err != nil {
		return nil, err
	}
	if err := svc.rejectSealed(b); err != nil {
		return nil, err
	}
	n, err := svc.store.NextVersionNo(batchID)
	if err != nil {
		return nil, err
	}
	v := &model.ScheduleVersion{
		BatchID: batchID, Version: n, Status: model.VersionDraft,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := svc.store.InsertVersion(v); err != nil {
		return nil, err
	}
	return v, nil
}

// ListVersions 列出播出表版本。
func (svc *Service) ListVersions(batchID int64) ([]model.ScheduleVersion, error) {
	if _, err := svc.requireBatch(batchID); err != nil {
		return nil, err
	}
	return svc.store.ListVersions(batchID)
}

// PublishVersion 持 serialMu：深拷贝现场冻结到 payload，旧 frozen 标 superseded。
func (svc *Service) PublishVersion(ctx context.Context, batchID, versionNo int64) (*model.ScheduleVersion, error) {
	ctx = context.Background()
	svc.serialMu.Lock()
	defer svc.serialMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b, err := svc.requireBatch(batchID)
	if err != nil {
		return nil, err
	}
	if err := svc.rejectSealed(b); err != nil {
		return nil, err
	}
	target, err := svc.store.GetVersionByNo(batchID, versionNo)
	if err != nil {
		return nil, err
	}
	var published *model.ScheduleVersion
	err = svc.store.WithTx(ctx, func(tx *sql.Tx) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		existing, err := store.GetVersionInBatchViaTx(tx, batchID, target.ID)
		if err != nil {
			return err
		}
		if existing.Frozen() {
			return model.ErrFrozenVersion
		}
		live, err := loadLivePayload(tx, batchID)
		if err != nil {
			return err
		}
		raw, hash, err := snapshot.Freeze(live)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := store.FreezeVersionTx(tx, batchID, target.ID, string(raw), hash); err != nil {
			return err
		}
		if b.Status == model.BatchPendingVerdict || b.Status == model.BatchPendingAlign {
			if err := store.UpdateBatchStatusTx(tx, batchID, model.BatchPublished, b.SealedAt); err != nil {
				return err
			}
		}
		published, err = store.GetVersionInBatchViaTx(tx, batchID, target.ID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return published, nil
}

func loadLivePayload(tx *sql.Tx, batchID int64) (snapshot.Payload, error) {
	entries, err := store.ListEntriesTx(tx, batchID)
	if err != nil {
		return snapshot.Payload{}, err
	}
	attrs, err := store.ListAttributionsTx(tx, batchID)
	if err != nil {
		return snapshot.Payload{}, err
	}
	confs, err := store.ListConflictsTx(tx, batchID)
	if err != nil {
		return snapshot.Payload{}, err
	}
	verts, err := store.ListVerdictsTx(tx, batchID)
	if err != nil {
		return snapshot.Payload{}, err
	}
	cites, err := store.ListCitationsTx(tx, batchID)
	if err != nil {
		return snapshot.Payload{}, err
	}
	return snapshot.Payload{
		Entries: entries, Attributions: attrs, Conflicts: confs,
		Verdicts: verts, Citations: cites,
	}, nil
}
