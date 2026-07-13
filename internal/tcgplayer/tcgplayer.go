package tcgplayer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"opdeals/internal/httpx"
	"opdeals/internal/logx"
)

const (
	listingsURL = "https://mp-search-api.tcgplayer.com/v1/product/%d/listings"
	concurrency = 3
	interval    = 350 * time.Millisecond
	rounds      = 2
	cooldown    = 8 * time.Second
)

// size is generous because we discard every non-gold-star and Japanese listing
// before picking a price; gold star sellers can sit well below the cheapest rows.
var requestBody = []byte(`{"filters":{"term":{"sellerStatus":"Live","channelId":0},"range":{"quantity":{"gte":1}}},"from":0,"size":50,"sort":{"field":"price+shipping","order":"asc"},"context":{"shippingCountry":"US","cart":{}}}`)

type Client struct {
	http *httpx.Client
	log  *logx.Logger
}

func New(client *httpx.Client, logger *logx.Logger) *Client {
	return &Client{http: client, log: logger}
}

type listingsResponse struct {
	Results []struct {
		Results []listing `json:"results"`
	} `json:"results"`
}

type listing struct {
	Price      float64 `json:"price"`
	Quantity   float64 `json:"quantity"` // wire sends 1.0; unmarshalling into int fails the whole response
	Condition  string  `json:"condition"`
	GoldSeller bool    `json:"goldSeller"`
	Language   string  `json:"language"`
	CustomData struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"customData"`
}

// PriceInfo is the live TCGplayer floor for a product plus the depth backing it:
// Listings is how many trusted (gold-star, English, NM-sellable) sellers sit at or
// near the floor and Qty the total copies among them. Depth turns a bare price into
// a trust signal — a floor backed by one listing is an outlier, not a market.
type PriceInfo struct {
	Price    float64
	Listings int
	Qty      int
}

func (c *Client) LowestPrices(ctx context.Context, productIDs []int) map[int]PriceInfo {
	result := make(map[int]PriceInfo, len(productIDs))
	if len(productIDs) == 0 {
		return result
	}
	c.log.Printf("US  fetching live TCGplayer listings for %d products", len(productIDs))

	pending := productIDs
	for round := 0; round < rounds && len(pending) > 0; round++ {
		if round > 0 {
			c.log.Printf("US  live price retry round %d: %d products (after %v cooldown)", round, len(pending), cooldown)
			select {
			case <-ctx.Done():
				return result
			case <-time.After(cooldown):
			}
		}
		pending = c.round(ctx, pending, result)
	}
	c.log.Printf("US  live prices: %d/%d products priced from current listings", len(result), len(productIDs))
	return result
}

func (c *Client) round(ctx context.Context, ids []int, result map[int]PriceInfo) []int {
	var (
		mu     sync.Mutex
		failed []int
	)
	limiter := time.NewTicker(interval)
	defer limiter.Stop()

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)
	for _, id := range ids {
		id := id
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-limiter.C:
			}
			info, err := c.lowest(ctx, id)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed = append(failed, id)
				return nil
			}
			if info.Price > 0 {
				result[id] = info
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		c.log.Printf("US  live price round interrupted: %v", err)
	}
	return failed
}

func (c *Client) lowest(ctx context.Context, productID int) (PriceInfo, error) {
	body, err := c.http.PostJSON(ctx, fmt.Sprintf(listingsURL, productID), requestBody, map[string]string{
		"Content-Type": "application/json",
		"Origin":       "https://www.tcgplayer.com",
		"Referer":      "https://www.tcgplayer.com/",
	})
	if err != nil {
		return PriceInfo{}, err
	}
	var resp listingsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return PriceInfo{}, err
	}
	if len(resp.Results) == 0 {
		return PriceInfo{}, nil
	}
	return pickPrice(resp.Results[0].Results), nil
}

// pickPrice chooses the price we trust for a product and the depth backing it. It
// only considers gold star sellers and never a foreign-language listing (Japanese
// or Chinese prints sell for far less than the English card we track). The listing's
// language field is unreliable — sellers file foreign cards under the English product
// with language "English" — so the title/description text is the real signal. Within
// that trusted set it prefers Near Mint, falling back to the cheapest gold-star
// listing of any grade. Listings/Qty count the tier the returned price came from
// (NM when any NM exists, else the overall gold-star tier) so the depth describes
// the same market the price does.
func pickPrice(listings []listing) PriceInfo {
	sellable, overall := 0.0, 0.0
	var nmCount, nmQty, allCount, allQty int
	for _, l := range listings {
		if l.Price <= 0 || !l.GoldSeller || isForeignLanguage(l) {
			continue
		}
		allCount++
		allQty += int(l.Quantity)
		if overall == 0 || l.Price < overall {
			overall = l.Price
		}
		if isSellableGrade(l.Condition) {
			nmCount++
			nmQty += int(l.Quantity)
			if sellable == 0 || l.Price < sellable {
				sellable = l.Price
			}
		}
	}
	if sellable > 0 {
		return PriceInfo{Price: sellable, Listings: nmCount, Qty: nmQty}
	}
	return PriceInfo{Price: overall, Listings: allCount, Qty: allQty}
}

func isSellableGrade(condition string) bool {
	return strings.Contains(strings.ToLower(condition), "near mint")
}

// foreignMarkers are language tags sellers write into a listing's title or
// description even though they filed it under the English product with language
// "English". A Japanese or Chinese print of the same card sells well below the
// English card we track, so a floor built on one is not a real comparable.
// Abbreviations count too: sellers routinely tag Japanese prints "JPN"/"JP"
// rather than spelling out "Japanese" (e.g. "**JPN** Edward.Newgate (SP)").
var foreignMarkers = []string{"japanese", "japan", "jpn", "chinese", "korean"}

func isForeignLanguage(l listing) bool {
	if l.Language != "" && !strings.EqualFold(l.Language, "English") {
		return true
	}
	text := strings.ToLower(l.CustomData.Title + " " + l.CustomData.Description)
	for _, marker := range foreignMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
