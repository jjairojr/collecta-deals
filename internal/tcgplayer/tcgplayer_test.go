package tcgplayer

import "testing"

func mk(price float64, cond string, gold bool, lang, title string) listing {
	return mkq(price, 1, cond, gold, lang, title)
}

func mkq(price float64, qty int, cond string, gold bool, lang, title string) listing {
	l := listing{Price: price, Quantity: float64(qty), Condition: cond, GoldSeller: gold, Language: lang}
	l.CustomData.Title = title
	return l
}

func TestPickPriceSkipsNonGoldAndJapanese(t *testing.T) {
	// Mirrors the real Dracule Mihawk OP12-030 (SP) listings: the cheapest rows
	// are a non-gold Japanese custom listing and non-gold English ones; the
	// first gold star English Near Mint sits higher and is what we must pick.
	listings := []listing{
		mk(120.00, "Near Mint", false, "English", "Dracule Mihawk - OP12-030 (SP) JAPANESE"),
		mk(134.99, "Lightly Played", false, "English", ""),
		mk(141.74, "Lightly Played", true, "English", ""),
		mk(140.99, "Near Mint", true, "English", ""),
		mk(143.99, "Near Mint", true, "English", ""),
	}
	if got := pickPrice(listings); got.Price != 140.99 {
		t.Fatalf("pickPrice = %.2f, want 140.99 (gold star English NM)", got.Price)
	}
}

func TestPickPricePrefersNearMintAmongGold(t *testing.T) {
	// Chippys/Super Nova/Mystic/Pandys shape: only two gold star sellers, one LP
	// and one NM. NM must win even though it is priced higher.
	listings := []listing{
		mk(70.00, "Near Mint", false, "English", "Kid and Killer SP EB01-003 (JAPANESE)"),
		mk(85.00, "Near Mint", false, "English", ""),
		mk(88.25, "Lightly Played", true, "English", ""),
		mk(91.31, "Near Mint", true, "English", ""),
	}
	if got := pickPrice(listings); got.Price != 91.31 {
		t.Fatalf("pickPrice = %.2f, want 91.31 (gold NM over gold LP)", got.Price)
	}
}

func TestPickPriceFallsBackToGoldNonNM(t *testing.T) {
	listings := []listing{
		mk(50.00, "Lightly Played", false, "English", ""),
		mk(60.00, "Lightly Played", true, "English", ""),
	}
	if got := pickPrice(listings); got.Price != 60.00 {
		t.Fatalf("pickPrice = %.2f, want 60.00 (only gold is LP)", got.Price)
	}
}

func TestPickPriceNoGoldSellerReturnsZero(t *testing.T) {
	listings := []listing{
		mk(40.00, "Near Mint", false, "English", ""),
		mk(45.00, "Near Mint", false, "English", ""),
	}
	if got := pickPrice(listings); got.Price != 0 {
		t.Fatalf("pickPrice = %.2f, want 0 (no gold seller)", got.Price)
	}
}

// Depth counts the tier the price came from: when any NM exists, Listings/Qty are
// the gold-star English NM sellers only, ignoring LP and Japanese/non-gold rows.
func TestPickPriceDepthCountsNMTier(t *testing.T) {
	listings := []listing{
		mkq(120.00, 5, "Near Mint", false, "English", "OP12-030 JAPANESE"), // non-gold JP: ignored
		mkq(134.99, 4, "Lightly Played", false, "English", ""),             // non-gold: ignored
		mkq(141.74, 3, "Lightly Played", true, "English", ""),             // gold LP: not NM tier
		mkq(140.99, 2, "Near Mint", true, "English", ""),                  // gold NM
		mkq(143.99, 6, "Near Mint", true, "English", ""),                  // gold NM
	}
	got := pickPrice(listings)
	if got.Price != 140.99 {
		t.Fatalf("Price = %.2f, want 140.99", got.Price)
	}
	if got.Listings != 2 {
		t.Errorf("Listings = %d, want 2 (two gold-star NM sellers)", got.Listings)
	}
	if got.Qty != 8 {
		t.Errorf("Qty = %d, want 8 (2+6 across the NM sellers)", got.Qty)
	}
}

// When no NM listing exists, depth describes the overall gold-star tier the
// fallback price came from.
func TestPickPriceDepthFallbackTier(t *testing.T) {
	listings := []listing{
		mkq(50.00, 9, "Lightly Played", false, "English", ""), // non-gold: ignored
		mkq(60.00, 3, "Lightly Played", true, "English", ""),  // gold LP
		mkq(70.00, 4, "Moderately Played", true, "English", ""), // gold MP
	}
	got := pickPrice(listings)
	if got.Price != 60.00 {
		t.Fatalf("Price = %.2f, want 60.00", got.Price)
	}
	if got.Listings != 2 || got.Qty != 7 {
		t.Errorf("depth = %dL/%dq, want 2L/7q (both gold non-NM)", got.Listings, got.Qty)
	}
}

func TestIsForeignLanguageByLanguageField(t *testing.T) {
	if !isForeignLanguage(mk(10, "Near Mint", true, "Japanese", "")) {
		t.Error("language=Japanese should be flagged")
	}
	if isForeignLanguage(mk(10, "Near Mint", true, "English", "Dracule Mihawk (SP)")) {
		t.Error("plain English listing should not be flagged")
	}
}

// Real Edward.Newgate (SP) Royal Blood OP10 (product 617176) shape: a Japanese
// print filed under the English product with language "English" and tagged with
// the abbreviation "**JPN**" undercut the real English floor by ~40%. It must be
// skipped so the trusted price is the English NM listing ($367.98), not $224.99.
func TestPickPriceSkipsJPNAbbrevFiledAsEnglish(t *testing.T) {
	listings := []listing{
		mk(224.99, "Near Mint", true, "English", "**JPN** Edward.Newgate (SP) - Royal Blood (OP10) **JPN**"),
		mk(367.98, "Near Mint", true, "English", ""),
		mk(367.99, "Near Mint", true, "English", ""),
	}
	if got := pickPrice(listings); got.Price != 367.98 {
		t.Fatalf("pickPrice = %.2f, want 367.98 (JPN row skipped)", got.Price)
	}
}

// Real Lee Sin Blind Monk (Overnumbered) 304/298 shape: Chinese prints filed under
// the English product with language "English" undercut the genuine English floor.
// They must be skipped so the trusted price is the real English NM listing.
func TestPickPriceSkipsChineseFiledAsEnglish(t *testing.T) {
	listings := []listing{
		mk(38.01, "Near Mint", true, "English", "Lee Sin Blind Monk (Overnumbered) **Chinese**"),
		mk(39.99, "Near Mint", true, "English", "Lee Sin Blink Monk (Chinese) Overnumber"),
		mk(65.00, "Near Mint", true, "English", ""),
	}
	if got := pickPrice(listings); got.Price != 65.00 {
		t.Fatalf("pickPrice = %.2f, want 65.00 (Chinese rows skipped)", got.Price)
	}
}
