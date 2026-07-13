package liga

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"opdeals/internal/game"
	"opdeals/internal/httpx"
	"opdeals/internal/logx"
	"opdeals/internal/model"
)

const (
	maxConcurrency = 2
	minInterval    = 500 * time.Millisecond
	retryRounds    = 3
	cooldown       = 4 * time.Second
	stockInterval  = 700 * time.Millisecond
	stockRounds    = 3
	stockCooldown  = 12 * time.Second
	stockWarmup    = 15 * time.Second
	stockLogEvery  = 50
)

// Fetcher retrieves a Liga URL's bytes. It is satisfied by *httpx.Client for
// direct fetches and by *flaresolverr.Router when a host sits behind Cloudflare's
// Managed Challenge and must be fetched through a headless browser.
type Fetcher interface {
	Get(ctx context.Context, url string) ([]byte, error)
}

type Client struct {
	http        Fetcher
	log         *logx.Logger
	game        game.Game
	concurrency int
	interval    time.Duration
	setFilter   map[string]struct{}
}

func New(client Fetcher, logger *logx.Logger, concurrency int, sets []string, g game.Game) *Client {
	if concurrency > maxConcurrency {
		concurrency = maxConcurrency
	}
	if concurrency < 1 {
		concurrency = 1
	}
	var filter map[string]struct{}
	if len(sets) > 0 {
		filter = make(map[string]struct{}, len(sets))
		for _, s := range sets {
			filter[s] = struct{}{}
		}
	}
	return &Client{http: client, log: logger, game: g, concurrency: concurrency, interval: minInterval, setFilter: filter}
}

func (c *Client) Name() string { return c.game.SourceName }

func (c *Client) editionsURL() string { return c.game.LigaBaseURL + "?view=cards/edicoes" }

func (c *Client) Listings(ctx context.Context) ([]model.BrazilListing, error) {
	body, err := c.http.Get(ctx, c.editionsURL())
	if err != nil {
		return nil, fmt.Errorf("editions: %w", err)
	}
	sets := c.filterSets(parseEditions(body))
	if len(sets) == 0 {
		return nil, fmt.Errorf("no editions to fetch")
	}
	c.log.Printf("BR  %s: %d sets to fetch (concurrency %d, %v/req)", c.game.SourceName, len(sets), c.concurrency, c.interval)

	total := len(sets)
	completed := 0
	var out []model.BrazilListing
	pending := sets
	for round := 0; round < retryRounds && len(pending) > 0; round++ {
		if round > 0 {
			c.log.Printf("BR  retry round %d: %d stragglers after %v cooldown", round, len(pending), cooldown)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(cooldown):
			}
		}
		got, failed, err := c.fetchRound(ctx, pending, &completed, total)
		if err != nil {
			return nil, err
		}
		out = append(out, got...)
		pending = failed
	}
	if len(pending) > 0 {
		fmt.Fprintf(os.Stderr, "%s: skipped %d/%d sets after %d rounds: %v\n", c.game.SourceName, len(pending), len(sets), retryRounds, pending)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no listings fetched (%d sets failed)", len(pending))
	}
	c.log.Printf("BR  %s: %d cards from %d/%d sets", c.game.SourceName, len(out), total-len(pending), total)
	return out, nil
}

func (c *Client) fetchRound(ctx context.Context, sets []string, completed *int, total int) ([]model.BrazilListing, []string, error) {
	var (
		mu     sync.Mutex
		out    []model.BrazilListing
		failed []string
	)
	limiter := time.NewTicker(c.interval)
	defer limiter.Stop()

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(c.concurrency)
	for _, set := range sets {
		set := set
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-limiter.C:
			}
			listings, err := c.setListings(ctx, set)
			if err != nil {
				if errors.Is(err, httpx.ErrBlocked) {
					return err
				}
				mu.Lock()
				failed = append(failed, set)
				mu.Unlock()
				c.log.Printf("BR  %-10s failed (%v), will retry", set, err)
				return nil
			}
			mu.Lock()
			out = append(out, listings...)
			*completed++
			done := *completed
			mu.Unlock()
			c.log.Printf("BR  [%2d/%d] %-10s %3d cards", done, total, set, len(listings))
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	return out, failed, nil
}

func (c *Client) filterSets(sets []string) []string {
	if c.setFilter == nil {
		return sets
	}
	out := make([]string, 0, len(c.setFilter))
	for _, s := range sets {
		if _, ok := c.setFilter[s]; ok {
			out = append(out, s)
		}
	}
	return out
}

