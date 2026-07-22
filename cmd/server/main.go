package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"opdeals/internal/api"
	"opdeals/internal/cardimg"
	"opdeals/internal/compare"
	"opdeals/internal/flaresolverr"
	"opdeals/internal/game"
	"opdeals/internal/httpx"
	"opdeals/internal/liga"
	"opdeals/internal/logx"
	"opdeals/internal/pipeline"
	"opdeals/internal/quotes"
	"opdeals/internal/store"
	"opdeals/internal/tracking"
	"opdeals/internal/trades"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	webDir := flag.String("web", "web/dist", "directory of built frontend to serve (empty to disable)")
	cachePath := flag.String("cache", "data/snapshot.json", "path to persist the scan snapshot")
	tradesPath := flag.String("trades", "data/trades.json", "path to the trade ledger (portfolio)")
	quotesPath := flag.String("quotes", "data/quotes.json", "path to the saved buy quotes (orçamentos)")
	setsFlag := flag.String("sets", "", "comma-separated Liga set codes to limit (default all)")
	fxOverride := flag.Float64("fx", 0, "BRL->USD rate override (0 = fetch live)")
	concurrency := flag.Int("concurrency", 8, "max concurrent requests per source")
	verifyFloor := flag.Float64("verify-floor", 10, "verify stock + fetch live TCGplayer prices for candidate deals selling above this US$ (0 = skip)")
	livePrices := flag.Bool("live-prices", true, "use live TCGplayer listing prices for candidate deals (vs TCGCSV snapshot)")
	myPCards := flag.Bool("mypcards", false, "also scrape MyP Cards (2nd BR source; needs Chrome installed, adds ~1-3 min)")
	refreshInterval := flag.Duration("refresh-interval", 2*time.Hour, "auto-refresh interval for the deals snapshot (0 = disabled)")
	timeout := flag.Duration("timeout", 30*time.Second, "per-request timeout")
	track := flag.Bool("track", true, "capture daily collection tracking snapshots")
	trackSet := flag.String("track-set", "OP-16,OP-15,OP-14,OP-13,OP-12,OP-11,OP-10,OP-09,OP-08,OP-07,OP-06,OP-05,OP-04,OP-03,OP-02,OP-01,EB-04,EB03,EB02,EB01,PRB2,PRB,ST30,ST29,ST28,ST27,ST26,ST25,ST24,ST23,ST22,ST21,ST20,ST19,ST18,ST17,ST16,ST15,ST14,ST-13,ST-12,ST-11,ST-10,ST-09,ST-08,ST-07,ST-06,ST-05,ST-04,ST-03,ST-02,ST-01", "comma-separated Liga set codes to track daily")
	captureOnce := flag.Bool("capture-once", false, "run one tracking capture then exit (for cron); does not start the HTTP server")
	trackSchedule := flag.Bool("track-schedule", false, "auto-capture tracking snapshots on track-interval; off by default — trigger manually via POST /api/tracking/capture or -capture-once")
	backfillImages := flag.Bool("backfill-images", false, "resolve and cache card image URLs for all tracked cards then exit")
	trackDir := flag.String("track-dir", "data/tracking", "directory to store daily tracking snapshots")
	trackSealed := flag.Bool("track-sealed", true, "also capture sealed-product (boxes/packs/decks) per-store stock as the SEALED set")
	captureSealedOnce := flag.Bool("capture-sealed-once", false, "run one sealed-product capture then exit (skips the singles scan)")
	trackTZ := flag.String("track-tz", "America/Sao_Paulo", "timezone defining the tracking calendar day")
	trackInterval := flag.Duration("track-interval", 6*time.Hour, "how often to capture a tracking snapshot (>=24h keeps the daily model)")
	trackMinPrice := flag.Float64("track-min-price", 1, "skip tracking cards whose lowest BRL price is below this (bulk commons)")
	pkmTrack := flag.Bool("pkm-track", true, "capture Pokémon (ligapokemon.com.br) tracking snapshots")
	pkmTrackDir := flag.String("pkm-track-dir", "data/tracking-pkm", "directory to store Pokémon tracking snapshots")
	pkmTradesPath := flag.String("pkm-trades", "data/trades-pkm.json", "path to the Pokémon trade ledger (portfolio)")
	pkmQuotesPath := flag.String("pkm-quotes", "data/quotes-pkm.json", "path to the Pokémon saved buy quotes")
	pkmTrackSet := flag.String("pkm-track-set", strings.Join(game.Pokemon().DefaultTrackSets, ","), "comma-separated Liga set codes to track daily for Pokémon")
	pkmTrackSealed := flag.Bool("pkm-track-sealed", true, "also capture Pokémon sealed products")
	pkmCaptureOnce := flag.Bool("pkm-capture-once", false, "run one Pokémon tracking capture then exit")
	pkmCaptureSealedOnce := flag.Bool("pkm-capture-sealed-once", false, "run one Pokémon sealed capture then exit")
	rftTrack := flag.Bool("rft-track", true, "capture Riftbound (ligariftbound.com.br) tracking snapshots")
	rftTrackDir := flag.String("rft-track-dir", "data/tracking-rft", "directory to store Riftbound tracking snapshots")
	rftTradesPath := flag.String("rft-trades", "data/trades-rft.json", "path to the Riftbound trade ledger (portfolio)")
	rftQuotesPath := flag.String("rft-quotes", "data/quotes-rft.json", "path to the Riftbound saved buy quotes")
	rftTrackSet := flag.String("rft-track-set", strings.Join(game.Riftbound().DefaultTrackSets, ","), "comma-separated Liga set codes to track daily for Riftbound")
	rftTrackSealed := flag.Bool("rft-track-sealed", true, "also capture Riftbound sealed products")
	rftCaptureOnce := flag.Bool("rft-capture-once", false, "run one Riftbound tracking capture then exit")
	rftCaptureSealedOnce := flag.Bool("rft-capture-sealed-once", false, "run one Riftbound sealed capture then exit")
	rftCache := flag.String("rft-cache", "data/snapshot-rft.json", "path to persist the Riftbound deals snapshot")
	rftDeals := flag.Bool("rft-deals", true, "run the Riftbound US deals pipeline")
	lorTrack := flag.Bool("lor-track", true, "capture Lorcana (ligalorcana.com.br) tracking snapshots")
	lorTrackDir := flag.String("lor-track-dir", "data/tracking-lor", "directory to store Lorcana tracking snapshots")
	lorTradesPath := flag.String("lor-trades", "data/trades-lor.json", "path to the Lorcana trade ledger (portfolio)")
	lorQuotesPath := flag.String("lor-quotes", "data/quotes-lor.json", "path to the Lorcana saved buy quotes")
	lorTrackSet := flag.String("lor-track-set", strings.Join(game.Lorcana().DefaultTrackSets, ","), "comma-separated Liga set codes to track daily for Lorcana")
	lorTrackSealed := flag.Bool("lor-track-sealed", true, "also capture Lorcana sealed products")
	lorCaptureOnce := flag.Bool("lor-capture-once", false, "run one Lorcana tracking capture then exit")
	lorCaptureSealedOnce := flag.Bool("lor-capture-sealed-once", false, "run one Lorcana sealed capture then exit")
	lorCache := flag.String("lor-cache", "data/snapshot-lor.json", "path to persist the Lorcana deals snapshot")
	lorDeals := flag.Bool("lor-deals", true, "run the Lorcana US deals pipeline")
	gndTrack := flag.Bool("gnd-track", true, "capture Gundam (ligagundam.com.br) tracking snapshots")
	gndTrackDir := flag.String("gnd-track-dir", "data/tracking-gnd", "directory to store Gundam tracking snapshots")
	gndTradesPath := flag.String("gnd-trades", "data/trades-gnd.json", "path to the Gundam trade ledger (portfolio)")
	gndQuotesPath := flag.String("gnd-quotes", "data/quotes-gnd.json", "path to the Gundam saved buy quotes")
	gndTrackSet := flag.String("gnd-track-set", strings.Join(game.Gundam().DefaultTrackSets, ","), "comma-separated Liga set codes to track daily for Gundam")
	gndTrackSealed := flag.Bool("gnd-track-sealed", true, "also capture Gundam sealed products")
	gndCaptureOnce := flag.Bool("gnd-capture-once", false, "run one Gundam tracking capture then exit")
	gndCaptureSealedOnce := flag.Bool("gnd-capture-sealed-once", false, "run one Gundam sealed capture then exit")
	gndCache := flag.String("gnd-cache", "data/snapshot-gnd.json", "path to persist the Gundam deals snapshot")
	gndDeals := flag.Bool("gnd-deals", true, "run the Gundam US deals pipeline")
	schedule := flag.Bool("schedule", false, "run all deals refreshes and tracking captures sequentially in one Liga lane (replaces -track-schedule and the per-store refresh tickers)")
	scheduleInterval := flag.Duration("schedule-interval", 6*time.Hour, "target cadence of a full scheduler cycle (all games, deals + captures)")
	scheduleGap := flag.Duration("schedule-gap", 2*time.Minute, "pause between scheduler jobs so rate-limit windows reset")
	flaresolverrURL := flag.String("flaresolverr", "", "FlareSolverr endpoint URL, e.g. http://flaresolverr:8191, used to fetch Cloudflare-challenged Liga hosts (empty = direct)")
	serveOnly := flag.Bool("serve-only", false, "read-only mode for a prod instance fed by local scrapes: load snapshots and serve, never scrape (no schedulers, no background deals refresh, refresh/capture endpoints disabled)")
	adminToken := flag.String("admin-token", os.Getenv("ADMIN_TOKEN"), "token for POST /api/admin/reload (reload deals snapshots from disk after a sync); empty disables the endpoint")
	flag.Parse()

	if *serveOnly {
		*schedule = false
		*trackSchedule = false
	}

	// Cloud platforms (Railway, Fly, Render, …) inject the listen port via the
	// PORT env var and run the start command without a shell, so a "-addr :$PORT"
	// never expands. Honor PORT directly unless -addr was set explicitly.
	if port := os.Getenv("PORT"); port != "" && *addr == ":8080" {
		*addr = ":" + port
	}

	logger := logx.New(os.Stderr)
	dealsLog := logger.WithPrefix("DEALS")
	trackLog := logger.WithPrefix("TRACKING")

	oneShot := *captureOnce || *captureSealedOnce || *pkmCaptureOnce || *pkmCaptureSealedOnce ||
		*rftCaptureOnce || *rftCaptureSealedOnce || *lorCaptureOnce || *lorCaptureSealedOnce ||
		*gndCaptureOnce || *gndCaptureSealedOnce
	// The unified scheduler owns all refresh timing, so the per-store background
	// refresh goroutines are suppressed alongside one-shot runs (Load still happens).
	// Serve-only never scrapes, so it also only loads.
	loadOnly := oneShot || *schedule || *serveOnly

	var solver *flaresolverr.Client
	if *flaresolverrURL != "" {
		solver = flaresolverr.New(*flaresolverrURL)
		defer solver.Close(context.Background())
		logger.Printf("FlareSolverr enabled at %s for Cloudflare-challenged Liga hosts", *flaresolverrURL)
	}

	tradeStore := trades.NewStore(*tradesPath, logger)
	rftTrades := trades.NewStore(*rftTradesPath, logger)
	lorTrades := trades.NewStore(*lorTradesPath, logger)
	gndTrades := trades.NewStore(*gndTradesPath, logger)

	st := store.New(*cachePath, dealsLog, pipeline.Options{
		Game:         game.OnePiece(),
		Sets:         parseSets(*setsFlag),
		FXOverride:   *fxOverride,
		Concurrency:  *concurrency,
		Timeout:      *timeout,
		VerifyFloor:  *verifyFloor,
		LivePrices:   *livePrices,
		MyPCards:     *myPCards,
		FlareSolverr: solver,
		HeldKeys:     heldCardKeys(tradeStore, compare.MatcherFor(game.OnePiece())),
	})
	scheduleDealsRefresh(st, dealsLog, *refreshInterval, 0, loadOnly)

	var rftDealsStore *store.Store
	if *rftDeals {
		rftLog := logger.WithPrefix("RFT DEALS")
		rftDealsStore = store.New(*rftCache, rftLog, pipeline.Options{
			Game:         game.Riftbound(),
			FXOverride:   *fxOverride,
			Concurrency:  *concurrency,
			Timeout:      *timeout,
			VerifyFloor:  *verifyFloor,
			LivePrices:   *livePrices,
			MyPCards:     *myPCards,
			FlareSolverr: solver,
			HeldKeys:     heldCardKeys(rftTrades, compare.MatcherFor(game.Riftbound())),
		})
		scheduleDealsRefresh(rftDealsStore, rftLog, *refreshInterval, 10*time.Minute, loadOnly)
	}

	var lorDealsStore *store.Store
	if *lorDeals {
		lorLog := logger.WithPrefix("LOR DEALS")
		lorDealsStore = store.New(*lorCache, lorLog, pipeline.Options{
			Game:         game.Lorcana(),
			FXOverride:   *fxOverride,
			Concurrency:  *concurrency,
			Timeout:      *timeout,
			VerifyFloor:  *verifyFloor,
			LivePrices:   *livePrices,
			FlareSolverr: solver,
			HeldKeys:     heldCardKeys(lorTrades, compare.MatcherFor(game.Lorcana())),
		})
		scheduleDealsRefresh(lorDealsStore, lorLog, *refreshInterval, 20*time.Minute, loadOnly)
	}

	var gndDealsStore *store.Store
	if *gndDeals {
		gndLog := logger.WithPrefix("GND DEALS")
		gndDealsStore = store.New(*gndCache, gndLog, pipeline.Options{
			Game:         game.Gundam(),
			FXOverride:   *fxOverride,
			Concurrency:  *concurrency,
			Timeout:      *timeout,
			VerifyFloor:  *verifyFloor,
			LivePrices:   *livePrices,
			FlareSolverr: solver,
			HeldKeys:     heldCardKeys(gndTrades, compare.MatcherFor(game.Gundam())),
		})
		scheduleDealsRefresh(gndDealsStore, gndLog, *refreshInterval, 30*time.Minute, loadOnly)
	}

	tz, err := time.LoadLocation(*trackTZ)
	if err != nil {
		logger.Printf("warning: track-tz %q: %v; falling back to UTC", *trackTZ, err)
		tz = time.UTC
	}

	opStack, opCapturer, err := buildStack(stackParams{
		game:        game.OnePiece(),
		trackDir:    *trackDir,
		tradeStore:  tradeStore,
		quotesPath:  *quotesPath,
		trackSets:   parseSets(*trackSet),
		track:       *track,
		trackSealed: *trackSealed,
		timeout:     *timeout,
		concurrency: *concurrency,
		tz:          tz,
		interval:    *trackInterval,
		minPrice:    *trackMinPrice,
		fxProvider:  func() float64 { return st.Snapshot().FXRate },
		solver:      solver,
		log:         trackLog,
	})
	if err != nil {
		return err
	}
	opStack.Deals = st

	if *backfillImages {
		return backfillCardImages(context.Background(), trackLog, opStack.Track, opStack.Cardimg)
	}

	pkmLog := logger.WithPrefix("POKEMON")
	pkmStack, pkmCapturer, err := buildStack(stackParams{
		game:        game.Pokemon(),
		trackDir:    *pkmTrackDir,
		tradesPath:  *pkmTradesPath,
		quotesPath:  *pkmQuotesPath,
		trackSets:   parseSets(*pkmTrackSet),
		track:       *pkmTrack,
		trackSealed: *pkmTrackSealed,
		timeout:     *timeout,
		concurrency: *concurrency,
		tz:          tz,
		interval:    *trackInterval,
		minPrice:    *trackMinPrice,
		fxProvider:  nil,
		solver:      solver,
		log:         pkmLog,
	})
	if err != nil {
		return err
	}

	rftStack, rftCapturer, err := buildStack(stackParams{
		game:        game.Riftbound(),
		trackDir:    *rftTrackDir,
		tradeStore:  rftTrades,
		quotesPath:  *rftQuotesPath,
		trackSets:   parseSets(*rftTrackSet),
		track:       *rftTrack,
		trackSealed: *rftTrackSealed,
		timeout:     *timeout,
		concurrency: *concurrency,
		tz:          tz,
		interval:    *trackInterval,
		minPrice:    *trackMinPrice,
		fxProvider:  dealsFX(rftDealsStore),
		solver:      solver,
		log:         logger.WithPrefix("RIFTBOUND"),
	})
	if err != nil {
		return err
	}
	rftStack.Deals = rftDealsStore

	lorStack, lorCapturer, err := buildStack(stackParams{
		game:        game.Lorcana(),
		trackDir:    *lorTrackDir,
		tradeStore:  lorTrades,
		quotesPath:  *lorQuotesPath,
		trackSets:   parseSets(*lorTrackSet),
		track:       *lorTrack,
		trackSealed: *lorTrackSealed,
		timeout:     *timeout,
		concurrency: *concurrency,
		tz:          tz,
		interval:    *trackInterval,
		minPrice:    *trackMinPrice,
		fxProvider:  dealsFX(lorDealsStore),
		solver:      solver,
		log:         logger.WithPrefix("LORCANA"),
	})
	if err != nil {
		return err
	}
	lorStack.Deals = lorDealsStore

	gndStack, gndCapturer, err := buildStack(stackParams{
		game:        game.Gundam(),
		trackDir:    *gndTrackDir,
		tradeStore:  gndTrades,
		quotesPath:  *gndQuotesPath,
		trackSets:   parseSets(*gndTrackSet),
		track:       *gndTrack,
		trackSealed: *gndTrackSealed,
		timeout:     *timeout,
		concurrency: *concurrency,
		tz:          tz,
		interval:    *trackInterval,
		minPrice:    *trackMinPrice,
		fxProvider:  dealsFX(gndDealsStore),
		solver:      solver,
		log:         logger.WithPrefix("GUNDAM"),
	})
	if err != nil {
		return err
	}
	gndStack.Deals = gndDealsStore

	if *captureSealedOnce || *captureOnce {
		if opCapturer == nil {
			return fmt.Errorf("capture-once requires -track enabled")
		}
		if *captureSealedOnce {
			return opCapturer.CaptureSealed(context.Background(), time.Now())
		}
		return opCapturer.Capture(context.Background(), time.Now())
	}
	if *pkmCaptureSealedOnce || *pkmCaptureOnce {
		if pkmCapturer == nil {
			return fmt.Errorf("pkm-capture-once requires -pkm-track enabled")
		}
		if *pkmCaptureSealedOnce {
			return pkmCapturer.CaptureSealed(context.Background(), time.Now())
		}
		return pkmCapturer.Capture(context.Background(), time.Now())
	}
	if *rftCaptureSealedOnce || *rftCaptureOnce {
		if rftCapturer == nil {
			return fmt.Errorf("rft-capture-once requires -rft-track enabled")
		}
		if *rftCaptureSealedOnce {
			return rftCapturer.CaptureSealed(context.Background(), time.Now())
		}
		return rftCapturer.Capture(context.Background(), time.Now())
	}
	if *lorCaptureSealedOnce || *lorCaptureOnce {
		if lorCapturer == nil {
			return fmt.Errorf("lor-capture-once requires -lor-track enabled")
		}
		if *lorCaptureSealedOnce {
			return lorCapturer.CaptureSealed(context.Background(), time.Now())
		}
		return lorCapturer.Capture(context.Background(), time.Now())
	}
	if *gndCaptureSealedOnce || *gndCaptureOnce {
		if gndCapturer == nil {
			return fmt.Errorf("gnd-capture-once requires -gnd-track enabled")
		}
		if *gndCaptureSealedOnce {
			return gndCapturer.CaptureSealed(context.Background(), time.Now())
		}
		return gndCapturer.Capture(context.Background(), time.Now())
	}

	if *schedule {
		if *trackSchedule {
			trackLog.Printf("-track-schedule is ignored while -schedule is on (the unified scheduler owns capture timing)")
		}
		jobs := make([]schedulerJob, 0, 9)
		jobs = append(jobs, dealsJob("onepiece deals", st, *scheduleInterval))
		jobs = append(jobs, captureJob("onepiece capture", opCapturer)...)
		if rftDealsStore != nil {
			jobs = append(jobs, dealsJob("riftbound deals", rftDealsStore, *scheduleInterval))
		}
		jobs = append(jobs, captureJob("riftbound capture", rftCapturer)...)
		if lorDealsStore != nil {
			jobs = append(jobs, dealsJob("lorcana deals", lorDealsStore, *scheduleInterval))
		}
		jobs = append(jobs, captureJob("lorcana capture", lorCapturer)...)
		if gndDealsStore != nil {
			jobs = append(jobs, dealsJob("gundam deals", gndDealsStore, *scheduleInterval))
		}
		jobs = append(jobs, captureJob("gundam capture", gndCapturer)...)
		jobs = append(jobs, captureJob("pokemon capture", pkmCapturer)...)
		go runUnifiedScheduler(jobs, *scheduleInterval, *scheduleGap, logger.WithPrefix("SCHED"))
	} else if *trackSchedule {
		for _, c := range []*tracking.Capturer{opCapturer, pkmCapturer, rftCapturer, lorCapturer, gndCapturer} {
			if c != nil {
				startTrackingScheduler(c, *trackInterval)
			}
		}
	} else {
		trackLog.Printf("scheduler off; trigger via POST /api/tracking/capture(?game=) or -capture-once (enable with -schedule)")
	}

	games := map[string]*api.GameStack{
		opStack.Game.ID:  opStack,
		pkmStack.Game.ID: pkmStack,
		rftStack.Game.ID: rftStack,
		lorStack.Game.ID: lorStack,
		gndStack.Game.ID: gndStack,
	}
	srv := api.New(*webDir, games, opStack.Game.ID, *serveOnly, *adminToken)
	logger.Printf("listening on %s (web dir %q)", *addr, *webDir)
	return http.ListenAndServe(*addr, srv.Handler())
}

