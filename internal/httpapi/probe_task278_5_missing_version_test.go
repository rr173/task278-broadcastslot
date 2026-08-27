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

func TestMissingVersionMapsNotFound(t *testing.T) {
	h := newProbeServer(t)
	id := createBatchHTTP(t, h, "VER-404")
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/batches/%d/versions/99", id), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d body=%s", rec.Code, rec.Body.String())
	}
}
