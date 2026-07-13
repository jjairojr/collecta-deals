package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"opdeals/internal/model"
	"opdeals/internal/quotes"
)

func validQuoteItems(items []quotes.Item) bool {
	for _, it := range items {
		if it.Number == "" && it.Name == "" {
			return false
		}
	}
	return true
}

func normalizeQuoteItems(items []quotes.Item) {
	for i := range items {
		items[i].Number = model.NormalizeNumber(items[i].Number)
	}
}

func (s *Server) handleQuotesList(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Quotes == nil {
		http.Error(w, "quotes store unavailable", http.StatusServiceUnavailable)
		return
	}
	all, err := gs.Quotes.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string][]quotes.Quote{"quotes": all})
}

func (s *Server) handleQuotesCreate(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Quotes == nil {
		http.Error(w, "quotes store unavailable", http.StatusServiceUnavailable)
		return
	}
	var q quotes.Quote
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !validQuoteItems(q.Items) {
		http.Error(w, "item number or name required", http.StatusBadRequest)
		return
	}
	normalizeQuoteItems(q.Items)
	created, err := gs.Quotes.Add(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, created)
}

func (s *Server) handleQuotesUpdate(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Quotes == nil {
		http.Error(w, "quotes store unavailable", http.StatusServiceUnavailable)
		return
	}
	var q quotes.Quote
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !validQuoteItems(q.Items) {
		http.Error(w, "item number or name required", http.StatusBadRequest)
		return
	}
	normalizeQuoteItems(q.Items)
	updated, err := gs.Quotes.Update(r.PathValue("id"), q)
	if errors.Is(err, quotes.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, updated)
}

func (s *Server) handleQuotesDelete(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Quotes == nil {
		http.Error(w, "quotes store unavailable", http.StatusServiceUnavailable)
		return
	}
	err := gs.Quotes.Delete(r.PathValue("id"))
	if errors.Is(err, quotes.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}
