package model

import (
	"context"
	"strings"
	"time"
)

type BrazilListing struct {
	Number       string
	SetCode      string
	Name         string
	Variant      string
	Source       string
	URL          string
	LowBRL       float64
	AvgBRL       float64
	StockChecked bool
	InStock      bool
	// FloorCopies is the number of copies available at the English NM floor and
	// Sellers the count of distinct stores with English NM stock. Both are set by
	// the stock-verification pass (0 when unverified) and answer "is this floor one
	// lucky flip or real supply I can load up on".
	FloorCopies int
	Sellers     int
}

type USPrice struct {
	Number    string
	Name      string
	Variant   string
	Rarity    string
	SetCode   string
	URL       string
	ProductID int
	MarketUSD float64
	LowUSD    float64
	LiveUSD   float64
	// LiveListings is how many trusted (gold-star, English, NM) sellers back the
	// live floor and LiveQty the total copies available among them. Both are set
	// only on the live-price path (0 when a card wasn't live-priced) and answer
	// "is this US comparison price real or one optimistic listing".
	LiveListings int
	LiveQty      int
}

type Deal struct {
	Number     string  `json:"number"`
	Name       string  `json:"name"`
	Set        string  `json:"set,omitempty"`
	Rarity     string  `json:"rarity"`
	Variant    string  `json:"variant"`
	Source     string  `json:"source"`
	BuyURL     string  `json:"buyUrl"`
	TCGURL     string  `json:"tcgUrl"`
	LowBRL     float64 `json:"lowBRL"`
	BuyUSD     float64 `json:"buyUSD"`
	SellUSD    float64 `json:"sellUSD"`
	MarginPct  float64 `json:"marginPct"`
	ProfitUSD  float64 `json:"profitUSD"`
	Verified   bool    `json:"verified"`
	USListings int     `json:"usListings"`
	USQty      int     `json:"usQty"`
	BRCopies   int     `json:"brCopies"`
	BRSellers  int     `json:"brSellers"`
}

type Snapshot struct {
	Listings  []BrazilListing `json:"listings"`
	Prices    []USPrice       `json:"prices"`
	FXRate    float64         `json:"fxRate"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type BrazilSource interface {
	Name() string
	Listings(ctx context.Context) ([]BrazilListing, error)
}

// StockVerifier is implemented by Brazil sources whose listing prices can
// persist with no current sellers, so candidate listings need a per-card stock
// check after the main fetch. Sources whose listings already carry live stock
// (StockChecked at parse time) do not implement this.
type StockVerifier interface {
	StockOf(ctx context.Context, listings []BrazilListing) map[string]bool
}

type EnglishStock struct {
	InStock  bool
	FloorBRL float64
	// Copies is how many copies are available at the English NM floor and Sellers
	// the number of distinct stores holding English NM stock — the buy-side depth.
	Copies  int
	Sellers int
}

type StockPricer interface {
	EnglishStock(ctx context.Context, listings []BrazilListing) map[string]EnglishStock
}

type USPriceSource interface {
	Prices(ctx context.Context) ([]USPrice, error)
}

func NormalizeNumber(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
