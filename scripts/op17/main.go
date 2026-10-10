package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"

	"opdeals/internal/clearance"
	"opdeals/internal/game"
	"opdeals/internal/httpx"
	"opdeals/internal/liga"
	"opdeals/internal/logx"
	"opdeals/internal/model"
)

type card struct {
	Set        string `json:"set"`
	Num        string `json:"num"`
	Name       string `json:"name"`
	Rarity     string `json:"rarity"`
	PrecoMenor string `json:"precoMenor"`
	P1a        string `json:"p1a"`
}

var langName = map[string]string{"2": "EN", "6": "JP", "8": "PT", "10": "CN"}
var condName = map[string]string{"1": "M", "2": "NM", "3": "SP", "4": "MP", "5": "HP", "6": "D"}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	in := flag.String("in", "", "cards json")
	out := flag.String("out", "", "csv out")
	clearancePath := flag.String("cf-clearance", "data/cf-clearance.json", "clearance file")
	proxy := flag.String("liga-proxy", "http://127.0.0.1:8891", "cfgate proxy")
	flag.Parse()

	raw, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	var cards []card
	if err := json.Unmarshal(raw, &cards); err != nil {
		return err
	}

	g := game.OnePiece()
	store, err := clearance.NewStore(*clearancePath).WithProxy(*proxy)
	if err != nil {
		return err
	}
	logger := logx.New(os.Stderr)
	ligaHTTP, err := httpx.NewThrottled(30 * time.Second)
	if err != nil {
		return err
	}
	sets := map[string]bool{}
	for i := range cards {
		if cards[i].Set == "" {
			cards[i].Set = "OP-17"
		}
		sets[cards[i].Set] = true
	}
	setList := make([]string, 0, len(sets))
	for s := range sets {
		setList = append(setList, s)
	}
	client := liga.New(store.FetcherFor()(ligaHTTP, g), logger, 2, setList, g)

	listings := make([]model.BrazilListing, 0, len(cards))
	byURL := make(map[string]card, len(cards))
	for _, c := range cards {
		u := cardURL(g.LigaBaseURL, c.Set, c.Num, c.Name)
		listings = append(listings, model.BrazilListing{Number: c.Num, SetCode: c.Set, Name: c.Name, URL: u})
		byURL[u] = c
	}

	detail := client.StockDetail(context.Background(), listings)

	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{
		"numero", "nome", "raridade", "menor_brl", "idioma_menor", "cond_menor", "loja_menor", "uf",
		"menor_en_nm_brl", "lojas_en_nm", "copias_no_piso_en_nm", "ofertas_total", "grid_preco_menor", "url",
	}); err != nil {
		return err
	}

	sort.Slice(listings, func(i, j int) bool { return listings[i].Number < listings[j].Number })
	for _, l := range listings {
		c := byURL[l.URL]
		rows := detail[l.URL]
		low, lang, cond, store, uf := cheapest(rows)
		enNM, _ := liga.FloorNM(rows, g.FloorLangs)
		copies, sellers := enNMDepth(rows, enNM)
		rec := []string{
			c.Num, c.Name, c.Rarity,
			money(low), langName[lang], condName[cond], store, uf,
			money(enNM), strconv.Itoa(sellers), strconv.Itoa(copies),
			strconv.Itoa(len(rows)), c.PrecoMenor, l.URL,
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d cards, %d with detail)\n", *out, len(listings), len(detail))
	return nil
}

func cheapest(rows []liga.StoreListing) (float64, string, string, string, string) {
	var best liga.StoreListing
	low := 0.0
	for _, r := range rows {
		if !r.PriceKnown || r.PriceBRL <= 0 {
			continue
		}
		if r.QtyKnown && r.Quantity <= 0 {
			continue
		}
		if low == 0 || r.PriceBRL < low {
			low, best = r.PriceBRL, r
		}
	}
	return low, best.Language, best.Condition, best.StoreName, best.StoreUF
}

func enNMDepth(rows []liga.StoreListing, floor float64) (copies, sellers int) {
	stores := map[int]bool{}
	for _, r := range rows {
		if r.Condition != "2" || r.Language != "2" {
			continue
		}
		if r.QtyKnown && r.Quantity <= 0 {
			continue
		}
		stores[r.StoreID] = true
		if floor > 0 && r.PriceKnown && r.PriceBRL == floor {
			if r.QtyKnown {
				copies += r.Quantity
			} else {
				copies++
			}
		}
	}
	return copies, len(stores)
}

func money(v float64) string {
	if v <= 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func cardURL(baseURL, setCode, number, name string) string {
	q := url.Values{}
	q.Set("view", "cards/card")
	q.Set("card", name)
	q.Set("ed", setCode)
	q.Set("num", number)
	return baseURL + "?" + q.Encode()
}
