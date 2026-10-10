package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"opdeals/internal/trades"
)

const itemPhotosPath = "/api/item-photos/"

const maxPhotoBytes = 10 << 20

const maxPhotosPerUpload = 20

var photoExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func ownPhotoFile(imageURL string) string {
	if !strings.HasPrefix(imageURL, itemPhotosPath) {
		return ""
	}
	name := path.Base(strings.TrimPrefix(imageURL, itemPhotosPath))
	if name == "." || name == "/" {
		return ""
	}
	return name
}

func (s *Server) removePhotoFile(imageURL string) {
	if name := ownPhotoFile(imageURL); name != "" && s.PhotoDir != "" {
		os.Remove(filepath.Join(s.PhotoDir, name))
	}
}

type photoError struct {
	status  int
	message string
}

func (s *Server) savePhoto(id string, fh *multipart.FileHeader) (string, *photoError) {
	file, err := fh.Open()
	if err != nil {
		return "", &photoError{http.StatusBadRequest, err.Error()}
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, maxPhotoBytes+1))
	if err != nil {
		return "", &photoError{http.StatusBadRequest, err.Error()}
	}
	if len(body) > maxPhotoBytes {
		return "", &photoError{http.StatusRequestEntityTooLarge, "photo larger than 10 MB"}
	}
	ext, ok := photoExt[http.DetectContentType(body)]
	if !ok {
		return "", &photoError{http.StatusUnsupportedMediaType, "photo must be jpeg, png or webp"}
	}
	name := id + "-" + randomHex(4) + ext
	if err := os.WriteFile(filepath.Join(s.PhotoDir, name), body, 0o644); err != nil {
		return "", &photoError{http.StatusInternalServerError, err.Error()}
	}
	return itemPhotosPath + name, nil
}

func (s *Server) handleTradePhotoUpload(w http.ResponseWriter, r *http.Request) {
	if s.PhotoDir == "" {
		http.Error(w, "photo storage unavailable", http.StatusServiceUnavailable)
		return
	}
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxPhotosPerUpload)*(maxPhotoBytes+1<<16))
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	headers := r.MultipartForm.File["photo"]
	if len(headers) == 0 {
		http.Error(w, "photo field required", http.StatusBadRequest)
		return
	}
	if len(headers) > maxPhotosPerUpload {
		http.Error(w, "too many photos in one upload", http.StatusBadRequest)
		return
	}
	id := r.PathValue("id")
	if err := os.MkdirAll(s.PhotoDir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	saved := make([]string, 0, len(headers))
	for _, fh := range headers {
		url, perr := s.savePhoto(id, fh)
		if perr != nil {
			for _, u := range saved {
				s.removePhotoFile(u)
			}
			http.Error(w, perr.message, perr.status)
			return
		}
		saved = append(saved, url)
	}
	updated, err := onLedger(s.ledgersFor(gs), func(st *trades.Store) (trades.Trade, error) {
		return st.Update(id, func(t *trades.Trade) {
			t.Photos = append(t.Photos, saved...)
			if t.ImageURL == "" {
				t.ImageURL = t.Photos[0]
			}
		})
	})
	if err != nil {
		for _, u := range saved {
			s.removePhotoFile(u)
		}
		if errors.Is(err, trades.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, updated)
}

func (s *Server) handleTradePhotoDelete(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	target := r.URL.Query().Get("url")
	if ownPhotoFile(target) == "" {
		http.Error(w, "url of an uploaded photo required", http.StatusBadRequest)
		return
	}
	found := false
	updated, err := onLedger(s.ledgersFor(gs), func(st *trades.Store) (trades.Trade, error) {
		return st.Update(r.PathValue("id"), func(t *trades.Trade) {
			kept := t.Photos[:0]
			for _, p := range t.Photos {
				if p == target {
					found = true
					continue
				}
				kept = append(kept, p)
			}
			t.Photos = kept
			if t.ImageURL == target {
				found = true
				t.ImageURL = ""
				if len(t.Photos) > 0 {
					t.ImageURL = t.Photos[0]
				}
			}
		})
	})
	if errors.Is(err, trades.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if found {
		s.removePhotoFile(target)
	}
	writeJSON(w, updated)
}

func (s *Server) handleItemPhoto(w http.ResponseWriter, r *http.Request) {
	name := path.Base(r.PathValue("name"))
	if s.PhotoDir == "" || name == "." || name == "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, filepath.Join(s.PhotoDir, name))
}

func publicBase(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}
	return scheme + "://" + r.Host
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "0"
	}
	return hex.EncodeToString(b)
}
