package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"opdeals/internal/adtestq"
)

func (s *Server) adTestRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/adtests", s.handleAdTestCreate)
	mux.HandleFunc("GET /api/adtests", s.handleAdTestList)
	mux.HandleFunc("POST /api/adtests/{id}/started", s.handleAdTestStarted)
	mux.HandleFunc("POST /api/adtests/{id}/meta", s.handleAdTestMeta)
	mux.HandleFunc("POST /api/adtests/{id}/finished", s.handleAdTestFinished)
	mux.HandleFunc("POST /api/adtests/{id}/failed", s.handleAdTestFailed)
	mux.HandleFunc("POST /api/aprovar-anuncio/{id}", s.handleAdTestDecide)
}

func adTestError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, adtestq.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, adtestq.ErrBadToken):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, adtestq.ErrNotPending), errors.Is(err, adtestq.ErrWrongStatus), errors.Is(err, adtestq.ErrOverBudget):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, adtestq.ErrInvalidInput):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleAdTestCreate(w http.ResponseWriter, r *http.Request) {
	var in adtestq.Test
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t, err := s.AdTests.Add(in)
	if err != nil {
		adTestError(w, err)
		return
	}
	writeJSON(w, t)
}

func (s *Server) handleAdTestList(w http.ResponseWriter, r *http.Request) {
	all, err := s.AdTests.List(r.URL.Query().Get("status"))
	if err != nil {
		adTestError(w, err)
		return
	}
	available, err := s.AdTests.Available()
	if err != nil {
		adTestError(w, err)
		return
	}
	writeJSON(w, map[string]any{"items": all, "availableCents": available, "weeklyCapCents": adtestq.WeeklyCapCents})
}

func (s *Server) handleAdTestStarted(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Meta   adtestq.Meta `json:"meta"`
		EndsAt time.Time    `json:"endsAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t, err := s.AdTests.Started(r.PathValue("id"), in.Meta, in.EndsAt)
	if err != nil {
		adTestError(w, err)
		return
	}
	writeJSON(w, t)
}

func (s *Server) handleAdTestMeta(w http.ResponseWriter, r *http.Request) {
	var in adtestq.Meta
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t, err := s.AdTests.SetMeta(r.PathValue("id"), in)
	if err != nil {
		adTestError(w, err)
		return
	}
	writeJSON(w, t)
}

func (s *Server) handleAdTestFinished(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Summary) == "" {
		http.Error(w, "informe o resumo do teste", http.StatusBadRequest)
		return
	}
	t, err := s.AdTests.Finished(r.PathValue("id"), strings.TrimSpace(in.Summary))
	if err != nil {
		adTestError(w, err)
		return
	}
	writeJSON(w, t)
}

func (s *Server) handleAdTestFailed(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Error string        `json:"error"`
		Meta  *adtestq.Meta `json:"meta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Error) == "" {
		http.Error(w, "informe o motivo da falha", http.StatusBadRequest)
		return
	}
	t, err := s.AdTests.Failed(r.PathValue("id"), strings.TrimSpace(in.Error), in.Meta)
	if err != nil {
		adTestError(w, err)
		return
	}
	writeJSON(w, t)
}

func (s *Server) handleAdTestDecide(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var approve bool
	switch q.Get("acao") {
	case "aprovar":
		approve = true
	case "rejeitar":
	default:
		http.Error(w, "acao precisa ser aprovar ou rejeitar", http.StatusBadRequest)
		return
	}
	t, err := s.AdTests.Decide(r.PathValue("id"), q.Get("t"), approve)
	if err != nil {
		adTestError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if t.Status == adtestq.StatusApproved {
		_, _ = w.Write([]byte("Aprovado: o teste sobe no Meta na próxima rodada, com a verba aprovada.\n"))
		return
	}
	_, _ = w.Write([]byte("Rejeitado: o teste não vai ao ar e a verba volta para a semana.\n"))
}
