package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type arenaEndpoint struct {
	method string
	path   string
}

var arenaEndpoints = []arenaEndpoint{
	{http.MethodGet, "/overview"},
	{http.MethodGet, "/games"},
	{http.MethodGet, "/pending"},
	{http.MethodGet, "/members"},
	{http.MethodPost, "/members/{id}/approve"},
	{http.MethodPost, "/members/{id}/reject"},
	{http.MethodPost, "/members/{id}/minor"},
	{http.MethodPost, "/members/{id}/remove"},
	{http.MethodGet, "/members/{id}/statement"},
	{http.MethodGet, "/events"},
	{http.MethodPost, "/events"},
	{http.MethodGet, "/events/{id}"},
	{http.MethodPost, "/events/{id}/checkin"},
	{http.MethodPost, "/events/{id}/end"},
	{http.MethodPost, "/credit"},
	{http.MethodPost, "/entries/{id}/reverse"},
	{http.MethodGet, "/duels"},
	{http.MethodPost, "/duels/{id}/resolve"},
	{http.MethodPost, "/duels/{id}/annul"},
}

func (s *Server) arenaRoutes(mux *http.ServeMux) {
	for _, e := range arenaEndpoints {
		mux.HandleFunc(e.method+" /api/arena"+e.path, s.arenaProxy(e.path))
	}
}

func arenaMessage(w http.ResponseWriter, status int, message string) {
	writeError(w, status, map[string]any{"message": message})
}

func (s *Server) arenaProxy(pattern string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Arena == nil {
			arenaMessage(w, http.StatusServiceUnavailable, "Arena Collecta desativada: defina COLLECTA_API e COLLECTA_STORE_TOKEN no servidor")
			return
		}
		path := "/store/arena" + pattern
		if strings.Contains(pattern, "{id}") {
			id := r.PathValue("id")
			if id == "" || id == "." || id == ".." {
				arenaMessage(w, http.StatusBadRequest, "id inválido")
				return
			}
			path = strings.Replace(path, "{id}", url.PathEscape(id), 1)
		}
		var body []byte
		if r.Method == http.MethodPost {
			raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if err != nil {
				arenaMessage(w, http.StatusBadRequest, "corpo inválido")
				return
			}
			if len(bytes.TrimSpace(raw)) == 0 {
				raw = []byte("{}")
			}
			if !json.Valid(raw) {
				arenaMessage(w, http.StatusBadRequest, "corpo inválido")
				return
			}
			body = raw
		}
		status, data, err := s.Arena.Arena(r.Context(), r.Method, path, r.URL.RawQuery, body)
		if err != nil {
			fmt.Printf("[ARENA] %s %s: %v\n", r.Method, path, err)
			arenaMessage(w, http.StatusBadGateway, "Arena Collecta indisponível no momento")
			return
		}
		if status == http.StatusUnauthorized {
			arenaMessage(w, http.StatusBadGateway, "o servidor do Collecta recusou o token da loja (COLLECTA_STORE_TOKEN)")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(data)
	}
}
