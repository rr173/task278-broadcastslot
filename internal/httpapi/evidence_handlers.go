package httpapi

import (
	"net/http"
)

func (s *Server) handleAddEntry(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Title          string `json:"title"`
		Callsign       string `json:"callsign"`
		PrintedStartMs int64  `json:"printed_start_ms"`
		PrintedEndMs   int64  `json:"printed_end_ms"`
		PageID         string `json:"page_id"`
		Transmitter    string `json:"transmitter"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	e, err := s.svc.AddEntry(batchID, req.Title, req.Callsign, req.PrintedStartMs, req.PrintedEndMs, req.PageID, req.Transmitter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (s *Server) handleListEntries(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	list, err := s.svc.ListEntries(batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleAddClip(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		ClipNo   int64  `json:"clip_no"`
		Callsign string `json:"callsign"`
		OffsetMs int64  `json:"offset_ms"`
		Source   string `json:"source"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	c, err := s.svc.AddClip(batchID, req.ClipNo, req.Callsign, req.OffsetMs, req.Source)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleListClips(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	list, err := s.svc.ListClips(batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleAddAd(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		AdNo           int64  `json:"ad_no"`
		PrintedStartMs int64  `json:"printed_start_ms"`
		PageID         string `json:"page_id"`
		Edition        string `json:"edition"`
		Note           string `json:"note"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	a, err := s.svc.AddAd(batchID, req.AdNo, req.PrintedStartMs, req.PageID, req.Edition, req.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) handleListAds(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	list, err := s.svc.ListAds(batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleAddCitation(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		FromRef string `json:"from_ref"`
		ToRef   string `json:"to_ref"`
		Kind    string `json:"kind"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	c, err := s.svc.AddCitation(batchID, req.FromRef, req.ToRef, req.Kind)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleListCitations(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	list, err := s.svc.ListCitations(batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
