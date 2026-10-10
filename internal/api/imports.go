package api

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"opdeals/internal/compare"
	"opdeals/internal/imports"
	"opdeals/internal/tracking"
)

const (
	importDepthLookback = 8
	importDefaultGames  = "onepiece,riftbound"
)

type importsGame struct {
	Game         gameInfo  `json:"game"`
	Ready        bool      `json:"ready"`
	FXRate       float64   `json:"fxRate"`
	UpdatedAt    time.Time `json:"updatedAt"`
	TrackingDate string    `json:"trackingDate"`
	Sets         int       `json:"sets"`
}

type importsResponse struct {
	Direction string               `json:"direction"`
	Games     []importsGame        `json:"games"`
	Count     int                  `json:"count"`
	Deals     []imports.Deal       `json:"deals,omitempty"`
	Exports   []imports.ExportDeal `json:"exports,omitempty"`
}

type supplyEntry struct {
	key          string
	trackingDate string
	sets         int
	supply       []tracking.ImportSupply
}

type importsCache struct {
	mu      sync.Mutex
	entries map[string]*supplyEntry
	builds  map[string]*sync.Mutex
}

func newImportsCache() *importsCache {
	return &importsCache{entries: map[string]*supplyEntry{}, builds: map[string]*sync.Mutex{}}
}

func (c *importsCache) buildLock(game string) *sync.Mutex {
	c.mu.Lock()
	defer c.mu.Unlock()
	l := c.builds[game]
	if l == nil {
		l = &sync.Mutex{}
		c.builds[game] = l
	}
	return l
}

func (c *importsCache) get(game, key string) (*supplyEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[game]
	if e == nil || e.key != key {
		return nil, false
	}
	return e, true
}

func (c *importsCache) put(game string, e *supplyEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[game] = e
}

func (s *Server) handleImports(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	direction := stringParam(q, "direction", "import")
	export := direction == "export"
	resp := importsResponse{Direction: direction, Games: []importsGame{}}
	var (
		all     []imports.Deal
		allExps []imports.ExportDeal
	)
	for _, id := range strings.Split(stringParam(q, "games", importDefaultGames), ",") {
		gs, ok := s.games[strings.TrimSpace(id)]
		if !ok || gs.Deals == nil || gs.Track == nil {
			continue
		}
		entry, info, err := s.importSupply(gs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp.Games = append(resp.Games, info)
		if entry == nil {
			continue
		}
		_, _, snap := gs.Deals.Status()
		opts := imports.BuildOptions{Game: gs.Game.ID, FXRate: snap.FXRate}
		matcher := compare.MatcherFor(gs.Game)
		if export {
			allExps = append(allExps, imports.BuildExport(entry.supply, snap.Prices, matcher, opts)...)
		} else {
			all = append(all, imports.Build(entry.supply, snap.Prices, matcher, opts)...)
		}
	}
	if export {
		resp.Exports = imports.ApplyExport(allExps, imports.ExportFilter{
			MinSellers:    intParam(q, "minSellers", 1),
			MinUSListings: intParam(q, "minUSListings", 2),
			MinUSD:        floatParam(q, "minUSD", 10),
			MinProfit:     floatParam(q, "minProfit", 10),
			SortBy:        stringParam(q, "sort", "score"),
			Limit:         intParam(q, "limit", 100),
		})
		if resp.Exports == nil {
			resp.Exports = []imports.ExportDeal{}
		}
		resp.Count = len(resp.Exports)
	} else {
		resp.Deals = imports.Apply(all, imports.Filter{
			MinSellers: intParam(q, "minSellers", 3),
			MinUSD:     floatParam(q, "minUSD", 3),
			MinProfit:  floatParam(q, "minProfit", 10),
			SortBy:     stringParam(q, "sort", "profit"),
			Limit:      intParam(q, "limit", 100),
		})
		if resp.Deals == nil {
			resp.Deals = []imports.Deal{}
		}
		resp.Count = len(resp.Deals)
	}
	writeJSON(w, resp)
}

func (s *Server) importSupply(gs *GameStack) (*supplyEntry, importsGame, error) {
	ready, _, snap := gs.Deals.Status()
	info := importsGame{Game: s.gameInfoFor(gs), Ready: ready, FXRate: snap.FXRate, UpdatedAt: snap.UpdatedAt}
	latest, err := s.latestTrackingDate(gs)
	if err != nil {
		return nil, info, err
	}
	if latest == "" || !ready {
		return nil, info, nil
	}
	key := latest
	if e, ok := s.imports.get(gs.Game.ID, key); ok {
		return e, withEntry(info, e), nil
	}
	lock := s.imports.buildLock(gs.Game.ID)
	lock.Lock()
	defer lock.Unlock()
	if e, ok := s.imports.get(gs.Game.ID, key); ok {
		return e, withEntry(info, e), nil
	}

	langs := gs.Game.BuyoutFloorLangs()
	entry := &supplyEntry{key: key}
	for _, sc := range s.trackSets(gs) {
		day, ok, err := gs.Track.LatestDayWithDepth(sc, importDepthLookback)
		if err != nil {
			return nil, info, err
		}
		if !ok {
			continue
		}
		entry.supply = append(entry.supply, tracking.ImportSupplyFor(day, langs, s.OwnLigaStoreID)...)
		entry.sets++
		if day.Date > entry.trackingDate {
			entry.trackingDate = day.Date
		}
	}
	s.imports.put(gs.Game.ID, entry)
	return entry, withEntry(info, entry), nil
}

func withEntry(info importsGame, e *supplyEntry) importsGame {
	info.TrackingDate, info.Sets = e.trackingDate, e.sets
	return info
}

func (s *Server) latestTrackingDate(gs *GameStack) (string, error) {
	latest := ""
	for _, sc := range s.trackSets(gs) {
		dates, err := gs.Track.ListDates(sc)
		if err != nil {
			return "", err
		}
		if n := len(dates); n > 0 && dates[n-1] > latest {
			latest = dates[n-1]
		}
	}
	return latest, nil
}