// scheduleDealsRefresh loads a deals store's cached snapshot and starts its
// background refresh cycle: an immediate refresh when the cache is missing or
// stale, then a ticker at interval. The stagger delays a game's first refresh
// so the per-game full scans don't hit TCGplayer and the Liga CDN at once.
// One-shot capture runs skip all background refreshing.
func scheduleDealsRefresh(st *store.Store, logger *logx.Logger, interval, stagger time.Duration, oneShot bool) {
	if err := st.Load(); err != nil {
		logger.Printf("warning: %v", err)
	}
	if oneShot {
		return
	}
	ready, _, snap := st.Status()
	stale := interval > 0 && !snap.UpdatedAt.IsZero() && time.Since(snap.UpdatedAt) > interval
	if !ready || stale {
		if stale {
			logger.Printf("cached snapshot is %v old; refreshing in background", time.Since(snap.UpdatedAt).Round(time.Hour))
		} else {
			logger.Printf("no cached snapshot; starting initial scan in background")
		}
		go func() {
			time.Sleep(stagger)
			st.Refresh(context.Background())
		}()
	}
	if interval > 0 {
		go func() {
			time.Sleep(stagger)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				st.Refresh(context.Background())
			}
		}()
	}
}

// dealsFX exposes a deals store's live FX rate to the tracking capturer; nil
// when the game has no deals pipeline.
func dealsFX(st *store.Store) func() float64 {
	if st == nil {
		return nil
	}
	return func() float64 { return st.Snapshot().FXRate }
}

