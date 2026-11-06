package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"opdeals/internal/publishq"
)

func (s *Server) publishRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/publish", s.handlePublishCreate)
	mux.HandleFunc("GET /api/publish", s.handlePublishList)
	mux.HandleFunc("POST /api/publish/{id}/published", s.handlePublishDone)
	mux.HandleFunc("POST /api/publish/{id}/failed", s.handlePublishFailed)
	mux.HandleFunc("POST /api/aprovar/{id}", s.handlePublishDecide)
}

func publishError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, publishq.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, publishq.ErrBadToken):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, publishq.ErrNotPending), errors.Is(err, publishq.ErrNotApproved):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, publishq.ErrInvalidInput):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handlePublishCreate(w http.ResponseWriter, r *http.Request) {
	var in publishq.Item
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	it, err := s.Publish.Add(in)
	if err != nil {
		publishError(w, err)
		return
	}
	writeJSON(w, it)
}

func (s *Server) handlePublishList(w http.ResponseWriter, r *http.Request) {
	all, err := s.Publish.List(r.URL.Query().Get("status"))
	if err != nil {
		publishError(w, err)
		return
	}
	writeJSON(w, map[string][]publishq.Item{"items": all})
}

func (s *Server) handlePublishDone(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Link string `json:"link"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Link) == "" {
		http.Error(w, "informe o link do post", http.StatusBadRequest)
		return
	}
	it, err := s.Publish.Published(r.PathValue("id"), strings.TrimSpace(in.Link))
	if err != nil {
		publishError(w, err)
		return
	}
	writeJSON(w, it)
}

func (s *Server) handlePublishFailed(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Error     string `json:"error"`
		Permanent bool   `json:"permanent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Error) == "" {
		http.Error(w, "informe o motivo da falha", http.StatusBadRequest)
		return
	}
	it, err := s.Publish.Failed(r.PathValue("id"), strings.TrimSpace(in.Error), in.Permanent)
	if err != nil {
		publishError(w, err)
		return
	}
	writeJSON(w, it)
}

func (s *Server) handlePublishDecide(w http.ResponseWriter, r *http.Request) {
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
	it, err := s.Publish.Decide(r.PathValue("id"), q.Get("t"), approve)
	if err != nil {
		publishError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if it.Status == publishq.StatusApproved {
		_, _ = w.Write([]byte("Aprovado: o Publicador posta no horário marcado.\n"))
		return
	}
	_, _ = w.Write([]byte("Rejeitado: não vai ser postado.\n"))
}
