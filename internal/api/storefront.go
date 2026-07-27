package api

import (
	"encoding/json"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"opdeals/internal/trades"
)

// storefrontItem is the public, buyer-facing view of one held card. It carries
// NO cost basis, profit, or margin — only what a buyer needs to see. This is the
// security boundary between the internal portfolio and the public catalog.
type storefrontItem struct {
	Game      string  `json:"game"`
	GameLabel string  `json:"gameLabel"`
	Kind      string  `json:"kind,omitempty"` // "sealed" for sealed product, empty for singles
	Number    string  `json:"number"`
	Name      string  `json:"name"`
	Set       string  `json:"set"`
	Variant   string  `json:"variant,omitempty"`
	Condition string  `json:"condition,omitempty"`
	Qty       int     `json:"qty"`
	ProductID int     `json:"productID,omitempty"`
	ImageURL  string  `json:"imageURL,omitempty"`
	AskBRL    float64 `json:"askBRL"`
	AskUSD    float64 `json:"askUSD,omitempty"`
	Featured  bool    `json:"featured,omitempty"`
}

type storefrontResponse struct {
	FXRate float64          `json:"fxRate"`
	Count  int              `json:"count"`
	Items  []storefrontItem `json:"items"`
}

// handleStorefront returns every listed, in-stock holding across all games,
// stripped of cost/profit, for the public catalog site. A card appears only when
// it is a holding (not sold), has qty, is flagged Listed, and has an asking price.
func (s *Server) handleStorefront(w http.ResponseWriter, r *http.Request) {
	resp := storefrontResponse{Items: []storefrontItem{}}
	fx := s.storefrontFX()
	resp.FXRate = fx
	for _, id := range s.orderedGameIDs() {
		gs := s.games[id]
		if gs.Trades == nil {
			continue
		}
		all, err := gs.Trades.List()
		if err != nil {
			continue
		}
		lookup, _ := s.priceLookup(gs)
		label := gs.Game.Name
		for _, t := range all {
			if t.Status != "holding" || !t.Listed || t.AskBRL <= 0 {
				continue
			}
			qty := t.Qty
			if qty <= 0 {
				qty = 1
			}
			item := storefrontItem{
				Game:      gs.Game.ID,
				GameLabel: label,
				Kind:      t.Kind,
				Number:    t.Number,
				Name:      t.Name,
				Set:       t.Set,
				Variant:   t.Variant,
				Condition: t.Condition,
				Qty:       qty,
				AskBRL:    round2(t.AskBRL),
				Featured:  t.Featured,
			}
			// A seller-supplied image wins over everything (the only art for
			// products without TCG data, e.g. sealed starters). Otherwise the
			// catalog resolves art through /api/card-image, which tries the
			// TCGplayer product art first and falls back to the Liga page image
			// when that print has none — the same chain the dashboard uses. The
			// product id only names the print; it is never turned into a bare
			// CDN url here, because a 404 there would leave a broken image with
			// nothing behind it.
			if t.ImageURL != "" {
				item.ImageURL = t.ImageURL
			} else if t.Kind != "sealed" && lookup != nil {
				if _, url, ok := lookup(t.Number, t.Name, t.Set); ok {
					item.ProductID = productIDFromURL(url)
				}
			}
			if fx > 0 {
				item.AskUSD = round2(t.AskBRL * fx)
			}
			resp.Items = append(resp.Items, item)
		}
	}
	resp.Count = len(resp.Items)
	writeJSON(w, resp)
}

// storefrontFX returns a single USD-per-BRL rate (from the first deals-enabled
// game's snapshot) used to display USD asking prices for every game, so a BR-only
// game's prices convert with the same rate rather than its placeholder fx of 1.
func (s *Server) storefrontFX() float64 {
	for _, id := range s.orderedGameIDs() {
		gs := s.games[id]
		if gs.Deals == nil {
			continue
		}
		if fx := gs.Deals.Snapshot().FXRate; fx > 0 {
			return fx
		}
	}
	return 0
}

type listingInput struct {
	ID       string  `json:"id"`
	AskBRL   float64 `json:"askBRL"`
	Listed   bool    `json:"listed"`
	ImageURL string  `json:"imageURL"`
	Featured bool    `json:"featured"`
}

// handleTradesListings sets storefront state (asking price + listed flag) on one
// or more holdings of the active game in a single write. Used by the Estoque tab
// for both single-card edits and bulk "list all / suggest prices" actions.
func (s *Server) handleTradesListings(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	var in struct {
		Items []listingInput `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	updates := make(map[string]trades.Listing, len(in.Items))
	for _, it := range in.Items {
		if it.ID == "" {
			continue
		}
		ask := it.AskBRL
		if ask < 0 {
			ask = 0
		}
		updates[it.ID] = trades.Listing{AskBRL: ask, Listed: it.Listed, ImageURL: strings.TrimSpace(it.ImageURL), Featured: it.Featured}
	}
	n, err := gs.Trades.SetListings(updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]int{"updated": n})
}

var productIDRe = regexp.MustCompile(`/product/(\d+)`)

// productIDFromURL pulls the TCGplayer product id out of a tcgUrl
// (".../product/696052"), so storefront art can use the exact per-variant image.
func productIDFromURL(url string) int {
	m := productIDRe.FindStringSubmatch(url)
	if m == nil {
		return 0
	}
	id, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return id
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
