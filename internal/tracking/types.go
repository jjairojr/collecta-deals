package tracking

import "time"

type DaySnapshot struct {
	Set        string    `json:"set"`
	Date       string    `json:"date"`
	CapturedAt time.Time `json:"capturedAt"`
	FXRate     float64   `json:"fxRate"`
	Cards      []CardDay `json:"cards"`
}

type CardDay struct {
	Number string     `json:"number"`
	Name   string     `json:"name"`
	LowBRL float64    `json:"lowBRL"`
	AvgBRL float64    `json:"avgBRL"`
	URL    string     `json:"url"`
	Stores []StoreQty `json:"stores"`
}

type StoreQty struct {
	StoreID    int     `json:"storeId"`
	StoreName  string  `json:"storeName"`
	Condition  string  `json:"condition"`
	Language   string  `json:"language"`
	Quantity   int     `json:"quantity"`
	Known      bool    `json:"known"`
	PriceBRL   float64 `json:"priceBRL"`
	PriceKnown bool    `json:"priceKnown"`
}

type CardTrend struct {
	Set      string  `json:"set,omitempty"`
	Number   string  `json:"number"`
	Name     string  `json:"name"`
	LowBRL   float64 `json:"lowBRL"`
	PrevBRL  float64 `json:"prevBRL"`
	DeltaPct float64 `json:"deltaPct"`
	URL      string  `json:"url"`
}

type PricePoint struct {
	Date   string  `json:"date"`
	LowBRL float64 `json:"lowBRL"`
}

type CardSale struct {
	Set        string       `json:"set,omitempty"`
	Number     string       `json:"number"`
	Name       string       `json:"name"`
	URL        string       `json:"url,omitempty"`
	Units      int          `json:"units"`
	RevenueBRL float64      `json:"revenueBRL"`
	Sellers    []CardSeller `json:"sellers,omitempty"`
}

type CardSeller struct {
	StoreID    int     `json:"storeId"`
	StoreName  string  `json:"storeName"`
	Units      int     `json:"units"`
	RevenueBRL float64 `json:"revenueBRL"`
	// PriceBRL is the store's own per-unit sale price (revenue / units), i.e. what
	// this store was actually listing the sold copies at — not the card's floor.
	PriceBRL float64 `json:"priceBRL"`
}

type SnapshotSales struct {
	Date           string     `json:"date"`
	PrevDate       string     `json:"prevDate"`
	CapturedAt     time.Time  `json:"capturedAt"`
	PrevCapturedAt time.Time  `json:"prevCapturedAt"`
	Units          int        `json:"units"`
	RevenueBRL     float64    `json:"revenueBRL"`
	Cards          []CardSale `json:"cards"`
}

type StoreStat struct {
	StoreID    int        `json:"storeId"`
	StoreName  string     `json:"storeName"`
	UnitsSold  int        `json:"unitsSold"`
	RevenueBRL float64    `json:"revenueBRL"`
	Cards      []CardSale `json:"cards"`
}

type StoreInventoryStat struct {
	StoreID       int     `json:"storeId"`
	StoreName     string  `json:"storeName"`
	Units         int     `json:"units"`
	Cards         int     `json:"cards"`
	ValueBRL      float64 `json:"valueBRL"`
	TopCardNumber string  `json:"topCardNumber"`
	TopCardName   string  `json:"topCardName"`
	TopCardBRL    float64 `json:"topCardBRL"`
}

type CardHolder struct {
	StoreID   int    `json:"storeId"`
	StoreName string `json:"storeName"`
	Quantity  int    `json:"quantity"`
}

type ExpensiveCard struct {
	Number   string       `json:"number"`
	Name     string       `json:"name"`
	LowBRL   float64      `json:"lowBRL"`
	TotalQty int          `json:"totalQty"`
	Stores   int          `json:"stores"`
	Holders  []CardHolder `json:"holders"`
}

type InventorySummary struct {
	Date         string               `json:"date"`
	ActiveStores int                  `json:"activeStores"`
	TotalUnits   int                  `json:"totalUnits"`
	TotalValue   float64              `json:"totalValue"`
	Stores       []StoreInventoryStat `json:"stores"`
	Expensive    []ExpensiveCard      `json:"expensive"`
}
