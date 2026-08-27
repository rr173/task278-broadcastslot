package service_test

import (
	"context"
	"errors"
	"testing"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/service"
	"task278-broadcastslot/internal/store"
)

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
