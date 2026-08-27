package service_test

import (
	"context"
	"testing"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/service"
	"task278-broadcastslot/internal/store"
)

// TestAlignDoesNotMutatePriorResult guards against the shared scratch buffer
// aliasing bug: aligning a second batch must not rewrite the attribution/
// conflict slices that a previous Align call already returned.
func TestAlignDoesNotMutatePriorResult(t *testing.T) {
	st, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	ctx := context.Background()

	// Batch A: one entry attributed to callsign SH-RADIO with clip 1.
	batchA := model.EvidenceBatch{Code: "ISO-A", Station: "S", AirDate: "1952-01-01", Timezone: "UTC"}
	if err := svc.CreateBatch(&batchA); err != nil {
		t.Fatal(err)
	}
	entryA, err := svc.AddEntry(batchA.ID, "晚间新闻", "SH-RADIO", 20*3600000, 20*3600000+1800000, "page-A", "TX-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddClip(batchA.ID, 1, "SH-RADIO", 0, "reel-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Correct(ctx, batchA.ID); err != nil {
		t.Fatal(err)
	}
	resA, err := svc.Align(ctx, batchA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(resA.Attributions) != 1 {
		t.Fatalf("batch A: want 1 attribution, got %d", len(resA.Attributions))
	}
	wantA := resA.Attributions[0]
	if wantA.EntryID != entryA.ID {
		t.Fatalf("batch A: attribution entry mismatch: want %d got %d", entryA.ID, wantA.EntryID)
	}

	// Batch B: a different entry attributed to a different callsign/clip, so
	// any aliasing of A's buffer by B's append is observable.
	batchB := model.EvidenceBatch{Code: "ISO-B", Station: "S", AirDate: "1952-01-02", Timezone: "UTC"}
	if err := svc.CreateBatch(&batchB); err != nil {
		t.Fatal(err)
	}
	entryB, err := svc.AddEntry(batchB.ID, "早间新闻", "BJ-RADIO", 8*3600000, 8*3600000+1800000, "page-B", "TX-2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddClip(batchB.ID, 1, "BJ-RADIO", 0, "reel-B"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Correct(ctx, batchB.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Align(ctx, batchB.ID); err != nil {
		t.Fatal(err)
	}

	// A's captured result must be untouched by B's alignment.
	if len(resA.Attributions) != 1 {
		t.Fatalf("batch A result mutated by batch B align: attributions len now %d", len(resA.Attributions))
	}
	gotA := resA.Attributions[0]
	if gotA != wantA {
		t.Fatalf("batch A attribution corrupted after batch B align:\nwant %+v\ngot  %+v",
			wantA, gotA)
	}
	if gotA.EntryID != entryA.ID {
		t.Errorf("batch A entry id corrupted: want %d got %d (batch B entry %d)",
			entryA.ID, gotA.EntryID, entryB.ID)
	}
}
