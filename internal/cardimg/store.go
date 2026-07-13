package cardimg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"opdeals/internal/logx"
)

// Fetcher retrieves a URL's bytes. It is satisfied by *httpx.Client for direct
// fetches and by *flaresolverr.Router when the game's Liga host sits behind
// Cloudflare's Managed Challenge — the router sends card pages through the
// solver while CDN image downloads (a different host) stay direct.
type Fetcher interface {
	Get(ctx context.Context, url string) ([]byte, error)
}

type Item struct {
	Set     string
	Number  string
	PageURL string
}

// sealedImageRe matches a sealed product's artwork on its Liga prod/view page.
// Sealed products live under a game-agnostic upload path (/arquivos/up/prod/...)
// rather than the per-game singles path (/arquivos/in/{game}/...), so it is used
// as a fallback when the singles regex finds nothing.
var sealedImageRe = regexp.MustCompile(`//repositorio\.sbrauble\.com/arquivos/up/prod/\d+/[^"'\s]+\.(?:jpg|jpeg|png)`)

type Store struct {
	path       string
	mu         sync.RWMutex
	urls       map[string]string
	byNum      map[string]string
	http       Fetcher
	log        *logx.Logger
	imgRe      *regexp.Regexp
	uniqueNums bool
}

func NewStore(path string, client Fetcher, log *logx.Logger, imgRe *regexp.Regexp, uniqueNums bool) *Store {
	s := &Store{path: path, urls: map[string]string{}, byNum: map[string]string{}, http: client, log: log, imgRe: imgRe, uniqueNums: uniqueNums}
	s.load()
	return s
}

func key(set, number string) string { return set + "/" + number }

// numberOf extracts the card number from a "set/number" cache key. When a
// game's card numbers are globally unique this backs a set-insensitive
// fallback that survives set-code format drift (e.g. "OP-11" warmed by
// tracking vs. "OP11" stored on a trade). Games with per-set numbers must not
// use it — the same number names a different card in every set.
func numberOf(cacheKey string) string {
	if i := strings.LastIndex(cacheKey, "/"); i >= 0 {
		return cacheKey[i+1:]
	}
	return cacheKey
}

func (s *Store) load() {
	body, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var m map[string]string
	if err := json.Unmarshal(body, &m); err != nil {
		s.log.Printf("cardimg: ignoring corrupt cache %s: %v", s.path, err)
		return
	}
	s.urls = m
	if !s.uniqueNums {
		return
	}
	s.byNum = make(map[string]string, len(m))
	for k, u := range m {
		if u != "" {
			s.byNum[numberOf(k)] = u
		}
	}
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(s.urls, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) cached(set, number string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if u, ok := s.urls[key(set, number)]; ok && u != "" {
		return u, true
	}
	if !s.uniqueNums {
		return "", false
	}
	u, ok := s.byNum[number]
	return u, ok && u != ""
}

func (s *Store) store(set, number, url string) {
	s.mu.Lock()
	s.urls[key(set, number)] = url
	if url != "" && s.uniqueNums {
		s.byNum[number] = url
	}
	err := s.save()
	s.mu.Unlock()
	if err != nil {
		s.log.Printf("cardimg: save cache: %v", err)
	}
}

// Resolve returns the CDN image URL for a card, fetching and scraping the card
// page once and caching the result. pageURL is the card's LigaOnePiece page.
func (s *Store) Resolve(ctx context.Context, set, number, pageURL string) (string, error) {
	if u, ok := s.cached(set, number); ok {
		return u, nil
	}
	body, err := s.http.Get(ctx, pageURL)
	if err != nil {
		return "", fmt.Errorf("card page %s: %w", number, err)
	}
	m := s.imgRe.Find(body)
	if m == nil {
		// Sealed products (prod/view pages) carry their art under a different path.
		m = sealedImageRe.Find(body)
	}
	if m == nil {
		return "", fmt.Errorf("no image on card page %s", number)
	}
	url := "https:" + string(m)
	s.store(set, number, url)
	return url, nil
}

// Warm resolves image URLs for many cards, skipping already-cached ones. It
// paces requests with a ticker (interval) and caps in-flight requests at
// concurrency, mirroring the polite capture rate to avoid 429s. It returns the
// count successfully resolved and the total requested.
func (s *Store) Warm(ctx context.Context, items []Item, concurrency int, interval time.Duration) (int, int) {
	if concurrency < 1 {
		concurrency = 1
	}
	var resolved, done int64
	total := int64(len(items))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)
	var ticker *time.Ticker
	if interval > 0 {
		ticker = time.NewTicker(interval)
		defer ticker.Stop()
	}
	for _, it := range items {
		it := it
		if _, ok := s.cached(it.Set, it.Number); ok {
			atomic.AddInt64(&resolved, 1)
			atomic.AddInt64(&done, 1)
			continue
		}
		if ticker != nil {
			select {
			case <-ctx.Done():
				break
			case <-ticker.C:
			}
		}
		g.Go(func() error {
			_, err := s.Resolve(ctx, it.Set, it.Number, it.PageURL)
			if err == nil {
				atomic.AddInt64(&resolved, 1)
			} else {
				s.log.Printf("cardimg: warm %s/%s: %v", it.Set, it.Number, err)
			}
			if d := atomic.AddInt64(&done, 1); d%100 == 0 {
				s.log.Printf("cardimg: warmed %d/%d", d, total)
			}
			return nil
		})
	}
	g.Wait()
	return int(resolved), len(items)
}

// FetchURL downloads bytes from an already-known image URL, bypassing Liga page
// resolution. It proxies card art whose URL is derived directly from a source id
// (e.g. a TCGplayer product image), so a browser <canvas> can draw it
// same-origin without tripping the CDN's missing CORS headers.
func (s *Store) FetchURL(ctx context.Context, url string) ([]byte, error) {
	return s.http.Get(ctx, url)
}

// Fetch resolves the image URL then downloads the JPEG bytes.
func (s *Store) Fetch(ctx context.Context, set, number, pageURL string) ([]byte, error) {
	url, err := s.Resolve(ctx, set, number, pageURL)
	if err != nil {
		return nil, err
	}
	body, err := s.http.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("image %s: %w", number, err)
	}
	return body, nil
}
