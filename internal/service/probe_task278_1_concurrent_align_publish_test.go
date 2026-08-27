package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
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

func TestConcurrentAlignAndPublishIsolation(t *testing.T) {
	svc := newTestService(t)
	b, _ := mustBatch(t, svc, "C-ALIGN-PUB", 4)
	if _, err := svc.Align(context.Background(), b.ID); err != nil {
		t.Fatal(err)
	}
	const workers = 20
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				if _, err := svc.Align(context.Background(), b.ID); err != nil {
					errCh <- fmt.Errorf("align %d: %w", i, err)
				}
				return
			}
			draft, err := svc.BuildVersion(b.ID)
			if err != nil {
				errCh <- fmt.Errorf("draft %d: %w", i, err)
				return
			}
			frozen, err := svc.PublishVersion(context.Background(), b.ID, draft.Version)
			if err != nil {
				errCh <- fmt.Errorf("publish %d: %w", i, err)
				return
			}
			if payloadAttrCount(t, frozen.Payload) == 0 {
				errCh <- fmt.Errorf("publish %d version %d has zero attributions", i, frozen.Version)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}
