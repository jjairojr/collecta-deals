// Package clearance carries a Cloudflare cf_clearance cookie, earned by a real
// headed Chrome on the host (see cmd/cfgate), into the scraper's ordinary HTTP
// client.
//
// Cloudflare's Managed Challenge on the Liga hosts rejects every headless
// browser we can run in a container — FlareSolverr and Byparr both fail — but it
// accepts a plain Go HTTP request that carries a clearance cookie obtained by a
// headed browser, as long as the request repeats the exact User-Agent that
// solved the challenge and leaves from the same address.
//
// The cookie expires in roughly half an hour, so a Fetcher treats a block as
// routine: it asks cfgate for a fresh cookie and retries. Nothing refreshes on a
// timer, which keeps an idle stack from driving a browser for no reason.
package clearance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"opdeals/internal/game"
	"opdeals/internal/httpx"
	"opdeals/internal/liga"
)

// ErrMissing means no clearance cookie is on file for a challenged host.
var ErrMissing = errors.New("no cloudflare clearance on file")

// solveClient talks to cfgate, never to Liga. Its timeout covers a full headed
// Chrome challenge.
var solveClient = &http.Client{Timeout: 3 * time.Minute}

type Entry struct {
	Cookie     string    `json:"cfClearance"`
	UserAgent  string    `json:"userAgent"`
	ObtainedAt time.Time `json:"obtainedAt"`
}

type document struct {
	Hosts map[string]Entry `json:"hosts"`
}

// Store reads the clearance file lazily and re-reads it whenever it changes on
// disk, so a refresh run by the host takes effect in a running container without
// a restart.
type Store struct {
	path  string
	proxy *url.URL

	mu      sync.Mutex
	mod     time.Time
	size    int64
	hosts   map[string]Entry
	loadErr error
}

func NewStore(path string) *Store {
	return &Store{path: path, hosts: map[string]Entry{}}
}

// WithProxy sends challenged hosts through proxyURL — the IPv6 gateway cfgate
// serves on the host. Cloudflare denies this network's IPv4 egress outright for
// non-browser clients, so a container (which has no IPv6 of its own) reaches
// Liga only by borrowing the host's IPv6 through that proxy.
func (s *Store) WithProxy(proxyURL string) (*Store, error) {
	if strings.TrimSpace(proxyURL) == "" {
		return s, nil
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse proxy %q: %w", proxyURL, err)
	}
	s.proxy = u
	return s, nil
}

func (s *Store) Path() string { return s.path }

// key normalizes a request host to the way clearance is scoped: Cloudflare
// issues cf_clearance for the registrable domain, so www and apex share one
// entry.
func key(host string) string {
	return strings.TrimPrefix(strings.ToLower(host), "www.")
}

func (s *Store) reload() {
	info, err := os.Stat(s.path)
	if err != nil {
		s.loadErr = err
		s.hosts = map[string]Entry{}
		s.mod, s.size = time.Time{}, 0
		return
	}
	if info.ModTime().Equal(s.mod) && info.Size() == s.size {
		return
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		s.loadErr = err
		return
	}
	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		s.loadErr = fmt.Errorf("parse %s: %w", s.path, err)
		return
	}
	hosts := make(map[string]Entry, len(doc.Hosts))
	for h, e := range doc.Hosts {
		hosts[key(h)] = e
	}
	s.hosts, s.mod, s.size, s.loadErr = hosts, info.ModTime(), info.Size(), nil
}

// Lookup returns the clearance on file for host, reloading it first if the file
// changed since the last read.
func (s *Store) Lookup(host string) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reload()
	e, ok := s.hosts[key(host)]
	if !ok || e.Cookie == "" || e.UserAgent == "" {
		return Entry{}, false
	}
	return e, true
}

func (s *Store) Put(host string, e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hosts[key(host)] = e
}

// Save writes entries atomically, replacing the file.
func (s *Store) Save(entries map[string]Entry) error {
	normalized := make(map[string]Entry, len(entries))
	for h, e := range entries {
		normalized[key(h)] = e
	}
	raw, err := json.MarshalIndent(document{Hosts: normalized}, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(s.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	s.mu.Lock()
	s.hosts, s.mod, s.size, s.loadErr = normalized, time.Time{}, 0, nil
	s.mu.Unlock()
	return nil
}

// Fetcher sends challenged hosts through the direct client with the clearance
// cookie and its matching User-Agent attached, and leaves every other host
// (unchallenged CDNs, non-Liga sources) untouched.
type Fetcher struct {
	direct *httpx.Client
	store  *Store
	hosts  map[string]bool
}

// FetcherFor returns a liga.FetcherFor that decorates a game's direct client
// when that game's Liga host is challenged.
func (s *Store) FetcherFor() liga.FetcherFor {
	return func(direct *httpx.Client, g game.Game) liga.Fetcher {
		if !g.Challenged {
			return direct
		}
		hosts := make(map[string]bool, len(g.LigaHosts))
		for _, h := range g.LigaHosts {
			hosts[key(h)] = true
		}
		direct.UseProxyFor(s.proxy, func(host string) bool { return hosts[key(host)] })
		return &Fetcher{direct: direct, store: s, hosts: hosts}
	}
}

func (f *Fetcher) Get(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil || !f.hosts[key(u.Host)] {
		return f.direct.Get(ctx, rawURL)
	}

	body, err := f.get(ctx, rawURL, u.Host)
	// Clearance lives about half an hour, far less than the gap between scrape
	// cycles, so the first request of a cycle is expected to find it stale. Ask
	// the gateway for a fresh one and retry once rather than failing the run.
	if err != nil && f.canResolve() && (errors.Is(err, httpx.ErrBlocked) || errors.Is(err, ErrMissing)) {
		if solveErr := f.solve(ctx, u.Host); solveErr != nil {
			return nil, fmt.Errorf("%w (refreshing clearance failed: %v)", err, solveErr)
		}
		return f.get(ctx, rawURL, u.Host)
	}
	return body, err
}

func (f *Fetcher) get(ctx context.Context, rawURL, host string) ([]byte, error) {
	e, ok := f.store.Lookup(host)
	if !ok {
		return nil, fmt.Errorf("%s: %w", host, ErrMissing)
	}
	return f.direct.GetWithHeaders(ctx, rawURL, map[string]string{
		"Cookie":     "cf_clearance=" + e.Cookie,
		"User-Agent": e.UserAgent,
	})
}

func (f *Fetcher) canResolve() bool { return f.store.proxy != nil }

// solve asks cfgate to re-earn this host's clearance and returns once the file
// on disk holds it.
func (f *Fetcher) solve(ctx context.Context, host string) error {
	endpoint := "http://" + f.store.proxy.Host + "/solve?host=" + url.QueryEscape(strings.ToLower(host))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := solveClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("%s: status %d", endpoint, resp.StatusCode)
	}
	var e Entry
	if err := json.NewDecoder(resp.Body).Decode(&e); err == nil && e.Cookie != "" && e.UserAgent != "" {
		f.store.Put(host, e)
	}
	return nil
}
