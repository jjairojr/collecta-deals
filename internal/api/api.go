package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	"opdeals/internal/cardimg"
	"opdeals/internal/compare"
	"opdeals/internal/game"
	"opdeals/internal/model"
	"opdeals/internal/quotes"
	"opdeals/internal/store"
	"opdeals/internal/tracking"
	"opdeals/internal/trades"
)

// GameStack bundles the per-game state. Every /api route selects a stack by
// the request's ?game= param, defaulting to One Piece so existing clients are
// unaffected. Deals is the game's US-arbitrage snapshot store; nil means the
// game is BR-tracking-only.
type GameStack struct {
	Game     game.Game
	Track    *tracking.Store
	Capturer *tracking.Capturer
	Cardimg  *cardimg.Store
	Trades   *trades.Store
	Quotes   *quotes.Store
	Deals    *store.Store
}

type Server struct {
	webDir      string
	games       map[string]*GameStack
	defaultGame string
	readOnly    bool
	adminToken  string
}

// New builds the API server. readOnly gates every scrape-triggering endpoint
// (refresh/capture) so a serve-only prod instance never scrapes; adminToken (when
// non-empty) enables POST /api/admin/reload to reload deals snapshots from disk
// after a local scrape pushes fresh files onto the volume.
func New(webDir string, games map[string]*GameStack, defaultGame string, readOnly bool, adminToken string) *Server {
	return &Server{webDir: webDir, games: games, defaultGame: defaultGame, readOnly: readOnly, adminToken: adminToken}
}

// stackFor resolves the game stack for a request's query, defaulting to the
// default game when ?game= is missing or unknown.
func (s *Server) stackFor(q map[string][]string) *GameStack {
	id := stringParam(q, "game", "")
	if gs, ok := s.games[id]; ok {
		return gs
	}
	return s.games[s.defaultGame]
}

type dealsResponse struct {
	Ready      bool         `json:"ready"`
	Refreshing bool         `json:"refreshing"`
	UpdatedAt  time.Time    `json:"updatedAt"`
	FXRate     float64      `json:"fxRate"`
	Count      int          `json:"count"`
	Deals      []model.Deal `json:"deals"`
	Sets       []string     `json:"sets"`
}

// listingSets returns the distinct set codes present in a deals snapshot's
// Brazil listings, sorted, so the UI can offer a set filter.
func listingSets(listings []model.BrazilListing) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, l := range listings {
		if l.SetCode == "" {
			continue
		}
		if _, ok := seen[l.SetCode]; ok {
			continue
		}
		seen[l.SetCode] = struct{}{}
		out = append(out, l.SetCode)
	}
	sort.Strings(out)
	return out
}