type schedulerJob struct {
	name string
	run  func(context.Context) error
}

// dealsJob refreshes a deals snapshot unless it is already younger than maxAge,
// so a server restart doesn't rescan data that is still fresh.
func dealsJob(name string, st *store.Store, maxAge time.Duration) schedulerJob {
	return schedulerJob{name: name, run: func(ctx context.Context) error {
		_, _, snap := st.Status()
		if age := time.Since(snap.UpdatedAt); !snap.UpdatedAt.IsZero() && age < maxAge {
			return fmt.Errorf("snapshot is %v old, still fresh: skipping", age.Round(time.Minute))
		}
		return st.Refresh(ctx)
	}}
}

// captureJob wraps a game's tracking capture; a nil capturer (tracking
// disabled) contributes no job.
func captureJob(name string, c *tracking.Capturer) []schedulerJob {
	if c == nil {
		return nil
	}
	return []schedulerJob{{name: name, run: func(ctx context.Context) error {
		return c.Capture(ctx, time.Now())
	}}}
}

// runUnifiedScheduler serializes every Liga-heavy job into one lane: exactly
// one scan runs at a time so it gets the full polite request budget instead of
// competing with the other games' scans for Liga's shared rate limit. Each
// cycle runs all jobs in order with a gap between them, then sleeps until
// cycleStart+interval (an overrunning cycle starts the next immediately).
func runUnifiedScheduler(jobs []schedulerJob, interval, gap time.Duration, log *logx.Logger) {
	log.Printf("unified scheduler: %d jobs per cycle, cadence %v, gap %v", len(jobs), interval, gap)
	for cycle := 1; ; cycle++ {
		cycleStart := time.Now()
		for i, j := range jobs {
			if i > 0 && gap > 0 {
				time.Sleep(gap)
			}
			jobStart := time.Now()
			log.Printf("cycle %d [%d/%d] %s: starting", cycle, i+1, len(jobs), j.name)
			if err := j.run(context.Background()); err != nil {
				log.Printf("cycle %d [%d/%d] %s: %v", cycle, i+1, len(jobs), j.name, err)
				continue
			}
			log.Printf("cycle %d [%d/%d] %s: done in %v", cycle, i+1, len(jobs), j.name, time.Since(jobStart).Round(time.Second))
		}
		elapsed := time.Since(cycleStart)
		log.Printf("cycle %d complete in %v", cycle, elapsed.Round(time.Second))
		if sleep := interval - elapsed; sleep > 0 {
			time.Sleep(sleep)
		}
	}
}

