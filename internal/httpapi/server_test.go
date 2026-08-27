package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"task278-broadcastslot/internal/httpapi"
	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/service"
	"task278-broadcastslot/internal/store"
)

func newTestServer(t *testing.T) *httpapi.Server {
	t.Helper()
	st, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return httpapi.NewServer(service.New(st))
}

func TestDuplicateCode409(t *testing.T) {
	srv := newTestServer(t)
	body, _ := json.Marshal(map[string]any{
		"code": "DUP-1", "station": "S", "air_date": "1952-01-01", "timezone": "UTC",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/batches", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first create: %d", rec.Code)
	}
	req2 := httptest.NewRequest(http.MethodPost, "/api/batches", bytes.NewReader(body))
	rec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("duplicate code want 409 got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestDuplicateFingerprintDifferentTransmitter409(t *testing.T) {
	srv := newTestServer(t)
	batch, _ := json.Marshal(map[string]any{
		"code": "ENT-1", "station": "S", "air_date": "1952-01-01", "timezone": "UTC",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/batches", bytes.NewReader(batch))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create batch: %d body=%s", rec.Code, rec.Body.String())
	}
	var b struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &b)

	// 同指纹（title/callsign/printed_start/end/page_id 全同），发射机不同 → 冲突，而非内部错误。
	entry := map[string]any{
		"title": "晚间新闻", "callsign": "SH-RADIO",
		"printed_start_ms": 72000000, "printed_end_ms": 73800000,
		"page_id": "page-A", "transmitter": "TX-A",
	}
	body, _ := json.Marshal(entry)
	req1 := httptest.NewRequest(http.MethodPost, "/api/batches/"+strconv.FormatInt(b.ID, 10)+"/entries", bytes.NewReader(body))
	rec1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first entry: %d body=%s", rec1.Code, rec1.Body.String())
	}

	entry["transmitter"] = "TX-B" // 同指纹，发射机不同
	body2, _ := json.Marshal(entry)
	req2 := httptest.NewRequest(http.MethodPost, "/api/batches/"+strconv.FormatInt(b.ID, 10)+"/entries", bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("duplicate-fingerprint-different-transmitter want 409 got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestMissingBatch404(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/batches/99999", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", rec.Code)
	}
}

func TestHealthz(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz: %d", rec.Code)
	}
}

var _ = model.ErrBatchNotFound
