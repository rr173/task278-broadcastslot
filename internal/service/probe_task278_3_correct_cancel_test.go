package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/snapshot"
	"task278-broadcastslot/internal/store"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "probe.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(st)
}

func mustBatch(t *testing.T, svc *Service, code string, n int) (*model.EvidenceBatch, []int64) {
	t.Helper()
	b := model.EvidenceBatch{Code: code, Station: "S", AirDate: "1952-11-08", Timezone: "UTC", DriftPPM: 0}
	if err := svc.CreateBatch(&b); err != nil {
		t.Fatal(err)
	}
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		start := int64(i) * 3600000
		e, err := svc.AddEntry(b.ID, fmt.Sprintf("slot-%s-%d", code, i), fmt.Sprintf("CS-%d", i), start, start+1800000, fmt.Sprintf("p-%d", i), "TX")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, e.ID)
		if _, err := svc.AddAd(b.ID, int64(i+1), start, fmt.Sprintf("p-%d", i), "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Correct(context.Background(), b.ID); err != nil {
		t.Fatal(err)
	}
	return &b, ids
}

func payloadAttrCount(t *testing.T, raw string) int {
	t.Helper()
	var p snapshot.Payload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	return len(p.Attributions)
}

func payloadVerdicts(t *testing.T, raw string) []model.SlotVerdict {
	t.Helper()
	var p snapshot.Payload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	return p.Verdicts
}

func payloadAttrs(t *testing.T, raw string) []model.SlotAttribution {
	t.Helper()
	var p snapshot.Payload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	return p.Attributions
}

func TestCorrectCancelRollsBack(t *testing.T) {
	svc := newTestService(t)
	b := model.EvidenceBatch{Code: "CANCEL-CORR", Station: "S", AirDate: "1952-01-01", Timezone: "UTC"}
	if err := svc.CreateBatch(&b); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i++ {
		start := int64(i+1) * 3600000
		if _, err := svc.AddEntry(b.ID, fmt.Sprintf("e-%d", i), "CS", start, start+60000, fmt.Sprintf("p%d", i), "TX"); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.AddAd(b.ID, int64(i+1), start, fmt.Sprintf("p%d", i), "", ""); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = svc.Correct(ctx, b.ID)
	rows, err := svc.ListCorrections(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("cancel must roll back all corrections, got %d", len(rows))
	}
}