type stackParams struct {
	game        game.Game
	trackDir    string
	tradesPath  string
	tradeStore  *trades.Store
	quotesPath  string
	trackSets   []string
	track       bool
	trackSealed bool
	timeout     time.Duration
	concurrency int
	tz          *time.Location
	interval    time.Duration
	minPrice    float64
	fxProvider  func() float64
	solver      *flaresolverr.Client
	log         *logx.Logger
}

// ligaFetcher returns the transport a liga.Client should use: FlareSolverr-routed
// when a solver is configured and the game's Liga host is behind Cloudflare's
// Managed Challenge, otherwise the direct client unchanged.
func ligaFetcher(direct *httpx.Client, solver *flaresolverr.Client, g game.Game) liga.Fetcher {
	if solver == nil || !g.Challenged {
		return direct
	}
	return flaresolverr.NewRouter(direct, solver, g.LigaHosts)
}

// buildStack assembles one game's BR tracking state: its snapshot store, image
// cache, trade ledger, and (when tracking is enabled) a Liga capturer wired to
// that game's host, set list, and floor languages. It returns the API stack plus
// the capturer (nil when tracking is off) for scheduling / one-shot captures.
func buildStack(p stackParams) (*api.GameStack, *tracking.Capturer, error) {
	trackStore := tracking.NewStore(p.trackDir, p.log)

	imgHTTP, err := httpx.New(p.timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("%s image http client: %w", p.game.ID, err)
	}
	imgStore := cardimg.NewStore(filepath.Join(p.trackDir, "images.json"), ligaFetcher(imgHTTP, p.solver, p.game), p.log, p.game.ImageURLRe, p.game.UniqueCardNumbers)

	tradeStore := p.tradeStore
	if tradeStore == nil {
		tradeStore = trades.NewStore(p.tradesPath, p.log)
	}
	quoteStore := quotes.NewStore(p.quotesPath, p.log)

	var capturer *tracking.Capturer
	if p.track {
		httpClient, err := httpx.NewThrottled(p.timeout)
		if err != nil {
			return nil, nil, fmt.Errorf("%s tracking http client: %w", p.game.ID, err)
		}
		fetcher := ligaFetcher(httpClient, p.solver, p.game)
		ligaClient := liga.New(fetcher, p.log, p.concurrency, p.trackSets, p.game)
		capturer = tracking.NewCapturer(p.log, ligaClient, trackStore, p.trackSets, p.minPrice, p.tz, p.interval, p.trackSealed, p.game.FloorLangs, p.fxProvider, imgStore)
	}

	return &api.GameStack{
		Game:     p.game,
		Track:    trackStore,
		Capturer: capturer,
		Cardimg:  imgStore,
		Trades:   tradeStore,
		Quotes:   quoteStore,
	}, capturer, nil
}

