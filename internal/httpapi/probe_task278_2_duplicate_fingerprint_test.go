package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/service"
	"task278-broadcastslot/internal/store"
)

func newProbeServer(t *testing.T) http.Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "probe-http.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return NewServer(service.New(st)).Handler()
}

func createBatchHTTP(t *testing.T, h http.Handler, code string) int64 {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"code": code, "station": "S", "air_date": "1952-01-01", "timezone": "UTC"})
	req := httptest.NewRequest(http.MethodPost, "/api/batches", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create batch: %d %s", rec.Code, rec.Body.String())
	}
	var b model.EvidenceBatch
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	return b.ID
}

func TestDuplicateFingerprintMapsConflict(t *testing.T) {
	h := newProbeServer(t)
	id := createBatchHTTP(t, h, "FP-1")
	body1, _ := json.Marshal(map[string]any{
		"title": "晚间新闻", "callsign": "SH", "printed_start_ms": 1, "printed_end_ms": 2,
		"page_id": "p", "transmitter": "TX-A",
	})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/batches/%d/entries", id), bytes.NewReader(body1))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first entry: %d %s", rec.Code, rec.Body.String())
	}
	body2, _ := json.Marshal(map[string]any{
		"title": "晚间新闻", "callsign": "SH", "printed_start_ms": 1, "printed_end_ms": 2,
		"page_id": "p", "transmitter": "TX-B",
	})
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/batches/%d/entries", id), bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("want 409 got %d body=%s", rec2.Code, rec2.Body.String())
	}
}
