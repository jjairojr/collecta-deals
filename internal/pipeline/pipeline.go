package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"opdeals/internal/compare"
	"opdeals/internal/flaresolverr"
	"opdeals/internal/fx"
	"opdeals/internal/game"
	"opdeals/internal/httpx"
	"opdeals/internal/liga"
	"opdeals/internal/logx"
	"opdeals/internal/model"
	"opdeals/internal/mypcards"
	"opdeals/internal/tcgcsv"
	"opdeals/internal/tcgplayer"
)

type Options struct {
	Game        game.Game
	Sets        []string
	FXOverride  float64
	Concurrency int
	Timeout     time.Duration
	VerifyFloor float64
	LivePrices  bool
	MyPCards    bool
	// FlareSolverr, when set, fetches Cloudflare-challenged Liga hosts through a
	// headless browser. Nil means fetch every host directly. Optional.
	FlareSolverr *flaresolverr.Client
	// HeldKeys returns the match keys of portfolio cards that should be
	// live-priced on TCGplayer alongside candidate deals, so cards you already
	// own get the same live floor instead of a stale TCGCSV baseline. Keys must
	// be built with the same game's Matcher.LookupKey. Optional.
	HeldKeys func() []string
}

func Fetch(ctx context.Context, logger *logx.Logger, opts Options) (model.Snapshot, error) {
	if opts.Concurrency < 1 {
		opts.Concurrency = 8
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}

	client, err := httpx.New(opts.Timeout)
	if err != nil {
		return model.Snapshot{}, err
	}
	ligaHTTP, err := httpx.NewThrottled(opts.Timeout)
	if err != nil {
		return model.Snapshot{}, err
	}

	gm := opts.Game
	if gm.ID == "" {
		gm = game.OnePiece()
	}
	matcher := compare.MatcherFor(gm)
	var ligaFetcher liga.Fetcher = ligaHTTP
	if opts.FlareSolverr != nil && gm.Challenged {
		ligaFetcher = flaresolverr.NewRouter(ligaHTTP, opts.FlareSolverr, gm.LigaHosts)
	}
	sources := []model.BrazilSource{liga.New(ligaFetcher, logger, opts.Concurrency, opts.Sets, gm)}
	if opts.MyPCards && gm.MyP != nil {
		sources = append(sources, mypcards.New(logger, opts.Concurrency, gm))
	}
	us := tcgcsv.New(client, logger, opts.Concurrency, gm.Market)
	tcg := tcgplayer.New(client, logger)

	var (
		listings []model.BrazilListing
		prices   []model.USPrice
		rate     float64
	)

	start := time.Now()
	g, gctx := errgroup.WithContext(ctx)
	bySource := make([][]model.BrazilListing, len(sources))
	for i, src := range sources {
		g.Go(func() error {
			l, err := src.Listings(gctx)
			if err != nil {
				return fmt.Errorf("%s: %w", src.Name(), err)
			}
			bySource[i] = l
			return nil
		})
	}
	g.Go(func() error {
		p, err := us.Prices(gctx)
		if err != nil {
			return fmt.Errorf("tcgcsv: %w", err)
		}
		prices = p
		return nil
	})
	g.Go(func() error {
		r, err := fx.Rate(gctx, client, logger, opts.FXOverride)
		if err != nil {
			return err
		}
		rate = r
		return nil
	})
	if err := g.Wait(); err != nil {
		return model.Snapshot{}, err
	}
	for i, ls := range bySource {
		listings = append(listings, ls...)
		if opts.MyPCards || len(sources) > 1 {
			logger.Printf("BR  %s: %d listings", sources[i].Name(), len(ls))
		}
	}
	logger.Printf("fetched %d BR listings, %d US printings in %s", len(listings), len(prices), time.Since(start).Round(time.Second))

	if opts.VerifyFloor > 0 {
		brs, usPrices := compare.CandidateListings(listings, prices, rate, opts.VerifyFloor, matcher)
		candidateIDs := uniqueProductIDs(usPrices)
		ids := withHeldProductIDs(candidateIDs, prices, opts.HeldKeys, matcher)
		if len(ids) > len(candidateIDs) {
			logger.Printf("US  portfolio: +%d held products to live-price", len(ids)-len(candidateIDs))
		}

		var (
			live    map[int]tcgplayer.PriceInfo
			stock   = make(map[string]model.EnglishStock)
			stockMu sync.Mutex
		)
		eg, ectx := errgroup.WithContext(ctx)
		for _, src := range sources {
			sub := listingsForSource(brs, src.Name())
			if len(sub) == 0 {
				continue
			}
			if sp, ok := src.(model.StockPricer); ok {
				eg.Go(func() error {
					m := sp.EnglishStock(ectx, sub)
					stockMu.Lock()
					for k, v := range m {
						stock[k] = v
					}
					stockMu.Unlock()
					return nil
				})
				continue
			}
			if sv, ok := src.(model.StockVerifier); ok {
				eg.Go(func() error {
					m := sv.StockOf(ectx, sub)
					stockMu.Lock()
					for k, v := range m {
						stock[k] = model.EnglishStock{InStock: v}
					}
					stockMu.Unlock()
					return nil
				})
			}
		}
		if opts.LivePrices {
			eg.Go(func() error {
				live = tcg.LowestPrices(ectx, ids)
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			return model.Snapshot{}, err
		}

		if len(live) > 0 {
			for i := range prices {
				if v, ok := live[prices[i].ProductID]; ok {
					prices[i].LiveUSD = v.Price
					prices[i].LiveListings = v.Listings
					prices[i].LiveQty = v.Qty
				}
			}
		}
		checked, dropped := 0, 0
		for i := range listings {
			info, ok := stock[listings[i].URL]
			if !ok {
				continue
			}
			listings[i].StockChecked = true
			listings[i].InStock = info.InStock
			listings[i].FloorCopies = info.Copies
			listings[i].Sellers = info.Sellers
			if info.InStock && info.FloorBRL > 0 {
				listings[i].LowBRL = info.FloorBRL
			}
			checked++
			if !info.InStock {
				dropped++
			}
		}
		logger.Printf("enrichment: %d live US prices; %d English-stock-checked, %d no-English-stock excluded", len(live), checked, dropped)
	}

	return model.Snapshot{
		Listings:  listings,
		Prices:    prices,
		FXRate:    rate,
		UpdatedAt: time.Now(),
	}, nil
}

func listingsForSource(listings []model.BrazilListing, source string) []model.BrazilListing {
	out := make([]model.BrazilListing, 0, len(listings))
	for _, l := range listings {
		if l.Source == source {
			out = append(out, l)
		}
	}
	return out
}

// withHeldProductIDs unions the portfolio's held-card product IDs into ids so the
// live TCGplayer price fetch covers cards you own, not just candidate deals. Held
// cards are resolved to a product via the same match key the deals path uses.
func withHeldProductIDs(ids []int, prices []model.USPrice, heldKeys func() []string, m compare.Matcher) []int {
	if heldKeys == nil {
		return ids
	}
	keys := heldKeys()
	if len(keys) == 0 {
		return ids
	}
	seen := make(map[int]bool, len(ids)+len(keys))
	for _, id := range ids {
		seen[id] = true
	}
	index := compare.USDIndex(prices, m)
	for _, key := range keys {
		p, ok := index[key]
		if !ok || p.ProductID == 0 || seen[p.ProductID] {
			continue
		}
		seen[p.ProductID] = true
		ids = append(ids, p.ProductID)
	}
	return ids
}

func uniqueProductIDs(prices []model.USPrice) []int {
	seen := make(map[int]struct{}, len(prices))
	ids := make([]int, 0, len(prices))
	for _, p := range prices {
		if p.ProductID == 0 {
			continue
		}
		if _, ok := seen[p.ProductID]; ok {
			continue
		}
		seen[p.ProductID] = struct{}{}
		ids = append(ids, p.ProductID)
	}
	return ids
}