type statusResponse struct {
	Ready      bool      `json:"ready"`
	Refreshing bool      `json:"refreshing"`
	UpdatedAt  time.Time `json:"updatedAt"`
	FXRate     float64   `json:"fxRate"`
	Listings   int       `json:"listings"`
	Prices     int       `json:"prices"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/deals", s.handleDeals)
	mux.HandleFunc("GET /api/search", s.handleSearch)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("POST /api/refresh", s.handleRefresh)
	mux.HandleFunc("GET /api/tracking/trends", s.handleTrends)
	mux.HandleFunc("GET /api/tracking/leaderboard", s.handleLeaderboard)
	mux.HandleFunc("GET /api/tracking/sold", s.handleTopSold)
	mux.HandleFunc("GET /api/tracking/sold-by-snapshot", s.handleSalesBySnapshot)
	mux.HandleFunc("GET /api/tracking/card", s.handleCardHistory)
	mux.HandleFunc("GET /api/tracking/sets", s.handleTrackingSets)
	mux.HandleFunc("GET /api/tracking/dates", s.handleTrackingDates)
	mux.HandleFunc("GET /api/tracking/latest", s.handleTrackingLatest)
	mux.HandleFunc("GET /api/tracking/snapshots", s.handleRecentSnapshots)
	mux.HandleFunc("GET /api/tracking/inventory", s.handleInventory)
	mux.HandleFunc("GET /api/tracking/buyout", s.handleBuyout)
	mux.HandleFunc("GET /api/tracking/cards", s.handleTrackingCards)
	mux.HandleFunc("POST /api/tracking/capture", s.handleCapture)
	mux.HandleFunc("POST /api/tracking/capture-sealed", s.handleCaptureSealed)
	mux.HandleFunc("GET /api/card-image", s.handleCardImage)
	mux.HandleFunc("POST /api/export/image", s.handleExportImage)
	mux.HandleFunc("GET /api/games", s.handleGames)
	mux.HandleFunc("GET /api/trades", s.handleTradesList)
	mux.HandleFunc("POST /api/trades", s.handleTradesCreate)
	mux.HandleFunc("PUT /api/trades/{id}", s.handleTradesUpdate)
	mux.HandleFunc("POST /api/trades/{id}/sell", s.handleTradesSell)
	mux.HandleFunc("DELETE /api/trades/{id}", s.handleTradesDelete)
	mux.HandleFunc("GET /api/trades/quote", s.handleTradesQuote)
	mux.HandleFunc("GET /api/portfolio/all", s.handlePortfolioAll)
	mux.HandleFunc("GET /api/quotes", s.handleQuotesList)
	mux.HandleFunc("POST /api/quotes", s.handleQuotesCreate)
	mux.HandleFunc("PUT /api/quotes/{id}", s.handleQuotesUpdate)
	mux.HandleFunc("DELETE /api/quotes/{id}", s.handleQuotesDelete)
	if s.adminToken != "" {
		mux.HandleFunc("POST /api/admin/reload", s.handleAdminReload)
	}
	if s.webDir != "" {
		mux.Handle("/", s.spaHandler())
	}
	return cors(mux)
}

type gameInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	HasDeals bool   `json:"hasDeals"`
	HasMyP   bool   `json:"hasMyP"`
}

func (s *Server) gameInfoFor(gs *GameStack) gameInfo {
	return gameInfo{
		ID:       gs.Game.ID,
		Name:     gs.Game.Name,
		HasDeals: gs.Game.HasDeals() && gs.Deals != nil,
		HasMyP:   gs.Game.MyP != nil && gs.Deals != nil,
	}
}

// handleGames lists the configured games so the UI can render a switcher. The
// default game is listed first.
func (s *Server) handleGames(w http.ResponseWriter, r *http.Request) {
	resp := struct {
		Default string     `json:"default"`
		Games   []gameInfo `json:"games"`
	}{Default: s.defaultGame, Games: []gameInfo{}}
	for _, id := range s.orderedGameIDs() {
		resp.Games = append(resp.Games, s.gameInfoFor(s.games[id]))
	}
	writeJSON(w, resp)
}

func (s *Server) handleDeals(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	if gs.Deals == nil {
		writeJSON(w, dealsResponse{Deals: []model.Deal{}, Sets: []string{}})
		return
	}
	ready, refreshing, snap := gs.Deals.Status()
	deals := compare.Deals(snap.Listings, snap.Prices, compare.Options{
		FXRate:         snap.FXRate,
		MinMargin:      floatParam(q, "minMargin", 20),
		MinPrice:       floatParam(q, "minPrice", 100),
		UsePrice:       stringParam(q, "usPrice", "low"),
		SortBy:         stringParam(q, "sort", "margin"),
		Set:            stringParam(q, "set", ""),
		Source:         stringParam(q, "source", ""),
		Limit:          intParam(q, "limit", 100),
		RequireInStock: boolParam(q, "requireInStock", true),
		SPOnly:         boolParam(q, "spOnly", false),
		Matcher:        compare.MatcherFor(gs.Game),
	})
	writeJSON(w, dealsResponse{
		Ready:      ready,
		Refreshing: refreshing,
		UpdatedAt:  snap.UpdatedAt,
		FXRate:     snap.FXRate,
		Count:      len(deals),
		Deals:      deals,
		Sets:       listingSets(snap.Listings),
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	if gs.Deals == nil {
		writeJSON(w, dealsResponse{Deals: []model.Deal{}, Sets: []string{}})
		return
	}
	ready, refreshing, snap := gs.Deals.Status()
	query := q.Get("q")
	deals := []model.Deal{}
	if query != "" {
		deals = compare.Deals(snap.Listings, snap.Prices, compare.Options{
			FXRate:         snap.FXRate,
			MinMargin:      -1e9,
			MinPrice:       0,
			UsePrice:       stringParam(q, "usPrice", "low"),
			SortBy:         stringParam(q, "sort", "margin"),
			Query:          query,
			Set:            stringParam(q, "set", ""),
			Source:         stringParam(q, "source", ""),
			Limit:          intParam(q, "limit", 100),
			RequireInStock: false,
			Matcher:        compare.MatcherFor(gs.Game),
		})
	}
	writeJSON(w, dealsResponse{
		Ready:      ready,
		Refreshing: refreshing,
		UpdatedAt:  snap.UpdatedAt,
		FXRate:     snap.FXRate,
		Count:      len(deals),
		Deals:      deals,
		Sets:       listingSets(snap.Listings),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Deals == nil {
		writeJSON(w, statusResponse{})
		return
	}
	ready, refreshing, snap := gs.Deals.Status()
	writeJSON(w, statusResponse{
		Ready:      ready,
		Refreshing: refreshing,
		UpdatedAt:  snap.UpdatedAt,
		FXRate:     snap.FXRate,
		Listings:   len(snap.Listings),
		Prices:     len(snap.Prices),
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if s.readOnly {
		writeJSON(w, map[string]bool{"started": false})
		return
	}
	gs := s.stackFor(r.URL.Query())
	if gs.Deals == nil {
		writeJSON(w, map[string]bool{"started": false})
		return
	}
	go gs.Deals.Refresh(context.Background())
	writeJSON(w, map[string]bool{"started": true})
}

// handleAdminReload re-reads every game's deals snapshot from disk so a serve-only
// instance picks up files a local scrape just pushed onto the volume, without a
// restart. Tracking data needs nothing here — it is read from disk on each request.
func (s *Server) handleAdminReload(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Admin-Token") != s.adminToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	reloaded := 0
	for _, gs := range s.games {
		if gs.Deals != nil {
			if err := gs.Deals.Load(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			reloaded++
		}
		if gs.Cardimg != nil {
			if err := gs.Cardimg.Reload(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	writeJSON(w, map[string]int{"reloaded": reloaded})
}

type trendsResponse struct {
	Set      string               `json:"set"`
	Range    string               `json:"range"`
	Date     string               `json:"date"`
	PrevDate string               `json:"prevDate"`
	Count    int                  `json:"count"`
	Trends   []tracking.CardTrend `json:"trends"`
}

type leaderboardResponse struct {
	Set    string               `json:"set"`
	From   string               `json:"from"`
	To     string               `json:"to"`
	Sort   string               `json:"sort"`
	Stores []tracking.StoreStat `json:"stores"`
}

type cardHistoryResponse struct {
	Set    string                `json:"set"`
	Number string                `json:"number"`
	Points []tracking.PricePoint `json:"points"`
}

type datesResponse struct {
	Set   string   `json:"set"`
	Dates []string `json:"dates"`
}

func (s *Server) trackSet(gs *GameStack, q map[string][]string) string {
	def := gs.Game.DefaultSet
	if gs.Capturer != nil && gs.Capturer.Set() != "" {
		def = gs.Capturer.Set()
	}
	return stringParam(q, "set", def)
}

func (s *Server) trackSets(gs *GameStack) []string {
	if gs.Capturer != nil && len(gs.Capturer.Sets()) > 0 {
		return gs.Capturer.Sets()
	}
	if gs.Track != nil {
		if sets, err := gs.Track.ListSets(); err == nil {
			return withoutSealed(sets)
		}
	}
	return nil
}

func withoutSealed(sets []string) []string {
	out := make([]string, 0, len(sets))
	for _, s := range sets {
		if s == "SEALED" {
			continue
		}
		out = append(out, s)
	}
	return out
}

var trendWindows = map[string]time.Duration{
	"daily":   24 * time.Hour,
	"weekly":  7 * 24 * time.Hour,
	"monthly": 30 * 24 * time.Hour,
}

// trendRange normalizes the movers comparison window, defaulting to daily.
func trendRange(q map[string][]string) string {
	r := stringParam(q, "range", "daily")
	if _, ok := trendWindows[r]; !ok {
		return "daily"
	}
	return r
}

// snapshotTime parses a storage date key ("2006-01-02" or "2006-01-02T15") into a
// comparable instant. The zone is irrelevant since it is only used for durations
// between two keys parsed the same way.
func snapshotTime(key string) (time.Time, bool) {
	if t, err := time.Parse("2006-01-02T15", key); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", key); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// baselineDate picks the snapshot to compare the latest against for a movers
// window: the newest snapshot at or before (latest - window). When history is
// shorter than the window it falls back to the earliest snapshot so the range
// still reports the change over whatever span exists. Returns "" when there is
// nothing older than latest to compare with.
func baselineDate(dates []string, latest string, window time.Duration) string {
	lt, ok := snapshotTime(latest)
	if !ok || len(dates) < 2 {
		return ""
	}
	cutoff := lt.Add(-window)
	pick := ""
	for _, d := range dates {
		if d == latest {
			continue
		}
		t, ok := snapshotTime(d)
		if !ok || t.After(cutoff) {
			continue
		}
		pick = d
	}
	if pick == "" {
		pick = dates[0]
	}
	return pick
}

func (s *Server) handleTrends(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	set := s.trackSet(gs, q)
	rng := trendRange(q)
	window := trendWindows[rng]
	resp := trendsResponse{Set: set, Range: rng, Trends: []tracking.CardTrend{}}
	if gs.Track == nil {
		writeJSON(w, resp)
		return
	}
	if set == "ALL" {
		merged := []tracking.CardTrend{}
		latest, prev := "", ""
		for _, sc := range s.trackSets(gs) {
			dates, err := gs.Track.ListDates(sc)
			if err != nil || len(dates) < 2 {
				continue
			}
			latestDate := dates[len(dates)-1]
			today, _, err := gs.Track.LoadDay(sc, latestDate)
			if err != nil {
				continue
			}
			pd := baselineDate(dates, latestDate, window)
			if pd == "" {
				continue
			}
			p, _, err := gs.Track.LoadDay(sc, pd)
			if err != nil {
				continue
			}
			for _, t := range tracking.PriceTrends(today, p) {
				if t.PrevBRL > 0 && t.DeltaPct != 0 {
					t.Set = sc
					merged = append(merged, t)
				}
			}
			if today.Date > latest {
				latest = today.Date
			}
			if p.Date > prev {
				prev = p.Date
			}
		}
		sort.SliceStable(merged, func(i, j int) bool {
			return absFloat(merged[i].DeltaPct) > absFloat(merged[j].DeltaPct)
		})
		if len(merged) > 120 {
			merged = merged[:120]
		}
		resp.Date, resp.PrevDate = latest, prev
		resp.Trends, resp.Count = merged, len(merged)
		writeJSON(w, resp)
		return
	}
	dates, err := gs.Track.ListDates(set)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(dates) == 0 {
		writeJSON(w, resp)
		return
	}
	today, _, err := gs.Track.LoadDay(set, dates[len(dates)-1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var prev tracking.DaySnapshot
	if pd := baselineDate(dates, dates[len(dates)-1], window); pd != "" {
		prev, _, _ = gs.Track.LoadDay(set, pd)
		resp.PrevDate = prev.Date
	}
	resp.Date = today.Date
	resp.Trends = tracking.PriceTrends(today, prev)
	resp.Count = len(resp.Trends)
	writeJSON(w, resp)
}

func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	set := s.trackSet(gs, q)
	sortBy := stringParam(q, "sort", "units")
	from := stringParam(q, "from", "")
	to := stringParam(q, "to", "")
	resp := leaderboardResponse{Set: set, From: from, To: to, Sort: sortBy, Stores: []tracking.StoreStat{}}
	if gs.Track == nil {
		writeJSON(w, resp)
		return
	}
	days, err := gs.Track.LoadRange(set, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stats := tracking.Leaderboard(days)
	tracking.SortLeaderboard(stats, sortBy)
	resp.Stores = stats
	writeJSON(w, resp)
}

type soldResponse struct {
	Set   string              `json:"set"`
	From  string              `json:"from"`
	To    string              `json:"to"`
	Cards []tracking.CardSale `json:"cards"`
}

func (s *Server) handleTopSold(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	set := s.trackSet(gs, q)
	from := stringParam(q, "from", "")
	to := stringParam(q, "to", "")
	top := intParam(q, "top", 24)
	resp := soldResponse{Set: set, From: from, To: to, Cards: []tracking.CardSale{}}
	if gs.Track == nil {
		writeJSON(w, resp)
		return
	}
	var cards []tracking.CardSale
	if set == "ALL" {
		for _, sc := range s.trackSets(gs) {
			days, err := gs.Track.LoadRange(sc, "", "")
			if err != nil || len(days) < 2 {
				continue
			}
			cs := tracking.TopSoldCards(days)
			for i := range cs {
				cs[i].Set = sc
			}
			cards = append(cards, cs...)
		}
	} else {
		days, err := gs.Track.LoadRange(set, from, to)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cards = tracking.TopSoldCards(days)
	}
	tracking.SortCardSales(cards)
	if top > 0 && len(cards) > top {
		cards = cards[:top]
	}
	resp.Cards = cards
	writeJSON(w, resp)
}

type snapshotSalesResponse struct {
	Set       string                   `json:"set"`
	From      string                   `json:"from"`
	To        string                   `json:"to"`
	Snapshots []tracking.SnapshotSales `json:"snapshots"`
}

func (s *Server) handleSalesBySnapshot(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	set := s.trackSet(gs, q)
	from := stringParam(q, "from", "")
	to := stringParam(q, "to", "")
	resp := snapshotSalesResponse{Set: set, From: from, To: to, Snapshots: []tracking.SnapshotSales{}}
	if gs.Track == nil {
		writeJSON(w, resp)
		return
	}
	if set == "ALL" {
		var perSet [][]tracking.SnapshotSales
		for _, sc := range s.trackSets(gs) {
			days, err := gs.Track.LoadRange(sc, from, to)
			if err != nil || len(days) < 2 {
				continue
			}
			snaps := tracking.SalesBySnapshot(days)
			for i := range snaps {
				for j := range snaps[i].Cards {
					snaps[i].Cards[j].Set = sc
				}
			}
			perSet = append(perSet, snaps)
		}
		resp.Snapshots = tracking.PoolSnapshotSales(perSet)
		writeJSON(w, resp)
		return
	}
	days, err := gs.Track.LoadRange(set, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Snapshots = tracking.SalesBySnapshot(days)
	writeJSON(w, resp)
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func (s *Server) handleCardHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	set := s.trackSet(gs, q)
	number := model.NormalizeNumber(q.Get("number"))
	resp := cardHistoryResponse{Set: set, Number: number, Points: []tracking.PricePoint{}}
	if gs.Track == nil || number == "" {
		writeJSON(w, resp)
		return
	}
	days, err := gs.Track.LoadRange(set, "", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Points = tracking.CardHistory(days, number)
	writeJSON(w, resp)
}

func (s *Server) handleTrackingSets(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	resp := struct {
		Sets []string `json:"sets"`
	}{Sets: []string{}}
	if gs.Track != nil {
		if sets, err := gs.Track.ListSets(); err == nil && len(sets) > 0 {
			resp.Sets = sets
			writeJSON(w, resp)
			return
		}
	}
	if gs.Capturer != nil {
		resp.Sets = gs.Capturer.Sets()
	}
	writeJSON(w, resp)
}

func (s *Server) handleRecentSnapshots(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	limit := intParam(q, "limit", 12)
	resp := struct {
		Dates []string `json:"dates"`
	}{Dates: []string{}}
	if gs.Track == nil {
		writeJSON(w, resp)
		return
	}
	sets, err := gs.Track.ListSets()
	if err != nil {
		writeJSON(w, resp)
		return
	}
	seen := make(map[string]struct{})
	var dates []string
	for _, set := range sets {
		ds, err := gs.Track.ListDates(set)
		if err != nil {
			continue
		}
		for _, d := range ds {
			if _, ok := seen[d]; ok {
				continue
			}
			seen[d] = struct{}{}
			dates = append(dates, d)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	if limit > 0 && len(dates) > limit {
		dates = dates[:limit]
	}
	resp.Dates = dates
	writeJSON(w, resp)
}

func (s *Server) handleTrackingDates(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	set := s.trackSet(gs, q)
	resp := datesResponse{Set: set, Dates: []string{}}
	if gs.Track == nil {
		writeJSON(w, resp)
		return
	}
	dates, err := gs.Track.ListDates(set)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if dates != nil {
		resp.Dates = dates
	}
	writeJSON(w, resp)
}

type latestResponse struct {
	CapturedAt time.Time `json:"capturedAt"`
	Date       string    `json:"date"`
	Set        string    `json:"set"`
}

func (s *Server) handleTrackingLatest(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	resp := latestResponse{}
	if gs.Track == nil {
		writeJSON(w, resp)
		return
	}
	day, ok, err := gs.Track.LatestSnapshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ok {
		resp.CapturedAt = day.CapturedAt
		resp.Date = day.Date
		resp.Set = day.Set
	}
	writeJSON(w, resp)
}

type inventoryResponse struct {
	Set     string                    `json:"set"`
	Ready   bool                      `json:"ready"`
	Summary tracking.InventorySummary `json:"summary"`
}

func (s *Server) handleInventory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	set := s.trackSet(gs, q)
	resp := inventoryResponse{Set: set}
	if gs.Track == nil {
		writeJSON(w, resp)
		return
	}
	var day tracking.DaySnapshot
	var ok bool
	if set == "ALL" {
		// Pool every collection's latest snapshot into one inventory so the
		// "all collections" view has data from day one (store totals sum across
		// sets; the chase-card list spans all of them).
		merged := tracking.DaySnapshot{Set: "ALL"}
		for _, sc := range s.trackSets(gs) {
			d, found, err := gs.Track.LatestDay(sc)
			if err != nil || !found {
				continue
			}
			if d.Date > merged.Date {
				merged.Date = d.Date
			}
			merged.Cards = append(merged.Cards, d.Cards...)
		}
		day, ok = merged, len(merged.Cards) > 0
	} else {
		dates, err := gs.Track.ListDates(set)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(dates) == 0 {
			writeJSON(w, resp)
			return
		}
		day, ok, err = gs.Track.LoadDay(set, dates[len(dates)-1])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if ok {
		resp.Ready = true
		resp.Summary = tracking.Inventory(day, intParam(q, "stores", 25), intParam(q, "cards", 12), intParam(q, "holders", 8))
	}
	writeJSON(w, resp)
}

type buyoutResponse struct {
	Set        string                     `json:"set"`
	Date       string                     `json:"date"`
	Ready      bool                       `json:"ready"`
	Budget     float64                    `json:"budget"`
	MinFloor   float64                    `json:"minFloor"`
	Shipping   float64                    `json:"shipping"`
	FXRate     float64                    `json:"fxRate"`
	Sort       string                     `json:"sort"`
	Chars      string                     `json:"chars"`
	Mode       string                     `json:"mode"`
	MinGap     float64                    `json:"minGap"`
	Candidates []tracking.BuyoutCandidate `json:"candidates"`
}

// enrichUSD attaches each candidate's US resale price and TCGPlayer link from
// the deals snapshot, joining by the same match key the deals path uses.
// Candidates with no US match keep zero values.
func enrichUSD(cands []tracking.BuyoutCandidate, index map[string]model.USPrice, m compare.Matcher, fallbackSet string) {
	if len(index) == 0 {
		return
	}
	for i := range cands {
		set := cands[i].Set
		if set == "" {
			set = fallbackSet
		}
		us, ok := index[m.LookupKey(cands[i].Number, cands[i].Name, set)]
		if !ok {
			continue
		}
		cands[i].SellUSD = compare.EffectiveUSD(us)
		cands[i].TCGURL = us.URL
	}
}

func (s *Server) handleBuyout(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	set := s.trackSet(gs, q)
	budget := floatParam(q, "budget", 500)
	minFloor := floatParam(q, "minFloor", 20)
	shipping := floatParam(q, "shipping", 15)
	sortBy := stringParam(q, "sort", "score")
	mainOnly := stringParam(q, "chars", "") == "main"
	spOnly := boolParam(q, "sp", false)
	mode := stringParam(q, "mode", "buyout")
	minGap := floatParam(q, "minGap", 50)
	top := intParam(q, "top", 40)
	cfg := tracking.MarketConfig{Langs: gs.Game.BuyoutFloorLangs(), MainTokens: gs.Game.MainCharTokens}
	matcher := compare.MatcherFor(gs.Game)
	fxRate := 0.0
	var usIndex map[string]model.USPrice
	// US resale enrichment applies to games with a deals snapshot.
	if gs.Deals != nil {
		_, _, snap := gs.Deals.Status()
		usIndex = compare.USDIndex(snap.Prices, matcher)
		fxRate = snap.FXRate
	}
	resp := buyoutResponse{Set: set, Budget: budget, MinFloor: minFloor, Shipping: shipping, FXRate: fxRate, Sort: sortBy, Chars: stringParam(q, "chars", ""), Mode: mode, MinGap: minGap, Candidates: []tracking.BuyoutCandidate{}}
	analyze := func(day tracking.DaySnapshot, top int) []tracking.BuyoutCandidate {
		if mode == "snipe" {
			return tracking.Snipe(day, minFloor, minGap, shipping, mainOnly, spOnly, sortBy, top, cfg)
		}
		return tracking.Buyout(day, budget, minFloor, shipping, mainOnly, spOnly, sortBy, top, cfg)
	}
	if gs.Track == nil {
		writeJSON(w, resp)
		return
	}
	if set == "ALL" {
		merged := []tracking.BuyoutCandidate{}
		latest := ""
		for _, sc := range s.trackSets(gs) {
			dates, err := gs.Track.ListDates(sc)
			if err != nil || len(dates) == 0 {
				continue
			}
			date := dates[len(dates)-1]
			day, ok, err := gs.Track.LoadDay(sc, date)
			if err != nil || !ok {
				continue
			}
			resp.Ready = true
			if date > latest {
				latest = date
			}
			cands := analyze(day, 0)
			for i := range cands {
				cands[i].Set = sc
			}
			merged = append(merged, cands...)
		}
		resp.Date = latest
		resp.Candidates = tracking.RankBuyout(merged, sortBy, top)
		enrichUSD(resp.Candidates, usIndex, matcher, "")
		writeJSON(w, resp)
		return
	}
	dates, err := gs.Track.ListDates(set)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(dates) == 0 {
		writeJSON(w, resp)
		return
	}
	day, ok, err := gs.Track.LoadDay(set, dates[len(dates)-1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ok {
		resp.Ready = true
		resp.Date = day.Date
		resp.Candidates = analyze(day, top)
		enrichUSD(resp.Candidates, usIndex, matcher, set)
	}
	writeJSON(w, resp)
}

type trackCard struct {
	Number string  `json:"number"`
	Name   string  `json:"name"`
	LowBRL float64 `json:"lowBRL"`
}

func (s *Server) handleTrackingCards(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	set := s.trackSet(gs, q)
	resp := struct {
		Set   string      `json:"set"`
		Cards []trackCard `json:"cards"`
	}{Set: set, Cards: []trackCard{}}
	if gs.Track == nil {
		writeJSON(w, resp)
		return
	}
	day, ok, err := gs.Track.LatestDay(set)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ok {
		for _, c := range day.Cards {
			resp.Cards = append(resp.Cards, trackCard{Number: c.Number, Name: c.Name, LowBRL: c.LowBRL})
		}
		sort.SliceStable(resp.Cards, func(i, j int) bool {
			if resp.Cards[i].LowBRL != resp.Cards[j].LowBRL {
				return resp.Cards[i].LowBRL > resp.Cards[j].LowBRL
			}
			return resp.Cards[i].Number < resp.Cards[j].Number
		})
	}
	writeJSON(w, resp)
}

func (s *Server) cardPageURL(gs *GameStack, set, number string) string {
	if gs.Track == nil {
		return ""
	}
	number = model.NormalizeNumber(number)
	if set != "" {
		if day, ok, err := gs.Track.LatestDay(set); err == nil && ok {
			for _, c := range day.Cards {
				if model.NormalizeNumber(c.Number) == number {
					return c.URL
				}
			}
		}
	}
	// The cross-set number index is only safe when numbers are globally unique;
	// for per-set-numbered games it would resolve another set's card.
	if !gs.Game.UniqueCardNumbers {
		return ""
	}
	if u, ok := gs.Track.PageURLByNumber(number); ok {
		return u
	}
	return ""
}

func (s *Server) handleCardImage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	gs := s.stackFor(q)
	if gs.Cardimg == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	// A TCGplayer product id names the exact print (variants share a card number
	// but not a product), so it resolves art the number-keyed Liga lookup can't —
	// e.g. Riftbound's "GG EZ" treatments. Proxy it so a browser canvas draws it
	// same-origin; fall through to the Liga page lookup when it's absent or fails.
	if id, err := strconv.Atoi(q.Get("productID")); err == nil && id > 0 {
		url := fmt.Sprintf("https://product-images.tcgplayer.com/fit-in/400x559/%d.jpg", id)
		if body, err := gs.Cardimg.FetchURL(r.Context(), url); err == nil {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "public, max-age=86400")
			w.Write(body)
			return
		}
	}
	set := q.Get("set")
	number := model.NormalizeNumber(q.Get("number"))
	if set == "" || number == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	pageURL := s.cardPageURL(gs, set, number)
	if pageURL == "" {
		if u := q.Get("url"); isLigaCardURL(gs.Game, u) {
			pageURL = u
		}
	}
	if pageURL == "" {
		http.Error(w, "unknown card", http.StatusNotFound)
		return
	}
	body, err := gs.Cardimg.Fetch(r.Context(), set, number, pageURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(body)
}

func isLigaCardURL(g game.Game, raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return false
	}
	return g.AllowsHost(u.Host)
}

type exportCard struct {
	Set    string `json:"set"`
	Number string `json:"number"`
}

type exportRequest struct {
	Cards []exportCard `json:"cards"`
	Cols  int          `json:"cols"`
}

const maxExportCards = 60

func (s *Server) handleExportImage(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Cardimg == nil {
		http.Error(w, "image export unavailable", http.StatusServiceUnavailable)
		return
	}
	var req exportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if len(req.Cards) == 0 {
		http.Error(w, "no cards selected", http.StatusBadRequest)
		return
	}
	if len(req.Cards) > maxExportCards {
		req.Cards = req.Cards[:maxExportCards]
	}

	urls := make([]string, len(req.Cards))
	pages := map[string]map[string]string{}
	for i, c := range req.Cards {
		m, ok := pages[c.Set]
		if !ok {
			m = map[string]string{}
			if day, found, err := gs.Track.LatestDay(c.Set); err == nil && found {
				for _, cd := range day.Cards {
					m[cd.Number] = cd.URL
				}
			}
			pages[c.Set] = m
		}
		urls[i] = m[model.NormalizeNumber(c.Number)]
	}

	images := make([][]byte, len(req.Cards))
	g, ctx := errgroup.WithContext(r.Context())
	g.SetLimit(6)
	for i := range req.Cards {
		i := i
		if urls[i] == "" {
			continue
		}
		g.Go(func() error {
			b, err := gs.Cardimg.Fetch(ctx, req.Cards[i].Set, model.NormalizeNumber(req.Cards[i].Number), urls[i])
			if err == nil {
				images[i] = b
			}
			return nil
		})
	}
	g.Wait()

	ordered := make([][]byte, 0, len(images))
	for _, b := range images {
		if b != nil {
			ordered = append(ordered, b)
		}
	}
	png, err := cardimg.Grid(ordered, req.Cols)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", `attachment; filename="compro-cartas.png"`)
	w.Write(png)
}

func (s *Server) handleCapture(w http.ResponseWriter, r *http.Request) {
	if s.readOnly {
		writeJSON(w, map[string]bool{"started": false})
		return
	}
	gs := s.stackFor(r.URL.Query())
	if gs.Capturer == nil {
		writeJSON(w, map[string]bool{"started": false})
		return
	}
	go gs.Capturer.CaptureSingles(context.Background(), time.Now())
	writeJSON(w, map[string]bool{"started": true})
}

func (s *Server) handleCaptureSealed(w http.ResponseWriter, r *http.Request) {
	if s.readOnly {
		writeJSON(w, map[string]bool{"started": false})
		return
	}
	gs := s.stackFor(r.URL.Query())
	if gs.Capturer == nil {
		writeJSON(w, map[string]bool{"started": false})
		return
	}
	go gs.Capturer.CaptureSealed(context.Background(), time.Now())
	writeJSON(w, map[string]bool{"started": true})
}

func (s *Server) spaHandler() http.Handler {
	fs := http.FileServer(http.Dir(s.webDir))
	index := filepath.Join(s.webDir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(s.webDir, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, index)
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func floatParam(q map[string][]string, key string, def float64) float64 {
	if vals, ok := q[key]; ok && len(vals) > 0 {
		if v, err := strconv.ParseFloat(vals[0], 64); err == nil {
			return v
		}
	}
	return def
}

func intParam(q map[string][]string, key string, def int) int {
	if vals, ok := q[key]; ok && len(vals) > 0 {
		if v, err := strconv.Atoi(vals[0]); err == nil {
			return v
		}
	}
	return def
}

func boolParam(q map[string][]string, key string, def bool) bool {
	if vals, ok := q[key]; ok && len(vals) > 0 {
		if v, err := strconv.ParseBool(vals[0]); err == nil {
			return v
		}
	}
	return def
}

func stringParam(q map[string][]string, key, def string) string {
	if vals, ok := q[key]; ok && len(vals) > 0 && vals[0] != "" {
		return vals[0]
	}
	return def
}
