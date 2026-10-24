package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"opdeals/internal/clearance"
	"opdeals/internal/game"
	"opdeals/internal/httpx"
	"opdeals/internal/liga"
	"opdeals/internal/logx"
	"opdeals/internal/model"
)

type trade struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	Number       string  `json:"number"`
	Name         string  `json:"name"`
	Set          string  `json:"set"`
	Variant      string  `json:"variant"`
	Condition    string  `json:"condition"`
	Qty          int     `json:"qty"`
	BuyBRL       float64 `json:"buyBRL"`
	CostBRL      float64 `json:"costBRL"`
	AskBRL       float64 `json:"askBRL"`
	Status       string  `json:"status"`
	LigaListed   bool    `json:"ligaListed"`
	LigaPriceBRL float64 `json:"ligaPriceBRL"`
}

type tradesResponse struct {
	Trades []trade `json:"trades"`
}

type offer struct {
	Store     string  `json:"store"`
	StoreID   int     `json:"storeId"`
	Condition string  `json:"condition"`
	Language  string  `json:"language"`
	Extras    *int    `json:"extras"`
	Qty       int     `json:"qty"`
	QtyKnown  bool    `json:"qtyKnown"`
	PriceBRL  float64 `json:"priceBRL"`
	Known     bool    `json:"priceKnown"`
}

type row struct {
	Trade  trade   `json:"trade"`
	URL    string  `json:"url"`
	Offers []offer `json:"offers"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	gameID := flag.String("game", "riftbound", "game id")
	base := flag.String("base", "https://collecta-deals-production.up.railway.app", "dashboard base URL holding the trade ledger")
	cfPath := flag.String("cf-clearance", "data/cf-clearance.json", "Cloudflare clearance file written by cmd/cfgate")
	proxy := flag.String("liga-proxy", "http://localhost:8891", "CONNECT proxy for Liga hosts")
	outPath := flag.String("out", "", "path to write the JSON result (default stdout)")
	adminToken := flag.String("admin-token", os.Getenv("ADMIN_TOKEN"), "X-Admin-Token for the dashboard API")
	flag.Parse()

	g := game.ByID(*gameID)
	log := logx.New(os.Stderr)

	held, err := fetchHeld(*base, *gameID, *adminToken)
	if err != nil {
		return err
	}
	if len(held) == 0 {
		return fmt.Errorf("no held singles for %s", *gameID)
	}

	sets := setsOf(held)
	if g.UniqueCardNumbers {
		sets = nil
	}
	log.Printf("MAIN %d held singles across sets %s", len(held), strings.Join(sets, ","))

	store, err := clearance.NewStore(*cfPath).WithProxy(*proxy)
	if err != nil {
		return err
	}
	direct, err := httpx.NewThrottled(45 * time.Second)
	if err != nil {
		return err
	}
	client := liga.New(store.FetcherFor()(direct, g), log, 2, sets, g)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	listings, err := client.Listings(ctx)
	if err != nil {
		return err
	}
	index := map[string]model.BrazilListing{}
	for _, l := range listings {
		for _, k := range numberKeys(l.SetCode, l.Number, g.UniqueCardNumbers) {
			if _, ok := index[k]; !ok {
				index[k] = l
			}
		}
	}

	wanted := map[string]model.BrazilListing{}
	matched := map[string]model.BrazilListing{}
	var missing []trade
	for _, t := range held {
		l, ok := lookup(index, t, g.UniqueCardNumbers)
		if !ok {
			missing = append(missing, t)
			continue
		}
		matched[t.ID] = l
		wanted[l.URL] = l
	}
	for _, t := range missing {
		log.Printf("MAIN sem carta na Liga: %s %s %s", t.Set, t.Number, t.Name)
	}

	targets := make([]model.BrazilListing, 0, len(wanted))
	for _, l := range wanted {
		targets = append(targets, l)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].URL < targets[j].URL })

	detail := client.StockDetail(ctx, targets)

	rows := make([]row, 0, len(held))
	for _, t := range held {
		l, ok := matched[t.ID]
		if !ok {
			continue
		}
		var offers []offer
		for _, s := range detail[l.URL] {
			if model.NormalizeNumber(s.Number) != model.NormalizeNumber(l.Number) {
				continue
			}
			offers = append(offers, offer{
				Store:     s.StoreName,
				StoreID:   s.StoreID,
				Condition: s.Condition,
				Language:  s.Language,
				Extras:    s.Extras,
				Qty:       s.Quantity,
				QtyKnown:  s.QtyKnown,
				PriceBRL:  s.PriceBRL,
				Known:     s.PriceKnown,
			})
		}
		sort.Slice(offers, func(i, j int) bool { return offers[i].PriceBRL < offers[j].PriceBRL })
		rows = append(rows, row{Trade: t, URL: l.URL, Offers: offers})
	}

	out, err := json.MarshalIndent(rows, "", " ")
	if err != nil {
		return err
	}
	if *outPath == "" {
		fmt.Println(string(out))
		return nil
	}
	return os.WriteFile(*outPath, out, 0o644)
}

func fetchHeld(base, gameID, adminToken string) ([]trade, error) {
	req, err := http.NewRequest(http.MethodGet, base+"/api/trades?game="+gameID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Admin-Token", adminToken)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("trades: HTTP %d", resp.StatusCode)
	}
	var body tradesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	out := make([]trade, 0, len(body.Trades))
	for _, t := range body.Trades {
		if t.Status != "holding" || t.Qty <= 0 {
			continue
		}
		if t.Kind != "" && t.Kind != "single" {
			continue
		}
		if t.Number == "" {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func numberKeys(set, number string, numberOnly bool) []string {
	n := model.NormalizeNumber(number)
	if i := strings.Index(n, "/"); i >= 0 {
		n = n[:i]
	}
	seen := map[string]struct{}{}
	var out []string
	for _, v := range []string{n, strings.TrimLeft(n, "0")} {
		if v == "" {
			continue
		}
		k := v
		if !numberOnly {
			k = model.NormalizeNumber(set) + "|" + v
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

func lookup(index map[string]model.BrazilListing, t trade, numberOnly bool) (model.BrazilListing, bool) {
	for _, k := range numberKeys(t.Set, t.Number, numberOnly) {
		if l, ok := index[k]; ok {
			return l, true
		}
	}
	return model.BrazilListing{}, false
}

func setsOf(ts []trade) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, t := range ts {
		if t.Set == "" {
			continue
		}
		if _, ok := seen[t.Set]; ok {
			continue
		}
		seen[t.Set] = struct{}{}
		out = append(out, t.Set)
	}
	sort.Strings(out)
	return out
}
