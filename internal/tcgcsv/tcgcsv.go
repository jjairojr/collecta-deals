package tcgcsv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"opdeals/internal/game"
	"opdeals/internal/httpx"
	"opdeals/internal/logx"
	"opdeals/internal/model"
)

const baseURL = "https://tcgcsv.com/tcgplayer"

type Client struct {
	http        *httpx.Client
	log         *logx.Logger
	concurrency int
	market      *game.Market
}

func New(client *httpx.Client, logger *logx.Logger, concurrency int, market *game.Market) *Client {
	return &Client{http: client, log: logger, concurrency: concurrency, market: market}
}

type groupsResponse struct {
	Results []struct {
		GroupID      int    `json:"groupId"`
		Name         string `json:"name"`
		Abbreviation string `json:"abbreviation"`
	} `json:"results"`
}

type groupInfo struct {
	id      int
	setCode string
}

// isDONProduct reports whether a TCGCSV product is a DON!! card. These have an
// empty Number, so they are otherwise dropped by the number filter.
func isDONProduct(name, rarity string) bool {
	return rarity == "DON!!" || strings.HasPrefix(name, "DON!! Card")
}

type productsResponse struct {
	Results []struct {
		ProductID    int    `json:"productId"`
		Name         string `json:"name"`
		ExtendedData []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"extendedData"`
	} `json:"results"`
}

type pricesResponse struct {
	Results []struct {
		ProductID   int      `json:"productId"`
		LowPrice    float64  `json:"lowPrice"`
		MarketPrice *float64 `json:"marketPrice"`
		SubTypeName string   `json:"subTypeName"`
	} `json:"results"`
}

func (c *Client) Prices(ctx context.Context) ([]model.USPrice, error) {
	groups, err := c.groups(ctx)
	if err != nil {
		return nil, err
	}
	c.log.Printf("US  tcgcsv: fetching %d set groups (concurrency %d)", len(groups), c.concurrency)

	var (
		mu        sync.Mutex
		out       []model.USPrice
		completed int
	)
	total := len(groups)
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(c.concurrency)
	for _, grp := range groups {
		grp := grp
		g.Go(func() error {
			prices, err := c.groupPrices(ctx, grp.id, grp.setCode)
			if err != nil {
				return err
			}
			mu.Lock()
			out = append(out, prices...)
			completed++
			done := completed
			mu.Unlock()
			if done%20 == 0 || done == total {
				c.log.Printf("US  [%d/%d] groups fetched", done, total)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	c.log.Printf("US  tcgcsv: %d priced printings from %d groups", len(out), total)
	return out, nil
}

func (c *Client) groups(ctx context.Context) ([]groupInfo, error) {
	body, err := c.http.Get(ctx, fmt.Sprintf("%s/%d/groups", baseURL, c.market.TCGCSVCategoryID))
	if err != nil {
		return nil, err
	}
	var gr groupsResponse
	if err := json.Unmarshal(body, &gr); err != nil {
		return nil, fmt.Errorf("groups: %w", err)
	}
	groups := make([]groupInfo, 0, len(gr.Results))
	for _, g := range gr.Results {
		setCode, ok := c.market.GroupSetCode(g.Abbreviation, g.Name)
		if !ok {
			continue
		}
		groups = append(groups, groupInfo{id: g.GroupID, setCode: setCode})
	}
	return groups, nil
}

type cardMeta struct {
	number  string
	rarity  string
	name    string
	setCode string
}

func (c *Client) groupPrices(ctx context.Context, groupID int, setCode string) ([]model.USPrice, error) {
	meta, err := c.groupProducts(ctx, groupID, setCode)
	if err != nil {
		return nil, err
	}

	body, err := c.http.Get(ctx, fmt.Sprintf("%s/%d/%d/prices", baseURL, c.market.TCGCSVCategoryID, groupID))
	if err != nil {
		return nil, err
	}
	var resp pricesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("prices %d: %w", groupID, err)
	}

	out := make([]model.USPrice, 0, len(resp.Results))
	for _, p := range resp.Results {
		m, ok := meta[p.ProductID]
		if !ok {
			continue
		}
		market := 0.0
		if p.MarketPrice != nil {
			market = *p.MarketPrice
		}
		out = append(out, model.USPrice{
			Number:    model.NormalizeNumber(m.number),
			Name:      m.name,
			Variant:   p.SubTypeName,
			Rarity:    m.rarity,
			SetCode:   m.setCode,
			URL:       fmt.Sprintf("https://www.tcgplayer.com/product/%d", p.ProductID),
			ProductID: p.ProductID,
			MarketUSD: market,
			LowUSD:    p.LowPrice,
		})
	}
	return out, nil
}

func (c *Client) groupProducts(ctx context.Context, groupID int, setCode string) (map[int]cardMeta, error) {
	body, err := c.http.Get(ctx, fmt.Sprintf("%s/%d/%d/products", baseURL, c.market.TCGCSVCategoryID, groupID))
	if err != nil {
		return nil, err
	}
	var resp productsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("products %d: %w", groupID, err)
	}
	meta := make(map[int]cardMeta, len(resp.Results))
	for _, p := range resp.Results {
		var number, rarity string
		for _, e := range p.ExtendedData {
			switch e.Name {
			case "Number":
				number = e.Value
			case "Rarity":
				rarity = e.Value
			}
		}
		// DON!! cards carry no Number; keep them only in a mapped premium set,
		// where they match the Brazil side by set-scoped name. Everything else
		// without a number is dropped as before.
		if number == "" && !(c.market.DON && setCode != "" && isDONProduct(p.Name, rarity)) {
			continue
		}
		meta[p.ProductID] = cardMeta{number: number, rarity: rarity, name: p.Name, setCode: setCode}
	}
	return meta, nil
}
