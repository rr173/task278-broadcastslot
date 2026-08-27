package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"task278-broadcastslot/internal/model"
	"task278-broadcastslot/internal/service"
)

// Server HTTP 服务。
type Server struct {
	svc *service.Service
	mux *http.ServeMux
}

// NewServer 构造并注册路由。
func NewServer(svc *service.Service) *Server {
	s := &Server{svc: svc, mux: http.NewServeMux()}
	s.register()
	return s
}

// Handler 返回 http.Handler。
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) register() {
	m := s.mux
	m.HandleFunc("GET /healthz", s.handleHealthz)
	m.HandleFunc("GET /api/stats", s.handleStats)
	m.HandleFunc("GET /", s.handleIndex)

	m.HandleFunc("POST /api/batches", s.handleCreateBatch)
	m.HandleFunc("GET /api/batches", s.handleListBatches)
	m.HandleFunc("GET /api/batches/{id}", s.handleGetBatch)
	m.HandleFunc("POST /api/batches/{id}/status", s.handleBatchStatus)
	m.HandleFunc("POST /api/batches/{id}/seal", s.handleSealBatch)

	m.HandleFunc("POST /api/batches/{id}/entries", s.handleAddEntry)
	m.HandleFunc("GET /api/batches/{id}/entries", s.handleListEntries)

	m.HandleFunc("POST /api/batches/{id}/clips", s.handleAddClip)
	m.HandleFunc("GET /api/batches/{id}/clips", s.handleListClips)

	m.HandleFunc("POST /api/batches/{id}/ads", s.handleAddAd)
	m.HandleFunc("GET /api/batches/{id}/ads", s.handleListAds)

	m.HandleFunc("POST /api/batches/{id}/citations", s.handleAddCitation)
	m.HandleFunc("GET /api/batches/{id}/citations", s.handleListCitations)

	m.HandleFunc("POST /api/batches/{id}/correct", s.handleCorrect)
	m.HandleFunc("GET /api/batches/{id}/corrections", s.handleListCorrections)

	m.HandleFunc("POST /api/batches/{id}/align", s.handleAlign)
	m.HandleFunc("GET /api/batches/{id}/attributions", s.handleListAttributions)
	m.HandleFunc("GET /api/batches/{id}/conflicts", s.handleListConflicts)

	m.HandleFunc("POST /api/batches/{id}/verdicts", s.handleRecordVerdict)
	m.HandleFunc("GET /api/batches/{id}/verdicts", s.handleListVerdicts)

	m.HandleFunc("POST /api/batches/{id}/versions", s.handleBuildVersion)
	m.HandleFunc("GET /api/batches/{id}/versions", s.handleListVersions)
	m.HandleFunc("GET /api/batches/{id}/versions/{vid}", s.handleGetVersion)
	m.HandleFunc("POST /api/batches/{id}/publish", s.handlePublish)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Printf("httpapi: encode: %v", err)
		}
	}
}

func writeError(w http.ResponseWriter, err error) {
	status := mapErrorStatus(err)
	code := "internal_error"
	switch status {
	case http.StatusNotFound:
		code = "not_found"
	case http.StatusConflict:
		code = "conflict"
	case http.StatusBadRequest:
		code = "bad_request"
	}
	writeJSON(w, status, map[string]string{"error": code, "message": err.Error()})
}

func mapErrorStatus(err error) int {
	switch {
	case errors.Is(err, model.ErrBatchNotFound),
		errors.Is(err, model.ErrVersionNotFound),
		errors.Is(err, model.ErrAttributionNotFound):
		return http.StatusNotFound
	case errors.Is(err, model.ErrDuplicateCode),
		err == model.ErrDuplicateFingerprint,
		errors.Is(err, model.ErrVersionConflict),
		errors.Is(err, model.ErrSourceCycle):
		return http.StatusConflict
	case errors.Is(err, model.ErrSlotInverted),
		errors.Is(err, model.ErrSealed),
		errors.Is(err, model.ErrIllegalTransition),
		errors.Is(err, model.ErrUnknownTimezone),
		errors.Is(err, model.ErrFrozenVersion):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func decodeJSON(r *http.Request, dst any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return io.EOF
	}
	return json.Unmarshal(body, dst)
}

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func pathVID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("vid"), 10, 64)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	st, err := s.svc.Stats()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}
