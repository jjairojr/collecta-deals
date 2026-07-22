package mypcards

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"opdeals/internal/game"
	"opdeals/internal/logx"
	"opdeals/internal/model"
)

const (
	sourceName      = "mypcards"
	baseURL         = "https://mypcards.com"
	editionsPerPage = 48
	maxEditionPages = 6
	maxPagesPerSet  = 25

	defaultConcurrency = 3
	maxConcurrency     = 4
	setInterval        = 400 * time.Millisecond
)

type Client struct {
	log         *logx.Logger
	concurrency int
	interval    time.Duration
	cfg         game.MyP
}

// New builds a scraper for one game's mypcards.com section. Games whose MyP
// config is nil are not carried on the site and must not reach here.
func New(logger *logx.Logger, concurrency int, g game.Game) *Client {
	if concurrency < 1 {
		concurrency = defaultConcurrency
	}
	if concurrency > maxConcurrency {
		concurrency = maxConcurrency
	}
	cfg := game.MyP{}
	if g.MyP != nil {
		cfg = *g.MyP
	}
	return &Client{log: logger, concurrency: concurrency, interval: setInterval, cfg: cfg}
}

func (c *Client) Name() string { return sourceName }

// Listings drives a headless Chrome through Cloudflare, then scrapes every
// edition of the configured game. It is fail-soft: any browser/challenge
// failure logs and returns no listings rather than failing the whole scan.
func (c *Client) Listings(ctx context.Context) ([]model.BrazilListing, error) {
	if c.cfg.Slug == "" {
		return nil, nil
	}
	b := newBrowser(ctx)
	defer b.close()

	c.log.Printf("BR  mypcards: launching headless Chrome + solving Cloudflare challenge")
	if err := b.solve(ctx); err != nil {
		c.log.Printf("BR  mypcards: skipped — %v", err)
		return nil, nil
	}

	slugs, err := c.editions(ctx, b)
	if err != nil {
		c.log.Printf("BR  mypcards: skipped — editions: %v", err)
		return nil, nil
	}
	if len(slugs) == 0 {
		c.log.Printf("BR  mypcards: no editions found, skipped")
		return nil, nil
	}
	c.log.Printf("BR  mypcards: %d sets to fetch (concurrency %d, via browser)", len(slugs), c.concurrency)

	var (
		mu        sync.Mutex
		out       []model.BrazilListing
		completed int
	)
	limiter := time.NewTicker(c.interval)
	defer limiter.Stop()

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(c.concurrency)
	for _, slug := range slugs {
		slug := slug
		g.Go(func() error {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case <-limiter.C:
			}
			items := c.setListings(gctx, b, slug)
			mu.Lock()
			for _, it := range items {
				out = append(out, it.listing)
			}
			completed++
			done := completed
			mu.Unlock()
			c.log.Printf("BR  [%2d/%d] %-34s %3d cards", done, len(slugs), slug, len(items))
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		c.log.Printf("BR  mypcards interrupted: %v (keeping %d cards)", err, len(out))
	}
	if len(out) == 0 {
		return nil, nil
	}
	c.log.Printf("BR  mypcards: %d cards from %d sets", len(out), len(slugs))
	return out, nil
}

func (c *Client) editions(ctx context.Context, b *browser) ([]string, error) {
	var slugs []string
	seen := make(map[string]bool)
	for page := 1; page <= maxEditionPages; page++ {
		url := fmt.Sprintf("%s/%s/edicoes?page=%d&per-page=%d", baseURL, c.cfg.Slug, page, editionsPerPage)
		doc, err := b.fetch(ctx, url)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			break
		}
		newCount := 0
		for _, s := range parseEditions([]byte(doc), c.cfg.Slug) {
			if seen[s] {
				continue
			}
			seen[s] = true
			slugs = append(slugs, s)
			newCount++
		}
		if newCount == 0 {
			break
		}
	}
	return slugs, nil
}

func (c *Client) setListings(ctx context.Context, b *browser, slug string) []listingItem {
	var all []listingItem
	seen := make(map[string]bool)
	for page := 1; page <= maxPagesPerSet; page++ {
		select {
		case <-ctx.Done():
			return all
		default:
		}
		url := fmt.Sprintf("%s/%s/%s?page=%d", baseURL, c.cfg.Slug, slug, page)
		doc, err := b.fetch(ctx, url)
		if err != nil {
			c.log.Printf("BR  %s p%d failed: %v", slug, page, err)
			break
		}
		newCount := 0
		for _, it := range parseListing([]byte(doc), c.cfg) {
			if seen[it.key] {
				continue
			}
			seen[it.key] = true
			all = append(all, it)
			newCount++
		}
		if newCount == 0 {
			break
		}
	}
	return all
}
