package liga

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"opdeals/internal/httpx"
)

var (
	prodTileRe   = regexp.MustCompile(`prod/view&(?:amp;)?pcode=(\d+)&(?:amp;)?prod=([^"]+)"`)
	prodStockRe  = regexp.MustCompile(`(?s)var prod_stock\s*=\s*(\[.*?\]);`)
	prodStoresRe = regexp.MustCompile(`(?s)var prod_stores\s*=\s*(\{.*?\});`)
)

type SealedProduct struct {
	PCode   string
	Name    string
	Type    string
	SetCode string
	URL     string
}

func (c *Client) sealedURL(pcode string) string {
	return c.game.LigaBaseURL + "?view=prod/view&pcode=" + pcode
}

func (c *Client) SealedListings(ctx context.Context) ([]SealedProduct, error) {
	seen := make(map[string]struct{})
	var out []SealedProduct
	cats := c.game.SealedCategories
	limiter := time.NewTicker(c.interval)
	defer limiter.Stop()
	for _, cat := range cats {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case <-limiter.C:
		}
		u := c.game.LigaBaseURL + "?view=cards/search&card=categ=" + cat.ID + "%20searchprod=1&category=products"
		body, err := c.http.Get(ctx, u)
		if err != nil {
			if errors.Is(err, httpx.ErrBlocked) {
				return out, fmt.Errorf("sealed catalog: %w", err)
			}
			c.log.Printf("BR  sealed category %s (%s): %v", cat.ID, cat.Type, err)
			continue
		}
		before := len(out)
		for _, p := range c.parseSealedTiles(body, cat.Type) {
			if _, ok := seen[p.PCode]; ok {
				continue
			}
			seen[p.PCode] = struct{}{}
			out = append(out, p)
		}
		c.log.Printf("BR  sealed %-13s (categ %s): %d products", cat.Type, cat.ID, len(out)-before)
	}
	c.log.Printf("BR  sealed catalog: %d products across %d categories", len(out), len(cats))
	return out, nil
}

func (c *Client) parseSealedTiles(body []byte, typ string) []SealedProduct {
	matches := prodTileRe.FindAllSubmatch(body, -1)
	out := make([]SealedProduct, 0, len(matches))
	for _, m := range matches {
		pcode := string(m[1])
		name := strings.TrimSpace(html.UnescapeString(string(m[2])))
		out = append(out, SealedProduct{
			PCode:   pcode,
			Name:    name,
			Type:    typ,
			SetCode: sealedSetCode(name, c.game.SealedSetCodeRe),
			URL:     c.sealedURL(pcode),
		})
	}
	return out
}

func sealedSetCode(name string, re *regexp.Regexp) string {
	if re == nil {
		return ""
	}
	m := re.FindString(name)
	if m == "" {
		return ""
	}
	m = strings.ToUpper(m)
	if !strings.Contains(m, "-") {
		for i := 2; i < len(m); i++ {
			if m[i] >= '0' && m[i] <= '9' {
				return m[:i] + "-" + m[i:]
			}
		}
	}
	return m
}

func (c *Client) SealedDetail(ctx context.Context, products []SealedProduct) map[string][]StoreListing {
	result := make(map[string][]StoreListing, len(products))
	if len(products) == 0 {
		return result
	}
	c.log.Printf("BR  fetching per-store stock for %d sealed products (%v warmup)", len(products), stockWarmup)
	select {
	case <-ctx.Done():
		return result
	case <-time.After(stockWarmup):
	}

	total := len(products)
	pending := products
	for round := 0; round < stockRounds && len(pending) > 0; round++ {
		if round > 0 {
			c.log.Printf("BR  sealed detail retry round %d: %d products (after %v cooldown)", round, len(pending), stockCooldown)
			select {
			case <-ctx.Done():
				return result
			case <-time.After(stockCooldown):
			}
		}
		var blocked bool
		pending, blocked = c.sealedDetailRound(ctx, pending, result, total)
		if blocked {
			c.log.Printf("BR  sealed detail: origin blocked us, aborting detail fetch")
			break
		}
	}
	if len(pending) > 0 {
		c.log.Printf("BR  sealed detail: %d products left unfetched after %d rounds", len(pending), stockRounds)
	}
	c.log.Printf("BR  sealed detail fetched for %d/%d products", len(result), len(products))
	return result
}

func (c *Client) sealedDetailRound(ctx context.Context, products []SealedProduct, result map[string][]StoreListing, total int) ([]SealedProduct, bool) {
	var (
		mu     sync.Mutex
		failed []SealedProduct
	)
	limiter := time.NewTicker(stockInterval)
	defer limiter.Stop()

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(c.concurrency)
	for _, p := range products {
		p := p
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-limiter.C:
			}
			body, err := c.http.Get(ctx, p.URL)
			if err != nil {
				if errors.Is(err, httpx.ErrBlocked) {
					return err
				}
				mu.Lock()
				failed = append(failed, p)
				mu.Unlock()
				return nil
			}
			qtyAtlas := c.fetchAtlas(ctx, quantAtlasURL(body))
			priceAtlas := c.fetchAtlas(ctx, priceAtlasURL(body))
			listings := parseProductStock(body, p.PCode, qtyAtlas, priceAtlas)
			mu.Lock()
			result[p.URL] = listings
			done := len(result)
			mu.Unlock()
			if done == total || done%stockLogEvery == 0 {
				c.log.Printf("BR  sealed detail [%d/%d] products", done, total)
			}
			return nil
		})
	}
	err := g.Wait()
	if errors.Is(err, httpx.ErrBlocked) {
		return failed, true
	}
	if err != nil {
		c.log.Printf("BR  sealed detail round interrupted: %v", err)
	}
	return failed, false
}

func parseProductStock(body []byte, pcode string, qtyAtlas, priceAtlas image.Image) []StoreListing {
	sm := prodStockRe.FindSubmatch(body)
	if sm == nil {
		return nil
	}
	var stock []rawStock
	if err := json.Unmarshal(sm[1], &stock); err != nil {
		return nil
	}
	stores := map[string]rawStore{}
	if stm := prodStoresRe.FindSubmatch(body); stm != nil {
		_ = json.Unmarshal(stm[1], &stores)
	}
	classPos := styleClassPositions(body)

	out := make([]StoreListing, 0, len(stock))
	for _, s := range stock {
		qty, known := decodeQuantity(qtyAtlas, classPos, s.QuantCss)
		store := stores[strconv.Itoa(s.LjID)]
		price, priceKnown := listingPrice(s, priceAtlas, classPos)
		out = append(out, StoreListing{
			StoreID:    s.LjID,
			StoreName:  store.Name,
			StoreCity:  store.Cidade,
			StoreUF:    store.UF,
			Number:     pcode,
			Condition:  s.Qualid,
			Language:   s.Idioma,
			Quantity:   qty,
			QtyKnown:   known,
			PriceBRL:   price,
			PriceKnown: priceKnown,
		})
	}
	return out
}
