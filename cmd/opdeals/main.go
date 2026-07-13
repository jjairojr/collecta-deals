package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"opdeals/internal/compare"
	"opdeals/internal/flaresolverr"
	"opdeals/internal/game"
	"opdeals/internal/logx"
	"opdeals/internal/pipeline"
	"opdeals/internal/report"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	minMargin := flag.Float64("min-margin", 30, "minimum margin percent to show")
	minPrice := flag.Float64("min-price", 1.0, "minimum US sell price to consider")
	fxOverride := flag.Float64("fx", 0, "BRL->USD rate override (0 = fetch live)")
	setsFlag := flag.String("sets", "", "comma-separated Liga set codes to limit (e.g. OP-01,OP-02)")
	concurrency := flag.Int("concurrency", 8, "max concurrent requests per source")
	usPrice := flag.String("us-price", "low", "US price basis: market or low")
	sortBy := flag.String("sort", "margin", "sort by: margin or profit")
	limit := flag.Int("limit", 0, "max rows to print (0 = all)")
	verifyFloor := flag.Float64("verify-floor", 10, "verify stock + fetch live TCGplayer prices for candidate deals selling above this US$ (0 = skip)")
	livePrices := flag.Bool("live-prices", true, "use live TCGplayer listing prices for candidate deals (vs TCGCSV snapshot)")
	myPCards := flag.Bool("mypcards", false, "also scrape MyP Cards (2nd BR source; needs Chrome installed, adds ~1-3 min)")
	timeout := flag.Duration("timeout", 30*time.Second, "per-request timeout")
	quiet := flag.Bool("quiet", false, "suppress progress logs on stderr")
	gameID := flag.String("game", "onepiece", "game to scan: onepiece, pokemon, riftbound, lorcana or gundam")
	flaresolverrURL := flag.String("flaresolverr", "", "FlareSolverr endpoint URL (e.g. http://localhost:8191) for Cloudflare-challenged Liga hosts (empty = direct)")
	flag.Parse()

	logWriter := io.Writer(os.Stderr)
	if *quiet {
		logWriter = io.Discard
	}
	logger := logx.New(logWriter)

	g := game.ByID(*gameID)

	var solver *flaresolverr.Client
	if *flaresolverrURL != "" {
		solver = flaresolverr.New(*flaresolverrURL)
		defer solver.Close(context.Background())
	}

	start := time.Now()
	logger.Printf("scan starting (%s, min-margin %.0f%%, min-price $%.2f)", g.ID, *minMargin, *minPrice)
	snap, err := pipeline.Fetch(context.Background(), logger, pipeline.Options{
		Game:         g,
		Sets:         parseSets(*setsFlag),
		FXOverride:   *fxOverride,
		Concurrency:  *concurrency,
		Timeout:      *timeout,
		VerifyFloor:  *verifyFloor,
		LivePrices:   *livePrices,
		MyPCards:     *myPCards,
		FlareSolverr: solver,
	})
	if err != nil {
		return err
	}

	deals := compare.Deals(snap.Listings, snap.Prices, compare.Options{
		FXRate:         snap.FXRate,
		MinMargin:      *minMargin,
		MinPrice:       *minPrice,
		UsePrice:       *usPrice,
		SortBy:         *sortBy,
		Limit:          *limit,
		RequireInStock: *verifyFloor > 0,
		Matcher:        compare.MatcherFor(g),
	})
	logger.Printf("%d deals after filters; done in %s", len(deals), time.Since(start).Round(time.Second))
	report.Print(os.Stdout, deals, snap.FXRate)
	return nil
}

func parseSets(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
