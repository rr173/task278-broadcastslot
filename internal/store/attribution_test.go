package store_test

import (
	"context"
	"database/sql"
	"testing"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/store"
)

// TestListAttributionsIndependentAcrossBatches 回归：列 A 批归属后再列 B 批，
// A 的结果不能变成 B 的内容。曾因全局 attrScratch 复用同一底层数组而串台。
func TestListAttributionsIndependentAcrossBatches(t *testing.T) {
	st, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// 两批
	if err := st.InsertBatch(&model.EvidenceBatch{Code: "A", Status: model.BatchPendingVerdict}); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertBatch(&model.EvidenceBatch{Code: "B", Status: model.BatchPendingVerdict}); err != nil {
		t.Fatal(err)
	}

	// 各批一条归属
	insertAttr := func(batchID, entryID int64) error {
		return st.WithTx(context.Background(), func(tx *sql.Tx) error {
			return store.InsertAttributionTx(tx, &model.SlotAttribution{
				BatchID: batchID, EntryID: entryID, Status: model.AttrFeasible,
			})
		})
	}
	if err := insertAttr(1, 11); err != nil {
		t.Fatal(err)
	}
	if err := insertAttr(2, 22); err != nil {
		t.Fatal(err)
	}

	a, err := st.ListAttributions(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 || a[0].EntryID != 11 {
		t.Fatalf("batch A: got %+v, want one attribution entry_id=11", a)
	}

	b, err := st.ListAttributions(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 1 || b[0].EntryID != 22 {
		t.Fatalf("batch B: got %+v, want one attribution entry_id=22", b)
	}

	// 关键断言：列出 B 之后，A 的结果不应被改写。
	if len(a) != 1 || a[0].EntryID != 11 {
		t.Fatalf("batch A mutated after listing B: got %+v, want entry_id=11", a)
	}
	if len(b) != 1 || b[0].EntryID != 22 {
		t.Fatalf("batch B: got %+v, want entry_id=22", b)
	}
}

// TestListConflictsIndependentAcrossBatches 同样回归冲突列表全局缓冲复用。
func TestListConflictsIndependentAcrossBatches(t *testing.T) {
	st, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if err := st.InsertBatch(&model.EvidenceBatch{Code: "A", Status: model.BatchPendingVerdict}); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertBatch(&model.EvidenceBatch{Code: "B", Status: model.BatchPendingVerdict}); err != nil {
		t.Fatal(err)
	}

	insertConflict := func(batchID, leftID int64) error {
		return st.WithTx(context.Background(), func(tx *sql.Tx) error {
			return store.InsertConflictTx(tx, &model.AttributionConflict{
				BatchID: batchID, LeftEntryID: leftID, Kind: model.ConflictCallsignOverlap,
			})
		})
	}
	if err := insertConflict(1, 11); err != nil {
		t.Fatal(err)
	}
	if err := insertConflict(2, 22); err != nil {
		t.Fatal(err)
	}

	a, err := st.ListConflicts(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 || a[0].LeftEntryID != 11 {
		t.Fatalf("batch A: got %+v, want one conflict left_entry_id=11", a)
	}

	b, err := st.ListConflicts(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 1 || b[0].LeftEntryID != 22 {
		t.Fatalf("batch B: got %+v, want left_entry_id=22", b)
	}

	if len(a) != 1 || a[0].LeftEntryID != 11 {
		t.Fatalf("batch A mutated after listing B: got %+v, want left_entry_id=11", a)
	}
}
