package report

import (
	"fmt"
	"io"
	"text/tabwriter"

	"opdeals/internal/model"
)

func Print(w io.Writer, deals []model.Deal, fxRate float64) {
	if len(deals) == 0 {
		fmt.Fprintln(w, "No deals found for the given filters.")
		return
	}
	fmt.Fprintf(w, "Found %d deals (BRL->USD rate %.4f)\n\n", len(deals), fxRate)
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "CARD\tNAME\tRARITY\tVARIANT\tSRC\tBR R$\tBUY US$\tSELL US$\tMARGIN%\tPROFIT US$\tUS DEPTH\tBR DEPTH\tTCGPLAYER\tBUY (BR)")
	for _, d := range deals {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%.2f\t%.2f\t%.2f\t%.1f\t%.2f\t%s\t%s\t%s\t%s\n",
			d.Number, truncate(d.Name, 28), d.Rarity, d.Variant, sourceLabel(d.Source),
			d.LowBRL, d.BuyUSD, d.SellUSD, d.MarginPct, d.ProfitUSD,
			usDepth(d.USListings, d.USQty), brDepth(d.BRCopies, d.BRSellers), d.TCGURL, d.BuyURL)
	}
	tw.Flush()
}

func sourceLabel(source string) string {
	switch source {
	case "ligaonepiece":
		return "Liga"
	case "mypcards":
		return "MYP"
	default:
		return source
	}
}

// usDepth renders the US sell-price trust: how many listings back the live floor
// and total copies. "—" when the card wasn't live-priced (no depth known).
func usDepth(listings, qty int) string {
	if listings <= 0 {
		return "—"
	}
	return fmt.Sprintf("%dL/%dq", listings, qty)
}

// brDepth renders the Brazil buy-side supply: copies at the floor and distinct
// sellers. "—" when stock wasn't verified.
func brDepth(copies, sellers int) string {
	if sellers <= 0 {
		return "—"
	}
	return fmt.Sprintf("%dc/%ds", copies, sellers)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
