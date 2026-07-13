package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	userAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	acceptHTML = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"
	acceptLang = "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7"

	defaultRetries = 3
	throttleGap    = 900 * time.Millisecond
	throttleJitter = 700 * time.Millisecond
	baseBackoff    = 2 * time.Second
	throttleBase   = 15 * time.Second
)

// ErrBlocked signals the origin refused the request (HTTP 403 — e.g. a
// Cloudflare IP ban or firewall rule). It is never retried: retrying into a
// block only deepens the ban. Callers should stop the whole run when they see
// it rather than moving on to the next URL.
var ErrBlocked = errors.New("blocked by origin")

type statusError struct {
	code       int
	retryAfter time.Duration
}

func (e *statusError) Error() string { return fmt.Sprintf("status %d", e.code) }

type Client struct {
	http    *http.Client
	retries int

	mu     sync.Mutex
	nextAt time.Time
	minGap time.Duration
	jitter time.Duration
}

// New returns an unthrottled client — suitable for well-behaved bulk endpoints
// (TCGCSV, TCGplayer, FX) that don't rate-limit us.
func New(timeout time.Duration) (*Client, error) {
	return newClient(timeout, 0, 0)
}

// NewThrottled returns a client that paces every request with a minimum gap
// plus jitter, so bursts (a page fetch plus its atlas sub-requests) are spread
// out. Use it for hosts that ban aggressive scraping, like LigaOnePiece.
func NewThrottled(timeout time.Duration) (*Client, error) {
	return newClient(timeout, throttleGap, throttleJitter)
}

func newClient(timeout, minGap, jitter time.Duration) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("cookiejar: %w", err)
	}
	return &Client{
		http:    &http.Client{Timeout: timeout, Jar: jar},
		retries: defaultRetries,
		minGap:  minGap,
		jitter:  jitter,
	}, nil
}

func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	return c.retry(ctx, http.MethodGet, url, nil, nil)
}

func (c *Client) PostJSON(ctx context.Context, url string, body []byte, headers map[string]string) ([]byte, error) {
	return c.retry(ctx, http.MethodPost, url, body, headers)
}

// pace blocks until this client is allowed to start another request, reserving
// the slot so concurrent callers stagger their starts by minGap+jitter.
func (c *Client) pace(ctx context.Context) error {
	if c.minGap <= 0 {
		return nil
	}
	c.mu.Lock()
	now := time.Now()
	start := c.nextAt
	if start.Before(now) {
		start = now
	}
	gap := c.minGap
	if c.jitter > 0 {
		gap += time.Duration(rand.Int63n(int64(c.jitter) + 1))
	}
	c.nextAt = start.Add(gap)
	c.mu.Unlock()
	return sleep(ctx, time.Until(start))
}

func (c *Client) retry(ctx context.Context, method, url string, body []byte, headers map[string]string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < c.retries; attempt++ {
		if err := c.pace(ctx); err != nil {
			return nil, err
		}
		out, err := c.do(ctx, method, url, body, headers)
		if err == nil {
			return out, nil
		}
		lastErr = err

		var se *statusError
		if errors.As(err, &se) {
			switch {
			case se.code == http.StatusForbidden:
				return nil, fmt.Errorf("%s %s: %w", method, url, ErrBlocked)
			case se.code == http.StatusNotFound || se.code == http.StatusGone:
				return nil, fmt.Errorf("%s %s: %w", method, url, err)
			case se.code == http.StatusTooManyRequests || se.code == http.StatusServiceUnavailable:
				wait := se.retryAfter
				if wait <= 0 {
					wait = backoff(throttleBase, attempt)
				}
				if serr := sleep(ctx, wait); serr != nil {
					return nil, serr
				}
				continue
			}
		}

		if attempt < c.retries-1 {
			if serr := sleep(ctx, backoff(baseBackoff, attempt)); serr != nil {
				return nil, serr
			}
		}
	}
	return nil, fmt.Errorf("%s %s: %w", method, url, lastErr)
}

func (c *Client) do(ctx context.Context, method, url string, body []byte, headers map[string]string) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", acceptLang)
	if method == http.MethodGet {
		req.Header.Set("Accept", acceptHTML)
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Sec-Fetch-User", "?1")
		if ref := origin(url); ref != "" {
			req.Header.Set("Referer", ref+"/")
		}
	} else {
		req.Header.Set("Accept", "application/json, text/plain, */*")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &statusError{code: resp.StatusCode, retryAfter: parseRetryAfter(resp.Header.Get("Retry-After"))}
	}
	return out, nil
}

func backoff(base time.Duration, attempt int) time.Duration {
	return base << attempt
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

func origin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
