package trades

type PriceLookup func(number, name, set string) (usd float64, url string, ok bool)

func BuildPortfolio(all []Trade, pct, fxRate float64, lookup PriceLookup) PortfolioResponse {
	resp := PortfolioResponse{
		TargetPct: pct,
		FXRate:    fxRate,
		Trades:    make([]TradeView, 0, len(all)),
	}
	sum := Summary{TargetPct: pct, FXRate: fxRate}
	for _, t := range all {
		v := value(t, pct, fxRate, lookup)
		resp.Trades = append(resp.Trades, v)
		if v.Realized {
			sum.Sold++
			sum.CostOfSoldBRL += v.CostBRL
			sum.ProceedsBRL += v.ValueBRL
		} else {
			sum.Holdings++
			sum.InvestedBRL += v.CostBRL
			sum.MarketBRL += v.ValueBRL
		}
	}
	sum.UnrealizedBRL = sum.MarketBRL - sum.InvestedBRL
	sum.RealizedBRL = sum.ProceedsBRL - sum.CostOfSoldBRL
	sum.TotalPnLBRL = sum.RealizedBRL + sum.UnrealizedBRL
	resp.Summary = sum
	return resp
}

func value(t Trade, pct, fxRate float64, lookup PriceLookup) TradeView {
	v := TradeView{Trade: t}
	if v.Qty <= 0 {
		v.Qty = 1
	}
	v.CostBRL = float64(v.Qty)*t.BuyBRL + t.ShippingBRL

	market, ok := 0.0, false
	if t.Kind != "sealed" {
		if lookup != nil {
			market, v.TCGURL, ok = lookup(t.Number, t.Name, t.Set)
		}
		if !ok && t.RefUSD > 0 {
			market, ok = t.RefUSD, true
		}
	}
	v.MarketUSD = market
	v.MarketKnown = ok

	if t.Status == "sold" {
		v.Realized = true
		v.ValueBRL = proceedsBRL(t, fxRate)
	} else if t.Kind == "sealed" {
		v.ValueBRL = t.ManualBRL * float64(v.Qty)
	} else if ok && fxRate > 0 {
		v.ValueBRL = pct / 100 * market * float64(v.Qty) / fxRate
	}
	v.ProfitBRL = v.ValueBRL - v.CostBRL
	if v.CostBRL > 0 {
		v.MarginPct = v.ProfitBRL / v.CostBRL * 100
	}
	return v
}

func proceedsBRL(t Trade, fxRate float64) float64 {
	if t.SellCurrency == "USD" {
		if fxRate <= 0 {
			return 0
		}
		return t.SellPrice / fxRate
	}
	return t.SellPrice
}
