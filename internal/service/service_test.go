package service_test

import (
	"context"
	"errors"
	"testing"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/service"
	"task278-broadcastslot/internal/store"
)

func seedBatch(t *testing.T, svc *service.Service) *model.EvidenceBatch {
	t.Helper()
	b := model.EvidenceBatch{Code: "C-1", Station: "S", AirDate: "1952-01-01", Timezone: "UTC"}
	if err := svc.CreateBatch(&b); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddEntry(b.ID, "新闻", "SH-RADIO", 20*3600000, 20*3600000+30*60000, "page-A", "TX-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddAd(b.ID, 1, 20*3600000, "page-A", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddClip(b.ID, 1, "SH-RADIO", 0, "reel-1"); err != nil {
		t.Fatal(err)
	}
	return &b
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

// TestCorrectCancelledNoPersist：校正请求被取消时不得落盘校正、不得推进批次状态。
func TestCorrectCancelledNoPersist(t *testing.T) {
	st, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	b := seedBatch(t, svc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 先取消，模拟客户端在响应前断开
	_, err = svc.Correct(ctx, b.ID)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	got, err := svc.ListCorrections(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("cancelled correct should leave no corrections, got %d", len(got))
	}
	batch, err := svc.GetBatch(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Status != model.BatchOrganizing {
		t.Fatalf("cancelled correct should not advance status, got %s", batch.Status)
	}
}

// TestAlignCancelledNoPersist：对齐请求被取消时不得写回归属/冲突、不得推进批次状态。
func TestAlignCancelledNoPersist(t *testing.T) {
	st, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	b := seedBatch(t, svc)
	if _, err := svc.Correct(context.Background(), b.ID); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = svc.Align(ctx, b.ID)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	attrs, err := svc.ListAttributions(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(attrs) != 0 {
		t.Fatalf("cancelled align should leave no attributions, got %d", len(attrs))
	}
	batch, err := svc.GetBatch(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Status != model.BatchPendingAlign {
		t.Fatalf("cancelled align should not advance status, got %s", batch.Status)
	}
}
