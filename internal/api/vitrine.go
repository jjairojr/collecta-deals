package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"opdeals/internal/vitrine"
)

// vitrineRoutes: the dashboard's "Vitrine" screen reads and replaces the whole
// config under the admin gate; the storefront reads only what is live today,
// under /api/storefront/ (public, no secret in it).
func (s *Server) vitrineRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vitrine", s.handleVitrineGet)
	mux.HandleFunc("PUT /api/vitrine", s.handleVitrinePut)
	mux.HandleFunc("GET /api/storefront/vitrine", s.handleVitrineLive)
}

func (s *Server) handleVitrineGet(w http.ResponseWriter, r *http.Request) {
	c, err := s.Vitrine.Get()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, c)
}

// popupSKUListed reports whether the pop-up's product is on sale now: a pop-up
// for something the buyer cannot add to the cart is never shown.
func (s *Server) popupSKUListed(sku string) (bool, error) {
	idx, err := s.buildSKUIndex()
	if err != nil {
		return false, err
	}
	_, ok := idx[sku]
	return ok, nil
}

func (s *Server) handleVitrinePut(w http.ResponseWriter, r *http.Request) {
	var c vitrine.Config
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&c); err != nil {
		writeError(w, http.StatusBadRequest, map[string]any{"error": "JSON inválido: " + err.Error()})
		return
	}
	if c.Popup != nil && c.Popup.Enabled {
		ok, err := s.popupSKUListed(c.Popup.SKU)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			writeError(w, http.StatusBadRequest, map[string]any{"error": "o SKU " + c.Popup.SKU + " do pop-up não está à venda na vitrine"})
			return
		}
	}
	saved, err := s.Vitrine.Put(c, s.psNow())
	if errors.Is(err, vitrine.ErrInvalid) {
		writeError(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, saved)
}

func (s *Server) handleVitrineLive(w http.ResponseWriter, r *http.Request) {
	c, err := s.Vitrine.Get()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	live := c.LiveAt(s.psNow())
	if live.Popup != nil {
		if ok, err := s.popupSKUListed(live.Popup.SKU); err != nil || !ok {
			live.Popup = nil
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, live)
}
