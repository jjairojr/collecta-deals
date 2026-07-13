package trades

import "time"

type Trade struct {
	ID        string `json:"id"`
	Kind      string `json:"kind,omitempty"`
	Number    string `json:"number"`
	Name      string `json:"name"`
	Set       string `json:"set"`
	Variant   string `json:"variant,omitempty"`
	Condition string `json:"condition,omitempty"`
	Qty       int    `json:"qty"`

	BuyBRL      float64 `json:"buyBRL"`
	ShippingBRL float64 `json:"shippingBRL"`
	Store       string  `json:"store,omitempty"`
	BuyDate     string  `json:"buyDate,omitempty"`
	Delivered   bool    `json:"delivered,omitempty"`

	RefUSD    float64 `json:"refUSD,omitempty"`
	ManualBRL float64 `json:"manualBRL,omitempty"`

	Status string `json:"status"` // "holding" | "sold"

	SellPrice    float64 `json:"sellPrice,omitempty"`
	SellCurrency string  `json:"sellCurrency,omitempty"` // "BRL" | "USD"
	SellDate     string  `json:"sellDate,omitempty"`
	Buyer        string  `json:"buyer,omitempty"`

	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TradeView struct {
	Trade
	CostBRL     float64 `json:"costBRL"`
	MarketUSD   float64 `json:"marketUSD"`
	MarketKnown bool    `json:"marketKnown"`
	TCGURL      string  `json:"tcgUrl,omitempty"`
	ValueBRL    float64 `json:"valueBRL"`
	ProfitBRL   float64 `json:"profitBRL"`
	MarginPct   float64 `json:"marginPct"`
	Realized    bool    `json:"realized"`
}

type Summary struct {
	TargetPct float64 `json:"targetPct"`
	FXRate    float64 `json:"fxRate"`

	Holdings      int     `json:"holdings"`
	InvestedBRL   float64 `json:"investedBRL"`
	MarketBRL     float64 `json:"marketBRL"`
	UnrealizedBRL float64 `json:"unrealizedBRL"`

	Sold          int     `json:"sold"`
	CostOfSoldBRL float64 `json:"costOfSoldBRL"`
	ProceedsBRL   float64 `json:"proceedsBRL"`
	RealizedBRL   float64 `json:"realizedBRL"`

	TotalPnLBRL float64 `json:"totalPnLBRL"`
}

type PortfolioResponse struct {
	TargetPct float64     `json:"targetPct"`
	FXRate    float64     `json:"fxRate"`
	Summary   Summary     `json:"summary"`
	Trades    []TradeView `json:"trades"`
}
