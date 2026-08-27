package httpapi

import (
	"net/http"

	"task278-broadcastslot/internal/model"
)

func (s *Server) handleCreateBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code     string `json:"code"`
		Station  string `json:"station"`
		AirDate  string `json:"air_date"`
		Timezone string `json:"timezone"`
		DriftPPM float64 `json:"drift_ppm"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	b := model.EvidenceBatch{
		Code: req.Code, Station: req.Station, AirDate: req.AirDate,
		Timezone: req.Timezone, DriftPPM: req.DriftPPM, Status: model.BatchOrganizing,
	}
	if err := s.svc.CreateBatch(&b); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) handleListBatches(w http.ResponseWriter, _ *http.Request) {
	list, err := s.svc.ListBatches()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleGetBatch(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	b, err := s.svc.GetBatch(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleBatchStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.svc.TransitionStatus(id, req.Status); err != nil {
		writeError(w, err)
		return
	}
	b, _ := s.svc.GetBatch(id)
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleSealBatch(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.svc.SealBatch(id); err != nil {
		writeError(w, err)
		return
	}
	b, _ := s.svc.GetBatch(id)
	writeJSON(w, http.StatusOK, b)
}