func (c *Client) StockOf(ctx context.Context, listings []model.BrazilListing) map[string]bool {
	result := make(map[string]bool, len(listings))
	if len(listings) == 0 {
		return result
	}
	c.log.Printf("BR  verifying current stock for %d candidate cards (%v warmup)", len(listings), stockWarmup)
	select {
	case <-ctx.Done():
		return result
	case <-time.After(stockWarmup):
	}

	total := len(listings)
	pending := listings
	for round := 0; round < stockRounds && len(pending) > 0; round++ {
		if round > 0 {
			c.log.Printf("BR  stock retry round %d: %d cards (after %v cooldown)", round, len(pending), stockCooldown)
			select {
			case <-ctx.Done():
				return result
			case <-time.After(stockCooldown):
			}
		}
		var blocked bool
		pending, blocked = c.stockRound(ctx, pending, result, total)
		if blocked {
			c.log.Printf("BR  stock: origin blocked us, aborting verification")
			break
		}
	}
	if len(pending) > 0 {
		c.log.Printf("BR  stock: %d cards left unverified (kept, not dropped)", len(pending))
	}
	c.log.Printf("BR  stock verified: %d/%d have current sellers", countTrue(result), len(result))
	return result
}

func (c *Client) stockRound(ctx context.Context, listings []model.BrazilListing, result map[string]bool, total int) ([]model.BrazilListing, bool) {
	var (
		mu     sync.Mutex
		failed []model.BrazilListing
	)
	limiter := time.NewTicker(stockInterval)
	defer limiter.Stop()

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(c.concurrency)
	for _, l := range listings {
		l := l
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-limiter.C:
			}
			body, err := c.http.Get(ctx, l.URL)
			if err != nil {
				if errors.Is(err, httpx.ErrBlocked) {
					return err
				}
				mu.Lock()
				failed = append(failed, l)
				mu.Unlock()
				return nil
			}
			mu.Lock()
			result[l.URL] = pageHasStock(body)
			done := len(result)
			mu.Unlock()
			if done == total || done%stockLogEvery == 0 {
				c.log.Printf("BR  stock [%d/%d] cards", done, total)
			}
			return nil
		})
	}
	err := g.Wait()
	if errors.Is(err, httpx.ErrBlocked) {
		return failed, true
	}
	if err != nil {
		c.log.Printf("BR  stock round interrupted: %v", err)
	}
	return failed, false
}

const LangEnglish = "2"

// condNM is the Liga Qualid code for Near Mint. Deals compare a Brazil buy price
// against the TCGplayer NM sell price, so the Brazil floor must be NM too — a
// played card's price is not the same product we sell. (Same code the tracking
// buyout ladder uses.)
const condNM = "2"

func (c *Client) EnglishStock(ctx context.Context, listings []model.BrazilListing) map[string]model.EnglishStock {
	detail := c.StockDetail(ctx, listings)
	out := make(map[string]model.EnglishStock, len(detail))
	for url, rows := range detail {
		floor, has := FloorNM(rows, c.game.FloorLangs)
		copies, sellers := floorDepth(rows, c.game.FloorLangs, floor)
		out[url] = model.EnglishStock{InStock: has, FloorBRL: floor, Copies: copies, Sellers: sellers}
	}
	return out
}

// floorDepth counts, among in-stock English NM listings, how many copies sit at
// the floor price and how many distinct stores hold English NM stock — the buy-side
// depth answering "one lucky flip or real supply". Copies counts only listings at
// the floor (unknown-price listings can't be placed on the ladder); Sellers counts
// every distinct store with English NM stock. Returns 0,0 when floor is unknown.
func floorDepth(rows []StoreListing, langs []string, floor float64) (copies, sellers int) {
	if floor <= 0 {
		return 0, 0
	}
	stores := make(map[int]bool)
	for _, r := range rows {
		if r.Condition != condNM || !langAllowed(r.Language, langs) {
			continue
		}
		if r.QtyKnown && r.Quantity <= 0 {
			continue
		}
		stores[r.StoreID] = true
		if r.PriceKnown && r.PriceBRL == floor {
			if r.QtyKnown {
				copies += r.Quantity
			} else {
				copies++
			}
		}
	}
	return copies, len(stores)
}

// Floor returns the lowest in-stock price among listings whose language is in
// langs (empty langs = any language), and whether any qualifying stock exists.
func Floor(rows []StoreListing, langs []string) (float64, bool) {
	return floorFiltered(rows, langs, false)
}

