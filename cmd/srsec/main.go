package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"opdeals/internal/clearance"
	"opdeals/internal/compare"
	"opdeals/internal/fx"
	"opdeals/internal/game"
	"opdeals/internal/httpx"
	"opdeals/internal/liga"
	"opdeals/internal/logx"
	"opdeals/internal/model"
	"opdeals/internal/tcgcsv"
	"opdeals/internal/tcgplayer"
)

type row struct {
	US    model.USPrice
	BR    model.BrazilListing
	Stock model.EnglishStock
	HasBR bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	setsFlag := flag.String("sets", "OP-11,OP-12,OP-13,OP-14,OP-15,OP-16,OP-17", "Liga set codes")
	rarities := flag.String("rarities", "SR,SEC", "TCGCSV rarities to keep")
	cfPath := flag.String("cf-clearance", "data/cf-clearance.json", "clearance file from cmd/cfgate")
	proxy := flag.String("liga-proxy", "http://localhost:8891", "CONNECT proxy for Liga hosts")
	outPath := flag.String("out", "", "CSV output path")
	only := flag.String("only", "", "comma-separated TCGCSV numbers to restrict to")
	flag.Parse()
	onlySet := map[string]bool{}
	for _, n := range strings.Split(*only, ",") {
		if n = strings.TrimSpace(strings.ToUpper(n)); n != "" {
			onlySet[n] = true
		}
	}

	log := logx.New(os.Stderr)
	g := game.OnePiece()
	matcher := compare.MatcherFor(g)
	sets := strings.Split(*setsFlag, ",")
	prefixes := make([]string, 0, len(sets))
	for _, s := range sets {
		prefixes = append(prefixes, strings.ReplaceAll(s, "-", "")+"-")
	}
	keep := map[string]bool{}
	for _, r := range strings.Split(*rarities, ",") {
		keep[strings.ToUpper(strings.TrimSpace(r))] = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Minute)
	defer cancel()

	client, err := httpx.New(30 * time.Second)
	if err != nil {
		return err
	}
	throttled, err := httpx.NewThrottled(45 * time.Second)
	if err != nil {
		return err
	}
	store, err := clearance.NewStore(*cfPath).WithProxy(*proxy)
	if err != nil {
		return err
	}
	lg := liga.New(store.FetcherFor()(throttled, g), log, 2, sets, g)
	us := tcgcsv.New(client, log, 8, g.Market)
	tcg := tcgplayer.New(client, log)

	rate, err := fx.Rate(ctx, client, log, 0)
	if err != nil {
		return err
	}
	prices, err := us.Prices(ctx)
	if err != nil {
		return err
	}
	wanted := make([]model.USPrice, 0)
	for _, p := range prices {
		if !keep[strings.ToUpper(p.Rarity)] {
			continue
		}
		if len(onlySet) > 0 && !onlySet[strings.ToUpper(p.Number)] {
			continue
		}
		for _, pre := range prefixes {
			if strings.HasPrefix(strings.ToUpper(p.Number), pre) {
				wanted = append(wanted, p)
				break
			}
		}
	}
	log.Printf("MAIN %d SR/SEC printings on TCGCSV", len(wanted))

	listings, err := lg.Listings(ctx)
	if err != nil {
		return err
	}
	brIndex := compare.BRIndex(listings, matcher)
	byNumber := map[string]model.BrazilListing{}
	for _, l := range listings {
		n := strings.ToUpper(l.Number)
		if cur, ok := byNumber[n]; !ok || l.LowBRL < cur.LowBRL {
			byNumber[n] = l
		}
	}

	rows := make([]row, 0, len(wanted))
	targets := map[string]model.BrazilListing{}
	ids := make([]int, 0, len(wanted))
	for _, p := range wanted {
		r := row{US: p}
		l, ok := brIndex[matcher.Key(p.Number, p.Name, p.SetCode)]
		if suffix, forced := overrideSuffix(p.Name); forced {
			l, ok = byNumber[strings.ToUpper(p.Number)+suffix]
		}
		if ok {
			r.BR, r.HasBR = l, true
			targets[l.URL] = l
		}
		if p.ProductID > 0 {
			ids = append(ids, p.ProductID)
		}
		rows = append(rows, r)
	}
	list := make([]model.BrazilListing, 0, len(targets))
	for _, l := range targets {
		list = append(list, l)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].URL < list[j].URL })

	done := make(chan map[int]tcgplayer.PriceInfo, 1)
	go func() { done <- tcg.LowestPrices(ctx, ids) }()
	stock := lg.EnglishStock(ctx, list)
	live := <-done

	for i := range rows {
		if v, ok := live[rows[i].US.ProductID]; ok {
			rows[i].US.LiveUSD = v.Price
			rows[i].US.LiveListings = v.Listings
			rows[i].US.LiveQty = v.Qty
		}
		if rows[i].HasBR {
			rows[i].Stock = stock[rows[i].BR.URL]
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].US.Number != rows[j].US.Number {
			return rows[i].US.Number < rows[j].US.Number
		}
		return rows[i].US.Name < rows[j].US.Name
	})

	out := os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}
	w := csv.NewWriter(out)
	defer w.Flush()
	w.Write([]string{"number", "name_tcg", "rarity", "variant", "liga_number", "liga_name", "us_live", "us_listings", "us_qty", "us_low_csv", "us_market_csv", "liga_grid_low", "liga_nm_en_floor", "liga_in_stock", "liga_copies", "liga_sellers", "fx", "tcg_url", "liga_url"})
	for _, r := range rows {
		rec := []string{
			r.US.Number, r.US.Name, r.US.Rarity, matcher.PrintKey(r.US.Number, r.US.Name, ""),
			r.BR.Number, r.BR.Name,
			f2(r.US.LiveUSD), strconv.Itoa(r.US.LiveListings), strconv.Itoa(r.US.LiveQty), f2(r.US.LowUSD), f2(r.US.MarketUSD),
			f2(r.BR.LowBRL), f2(r.Stock.FloorBRL), strconv.FormatBool(r.Stock.InStock), strconv.Itoa(r.Stock.Copies), strconv.Itoa(r.Stock.Sellers),
			strconv.FormatFloat(rate, 'f', 4, 64), r.US.URL, r.BR.URL,
		}
		w.Write(rec)
	}
	return nil
}

func f2(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func overrideSuffix(name string) (string, bool) {
	switch {
	case strings.Contains(name, "(Red Super Alternate Art)"):
		return "", true
	case strings.Contains(name, "(Super Alternate Art)"):
		if strings.HasPrefix(name, "Kaido") {
			return "-SA", true
		}
		return "-MA", true
	case strings.Contains(name, "(Parallel)"):
		return "-AA", true
	}
	return "", false
}