func backfillCardImages(ctx context.Context, logger *logx.Logger, ts *tracking.Store, is *cardimg.Store) error {
	sets, err := ts.ListSets()
	if err != nil {
		return fmt.Errorf("backfill list sets: %w", err)
	}
	var items []cardimg.Item
	for _, set := range sets {
		day, ok, err := ts.LatestDay(set)
		if err != nil || !ok {
			continue
		}
		for _, c := range day.Cards {
			items = append(items, cardimg.Item{Set: set, Number: c.Number, PageURL: c.URL})
		}
	}
	logger.Printf("cardimg: backfilling %d cards from %d sets", len(items), len(sets))
	resolved, total := is.Warm(ctx, items, 2, 700*time.Millisecond)
	logger.Printf("cardimg: backfill done: %d/%d resolved", resolved, total)
	return nil
}

func startTrackingScheduler(capturer *tracking.Capturer, interval time.Duration) {
	check := interval / 8
	if check < time.Minute {
		check = time.Minute
	}
	if check > 30*time.Minute {
		check = 30 * time.Minute
	}
	go func() {
		capturer.Capture(context.Background(), time.Now())
		ticker := time.NewTicker(check)
		defer ticker.Stop()
		for range ticker.C {
			capturer.Capture(context.Background(), time.Now())
		}
	}()
}

// heldCardKeys returns a live snapshot of the portfolio's held-card match keys so
// the deals pipeline live-prices cards you own on TCGplayer, not just candidate
// deals. Sold trades are skipped since they no longer need a current price.
func heldCardKeys(ledger *trades.Store, m compare.Matcher) func() []string {
	return func() []string {
		all, err := ledger.List()
		if err != nil {
			return nil
		}
		keys := make([]string, 0, len(all))
		for _, t := range all {
			if t.Status == "sold" || t.Kind == "sealed" {
				continue
			}
			keys = append(keys, m.LookupKey(t.Number, t.Name, t.Set))
		}
		return keys
	}
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
