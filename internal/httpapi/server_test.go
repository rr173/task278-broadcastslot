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

// TestCitationCycleReturns409 两条互相引用的来源在登记第二条边时成环，
// 接口必须以 409 冲突返回，而非 500 内部错误。
func TestCitationCycleReturns409(t *testing.T) {
	srv := newTestServer(t)

	batch := map[string]any{
		"code": "CYC-1", "station": "S", "air_date": "1952-01-01", "timezone": "UTC",
	}
	bb, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/api/batches", bytes.NewReader(bb))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create batch: %d body=%s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	post := func(from, to string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"from_ref": from, "to_ref": to, "kind": "cross"})
		r := httptest.NewRequest(http.MethodPost,
			"/api/batches/"+strconv.FormatInt(created.ID, 10)+"/citations", bytes.NewReader(body))
		rc := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rc, r)
		return rc
	}

	if rc := post("entry:1", "clip:1"); rc.Code != http.StatusCreated {
		t.Fatalf("first citation want 201 got %d body=%s", rc.Code, rc.Body.String())
	}
	// 互相引用：clip:1 -> entry:1，与已登记的 entry:1 -> clip:1 成环。
	rc := post("clip:1", "entry:1")
	if rc.Code != http.StatusConflict {
		t.Fatalf("mutual citation cycle want 409 got %d body=%s", rc.Code, rc.Body.String())
	}
}

var _ = model.ErrBatchNotFound
