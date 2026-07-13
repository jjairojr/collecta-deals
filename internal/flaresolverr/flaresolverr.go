package flaresolverr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"opdeals/internal/httpx"
)

const (
	sessionName    = "opdeals"
	requestTimeout = 90 * time.Second
	solveTimeout   = 80000 // ms, passed to FlareSolverr as maxTimeout
	sessionRetries = 6
	sessionBackoff = 5 * time.Second
)

// Client talks to a FlareSolverr instance, which drives a headless Chrome to
// solve Cloudflare's Managed Challenge and returns the rendered HTML. It holds a
// single persistent session so the challenge is solved once and the warm browser
// plus its cf_clearance cookie are reused across requests; a per-request throwaway
// browser would re-solve every time. A mutex serializes /v1 calls because one
// FlareSolverr session is a single browser tab.
type Client struct {
	endpoint string
	http     *http.Client

	mu      sync.Mutex
	session string
}

// New returns a Client pointed at a FlareSolverr endpoint (e.g.
// "http://flaresolverr:8191"). The session is created lazily on first Get.
func New(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
		http:     &http.Client{Timeout: requestTimeout},
	}
}

type v1Request struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url,omitempty"`
	Session    string `json:"session,omitempty"`
	MaxTimeout int    `json:"maxTimeout,omitempty"`
}

type v1Response struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Session  string `json:"session"`
	Solution struct {
		Status   int    `json:"status"`
		Response string `json:"response"`
	} `json:"solution"`
}

func (c *Client) post(ctx context.Context, in v1Request) (*v1Response, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out v1Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if out.Status != "ok" {
		return nil, fmt.Errorf("flaresolverr: %s", out.Message)
	}
	return &out, nil
}

// ensureSession creates the persistent session if it does not exist yet,
// retrying so it tolerates the sidecar still booting.
func (c *Client) ensureSession(ctx context.Context) error {
	if c.session != "" {
		return nil
	}
	var lastErr error
	for attempt := 0; attempt < sessionRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sessionBackoff):
			}
		}
		_, err := c.post(ctx, v1Request{Cmd: "sessions.create", Session: sessionName})
		if err == nil {
			c.session = sessionName
			return nil
		}
		lastErr = err
	}
	return fmt.Errorf("create session: %w", lastErr)
}

// Get fetches url through FlareSolverr and returns the solved HTML. It is
// concurrency-safe; calls to the shared session are serialized.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}
	out, err := c.post(ctx, v1Request{
		Cmd:        "request.get",
		URL:        rawURL,
		Session:    c.session,
		MaxTimeout: solveTimeout,
	})
	if err != nil {
		return nil, err
	}
	return []byte(out.Solution.Response), nil
}

// Close destroys the FlareSolverr session so its browser is freed.
func (c *Client) Close(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == "" {
		return
	}
	_, _ = c.post(ctx, v1Request{Cmd: "sessions.destroy", Session: c.session})
	c.session = ""
}

// Router sends requests to challenged hosts through FlareSolverr and everything
// else (unchallenged CDNs like the sprite-atlas host, plus non-challenged games)
// straight through the direct client. It satisfies liga's Fetcher interface.
type Router struct {
	direct *httpx.Client
	solver *Client
	hosts  map[string]bool
}

// NewRouter routes GETs whose host is in challengedHosts through solver; all
// other hosts go to direct.
func NewRouter(direct *httpx.Client, solver *Client, challengedHosts []string) *Router {
	hosts := make(map[string]bool, len(challengedHosts))
	for _, h := range challengedHosts {
		hosts[h] = true
	}
	return &Router{direct: direct, solver: solver, hosts: hosts}
}

func (r *Router) Get(ctx context.Context, rawURL string) ([]byte, error) {
	if u, err := url.Parse(rawURL); err == nil && r.solver != nil && r.hosts[u.Host] {
		return r.solver.Get(ctx, rawURL)
	}
	return r.direct.Get(ctx, rawURL)
}