// FloorNM is Floor restricted to Near Mint listings — the buy-side basis for
// deals, so the Brazil floor is grade-matched to the NM US sell price.
func FloorNM(rows []StoreListing, langs []string) (float64, bool) {
	return floorFiltered(rows, langs, true)
}

func floorFiltered(rows []StoreListing, langs []string, nmOnly bool) (float64, bool) {
	low, inStock := 0.0, false
	for _, r := range rows {
		if nmOnly && r.Condition != condNM {
			continue
		}
		if !langAllowed(r.Language, langs) {
			continue
		}
		if r.QtyKnown && r.Quantity <= 0 {
			continue
		}
		inStock = true
		if r.PriceKnown && r.PriceBRL > 0 && (low == 0 || r.PriceBRL < low) {
			low = r.PriceBRL
		}
	}
	return low, inStock
}

func langAllowed(lang string, langs []string) bool {
	if len(langs) == 0 {
		return true
	}
	for _, l := range langs {
		if l == lang {
			return true
		}
	}
	return false
}

func (c *Client) StockDetail(ctx context.Context, listings []model.BrazilListing) map[string][]StoreListing {
	result := make(map[string][]StoreListing, len(listings))
	if len(listings) == 0 {
		return result
	}
	c.log.Printf("BR  fetching per-store stock detail for %d cards (%v warmup)", len(listings), stockWarmup)
	select {
	case <-ctx.Done():
		return result
	case <-time.After(stockWarmup):
	}

	total := len(listings)
	pending := listings
	for round := 0; round < stockRounds && len(pending) > 0; round++ {
		if round > 0 {
			c.log.Printf("BR  detail retry round %d: %d cards (after %v cooldown)", round, len(pending), stockCooldown)
			select {
			case <-ctx.Done():
				return result
			case <-time.After(stockCooldown):
			}
		}
		var blocked bool
		pending, blocked = c.detailRound(ctx, pending, result, total)
		if blocked {
			c.log.Printf("BR  detail: origin blocked us, aborting detail fetch")
			break
		}
	}
	if len(pending) > 0 {
		c.log.Printf("BR  detail: %d cards left unfetched after %d rounds", len(pending), stockRounds)
	}
	c.log.Printf("BR  detail fetched for %d/%d cards", len(result), len(listings))
	return result
}

func (c *Client) detailRound(ctx context.Context, listings []model.BrazilListing, result map[string][]StoreListing, total int) ([]model.BrazilListing, bool) {
	var (
		mu     sync.Mutex
		failed []model.BrazilListing
	)
	limiter := time.NewTicker(stockInterval)
	defer limiter.Stop()

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(c.concurrency)
	for _, l := range listings {
		l := l
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-limiter.C:
			}
			body, err := c.http.Get(ctx, l.URL)
			if err != nil {
				if errors.Is(err, httpx.ErrBlocked) {
					return err
				}
				mu.Lock()
				failed = append(failed, l)
				mu.Unlock()
				return nil
			}
			qtyAtlas := c.fetchAtlas(ctx, quantAtlasURL(body))
			priceAtlas := c.fetchAtlas(ctx, priceAtlasURL(body))
			listings, perr := parseCardStock(body, qtyAtlas, priceAtlas)
			if perr != nil {
				c.log.Printf("BR  %-14s detail parse: %v", l.Number, perr)
			}
			mu.Lock()
			result[l.URL] = listings
			done := len(result)
			mu.Unlock()
			if done == total || done%stockLogEvery == 0 {
				c.log.Printf("BR  detail [%d/%d] cards", done, total)
			}
			return nil
		})
	}
	err := g.Wait()
	if errors.Is(err, httpx.ErrBlocked) {
		return failed, true
	}
	if err != nil {
		c.log.Printf("BR  detail round interrupted: %v", err)
	}
	return failed, false
}

func (c *Client) fetchAtlas(ctx context.Context, u string) image.Image {
	if u == "" {
		return nil
	}
	raw, err := c.http.Get(ctx, u)
	if err != nil {
		return nil
	}
	img, ok := decodeAtlas(raw)
	if !ok {
		return nil
	}
	return img
}

func countTrue(m map[string]bool) int {
	n := 0
	for _, v := range m {
		if v {
			n++
		}
	}
	return n
}

func (c *Client) setListings(ctx context.Context, setCode string) ([]model.BrazilListing, error) {
	url := c.game.LigaBaseURL + "?view=cards/search&card=ed=" + setCode + "%20searchprod=0&tipo=1"
	body, err := c.http.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("set %s: %w", setCode, err)
	}
	return parseCards(body, setCode, c.game)
}
