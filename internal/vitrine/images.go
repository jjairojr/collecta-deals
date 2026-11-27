package vitrine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MaxImageBytes is the largest art file the Vitrine screen accepts. The art
// team works to much smaller targets (150 KB for the bar, 350 KB for the
// banner); this only stops a mistake from slowing the site down.
const MaxImageBytes = 1 << 20

// ImagePath is where the storefront fetches an uploaded file. It sits under
// /api/storefront/, which is public, and the name is the file's own hash, so the
// content behind a URL never changes and can be cached for a year.
const ImagePath = "/api/storefront/vitrine/img/"

var (
	ErrImageTooBig  = errors.New("imagem acima de 1 MB")
	ErrImageType    = errors.New("a imagem precisa ser WebP, JPG ou PNG")
	ErrImageMissing = errors.New("imagem não encontrada")

	imageNameRe = regexp.MustCompile(`^[0-9a-f]{16}\.(webp|jpg|png)$`)
)

// IsImageURL reports whether u points at a file uploaded through SaveImage.
func IsImageURL(u string) bool {
	name, ok := strings.CutPrefix(u, ImagePath)
	return ok && imageNameRe.MatchString(name)
}

// imageKind tells WebP, JPEG and PNG apart by their first bytes: the
// Content-Type the browser sends is not trusted.
func imageKind(b []byte) (ext, mime string, ok bool) {
	switch {
	case len(b) >= 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return "webp", "image/webp", true
	case len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		return "jpg", "image/jpeg", true
	case len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}):
		return "png", "image/png", true
	}
	return "", "", false
}

// MimeFor is the Content-Type to serve a stored image with.
func MimeFor(name string) string {
	switch filepath.Ext(name) {
	case ".webp":
		return "image/webp"
	case ".png":
		return "image/png"
	}
	return "image/jpeg"
}

func (s *Store) imageDir() string {
	return strings.TrimSuffix(s.path, filepath.Ext(s.path)) + "-img"
}

// SaveImage stores an uploaded art file and returns its public URL. The same
// file uploaded twice gets the same URL.
func (s *Store) SaveImage(data []byte) (string, error) {
	if len(data) > MaxImageBytes {
		return "", ErrImageTooBig
	}
	ext, _, ok := imageKind(data)
	if !ok {
		return "", ErrImageType
	}
	sum := sha256.Sum256(data)
	name := hex.EncodeToString(sum[:])[:16] + "." + ext
	dir := s.imageDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return ImagePath + name, nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", fmt.Errorf("guardar imagem: %w", err)
	}
	return ImagePath + name, nil
}

// ImageFile returns the path of a stored image by its name, refusing any name
// that is not one SaveImage produced (no "..", no slashes).
func (s *Store) ImageFile(name string) (string, error) {
	if !imageNameRe.MatchString(name) {
		return "", ErrImageMissing
	}
	path := filepath.Join(s.imageDir(), name)
	if _, err := os.Stat(path); err != nil {
		return "", ErrImageMissing
	}
	return path, nil
}
