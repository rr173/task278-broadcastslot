package service_test

import (
	"context"
	"errors"
	"testing"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/service"
	"task278-broadcastslot/internal/store"
)

// TestCorrectCancelRollsBackWholeBatch 钟差校正被取消后必须整批回滚，
// 库里不得残留任何半写成的校正行。这是取消语义的核心不变量。
//
// 用已取消的 ctx 调用 Correct：旧实现逐条用 context.Background() 提交，
// 会在返回取消错误的同时把整批校正留在库里（半写成的钟差）；新实现
// 把整批校正放进单一 ctx 事务，取消则零提交、状态不前进。
func TestCorrectCancelRollsBackWholeBatch(t *testing.T) {
	st, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)

	b := model.EvidenceBatch{Code: "CANCEL-1", Station: "S", AirDate: "1952-01-01", Timezone: "Asia/Shanghai", DriftPPM: 120}
	if err := svc.CreateBatch(&b); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := svc.AddEntry(b.ID, "新闻", "SH", int64(20+i)*3600000, int64(20+i)*3600000+30*60000, "p", "TX"); err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = svc.Correct(ctx, b.ID)
	if err == nil {
		t.Fatal("expected cancel error, got nil")
	}

	rows, err := svc.ListCorrections(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("cancel must roll back all corrections, got %d half-written rows", len(rows))
	}

	got, err := svc.GetBatch(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.BatchOrganizing {
		t.Fatalf("status must not advance on cancel, got %s", got.Status)
	}
}

func TestSealedBatchRejectsWrite(t *testing.T) {
	st, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)

	b := model.EvidenceBatch{Code: "SEAL-1", Station: "S", AirDate: "1952-01-01", Timezone: "UTC"}
	if err := svc.CreateBatch(&b); err != nil {
		t.Fatal(err)
	}
	_ = svc.TransitionStatus(b.ID, model.BatchPendingAlign)
	_ = svc.TransitionStatus(b.ID, model.BatchPendingVerdict)
	_ = svc.TransitionStatus(b.ID, model.BatchPublished)
	if err := svc.SealBatch(b.ID); err != nil {
		t.Fatal(err)
	}

	_, err = svc.AddEntry(b.ID, "x", "c", 1, 2, "p", "t")
	if !errors.Is(err, model.ErrSealed) {
		t.Fatalf("expected sealed error, got %v", err)
	}
	_, err = svc.Correct(context.Background(), b.ID)
	if !errors.Is(err, model.ErrSealed) {
		t.Fatalf("correct on sealed: %v", err)
	}
}
