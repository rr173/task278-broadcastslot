package snapshot_test

import (
	"context"
	"testing"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/service"
	"task278-broadcastslot/internal/snapshot"
	"task278-broadcastslot/internal/store"
)

func TestFrozenPayloadUnchangedAfterLiveEdit(t *testing.T) {
	st, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)

	b := model.EvidenceBatch{Code: "SNAP-1", Station: "S", AirDate: "1952-01-01", Timezone: "UTC"}
	if err := svc.CreateBatch(&b); err != nil {
		t.Fatal(err)
	}
	e, err := svc.AddEntry(b.ID, "News", "X", 3600000, 7200000, "p1", "TX")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddAd(b.ID, 1, 3600000, "p1", "", ""); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := svc.Correct(ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Align(ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	draft, err := svc.BuildVersion(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := svc.PublishVersion(ctx, b.ID, draft.Version)
	if err != nil {
		t.Fatal(err)
	}
	savedPayload := frozen.Payload
	savedHash := frozen.ContentHash

	if err := svc.UpdateEntryTitle(b.ID, e.ID, "mutated title"); err != nil {
		t.Fatal(err)
	}

	v, err := svc.GetVersionByNo(b.ID, draft.Version)
	if err != nil {
		t.Fatal(err)
	}
	if v.Payload != savedPayload {
		t.Fatal("frozen payload changed after live edit")
	}
	if v.ContentHash != savedHash {
		t.Fatal("content_hash changed after live edit")
	}

	parsed, err := snapshot.Parse([]byte(savedPayload))
	if err != nil || len(parsed.Entries) == 0 {
		t.Fatalf("payload parse: %v len=%d", err, len(parsed.Entries))
	}
}
