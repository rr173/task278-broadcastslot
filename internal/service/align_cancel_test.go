package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/service"
	"task278-broadcastslot/internal/store"
)

// seedAlignedBatch 建批次→条目→广告→校正→对齐，返回已落盘归属数。
// 用 ctx2/onCancel 模拟第二次对齐在删除归属后、写回前被取消。
func seedAlignedBatch(t *testing.T, svc *service.Service, batchID int64) {
	t.Helper()
	ctx := context.Background()
	if _, err := svc.AddEntry(batchID, "News", "X", 3600000, 7200000, "p1", "TX"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddAd(batchID, 1, 3600000, "p1", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Correct(ctx, batchID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Align(ctx, batchID); err != nil {
		t.Fatal(err)
	}
}

// TestAlignCancelRestoresAttributions 复现：对齐被取消时归属表被清空却未写回。
// 修复前 WithTx 在 fn 报错时 Commit 而非 Rollback，把 DeleteAttributionsTx
// 落盘，ListAttributions 返回空表。修复后回滚，归属恢复到对齐前。
func TestAlignCancelRestoresAttributions(t *testing.T) {
	st, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)

	b := model.EvidenceBatch{Code: "CANCEL-1", Station: "S", AirDate: "1952-01-01", Timezone: "UTC"}
	if err := svc.CreateBatch(&b); err != nil {
		t.Fatal(err)
	}
	seedAlignedBatch(t, svc, b.ID)

	before, err := svc.ListAttributions(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) == 0 {
		t.Fatal("precondition: expected attributions from first align")
	}

	// 第二次对齐：DeleteAttributionsTx 后有 25ms sleep 再 ctx 检查，
	// 在此处取消使 fn 返回 ctx.Err()，触发 WithTx 的回滚路径。
	ctx, cancel := context.WithCancel(context.Background())
	// Align 内 DeleteAttributionsTx→DeleteConflictsTx→time.Sleep(25ms)→ctx.Err()。
	// 给足时间让删除先发生，再取消命中 sleep 之后的检查点。
	go func() {
		time.Sleep(15 * time.Millisecond)
		cancel()
	}()
	_, err = svc.Align(ctx, b.ID)
	if err == nil {
		t.Fatal("expected align to be cancelled, got nil error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	after, err := svc.ListAttributions(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("attributions lost after cancelled align: before=%d after=%d (空表被落盘)",
			len(before), len(after))
	}
}
