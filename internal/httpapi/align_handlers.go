package httpapi

import (
	"io"
	"net/http"
)

func (s *Server) handleCorrect(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	list, err := s.svc.Correct(r.Context(), batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleListCorrections(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	list, err := s.svc.ListCorrections(batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleAlign(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	res, err := s.svc.Align(r.Context(), batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleListAttributions(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	list, err := s.svc.ListAttributions(batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleListConflicts(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	list, err := s.svc.ListConflicts(batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleRecordVerdict(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		EntryID         int64  `json:"entry_id"`
		Decision        string `json:"decision"`
		Reviewer        string `json:"reviewer"`
		Note            string `json:"note"`
		ExpectedVersion int64  `json:"expected_version"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	v, err := s.svc.RecordVerdict(batchID, req.EntryID, req.Decision, req.Reviewer, req.Note, req.ExpectedVersion)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) handleListVerdicts(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	list, err := s.svc.ListVerdicts(batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleBuildVersion(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	v, err := s.svc.BuildVersion(batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	list, err := s.svc.ListVersions(batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	vid, err := pathVID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	v, err := s.svc.GetVersionByNo(batchID, vid)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	batchID, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Version int64 `json:"version"`
	}
	if err := decodeJSON(r, &req); err != nil && err != io.EOF {
		writeError(w, err)
		return
	}
	v, err := s.svc.PublishVersion(r.Context(), batchID, req.Version)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
