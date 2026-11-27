package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"opdeals/internal/vitrine"
)

// vitrineRoutes: the dashboard's "Vitrine" screen reads and replaces the whole
// config under the admin gate; the storefront reads only what is live today,
// under /api/storefront/ (public, no secret in it).
func (s *Server) vitrineRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vitrine", s.handleVitrineGet)
	mux.HandleFunc("PUT /api/vitrine", s.handleVitrinePut)
	mux.HandleFunc("POST /api/vitrine/image", s.handleVitrineImageUpload)
	mux.HandleFunc("GET /api/storefront/vitrine", s.handleVitrineLive)
	mux.HandleFunc("GET /api/storefront/vitrine/img/{name}", s.handleVitrineImage)
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
	// Only a pop-up that would show today must have its product on sale now; one
	// scheduled for later may be for something that is not listed yet. The live
	// route skips any pop-up whose product is not on sale.
	if c.Popup != nil && len(c.Popups) == 0 {
		c.Popups = []vitrine.Popup{*c.Popup}
		c.Popup = nil
	}
	for _, p := range c.PopupsOn(s.psNow()) {
		ok, err := s.popupSKUListed(p.SKU)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			writeError(w, http.StatusBadRequest, map[string]any{"error": "o SKU " + p.SKU + " do pop-up de hoje não está à venda na vitrine"})
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
	now := s.psNow()
	live := c.LiveAt(now)
	live.Popup = nil
	for _, p := range c.PopupsOn(now) {
		if ok, err := s.popupSKUListed(p.SKU); err == nil && ok {
			cp := p
			live.Popup = &cp
			break
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, live)
}

// handleVitrineImageUpload takes one art file (multipart field "file") from the
// Vitrine screen and answers with its public URL. Admin only (the gate).
func (s *Server) handleVitrineImageUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, vitrine.MaxImageBytes+64<<10)
	if err := r.ParseMultipartForm(vitrine.MaxImageBytes + 64<<10); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, map[string]any{"error": vitrine.ErrImageTooBig.Error()})
		return
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, map[string]any{"error": "envie o arquivo no campo file"})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, vitrine.MaxImageBytes+1))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	url, err := s.Vitrine.SaveImage(data)
	switch {
	case errors.Is(err, vitrine.ErrImageTooBig):
		writeError(w, http.StatusRequestEntityTooLarge, map[string]any{"error": err.Error()})
		return
	case errors.Is(err, vitrine.ErrImageType):
		writeError(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"url": url, "bytes": len(data)})
}

// handleVitrineImage serves an uploaded art file. The name is the content's hash,
// so the browser and the CDN may keep it for a year.
func (s *Server) handleVitrineImage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	path, err := s.Vitrine.ImageFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", vitrine.MimeFor(name))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, name, time.Time{}, f)
}
