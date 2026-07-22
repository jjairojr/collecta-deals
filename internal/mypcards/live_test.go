package mypcards

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"opdeals/internal/game"
	"opdeals/internal/logx"
)

// TestLiveOneSet exercises the full live path (headless Chrome → Cloudflare
// challenge → paginated scrape → parse) against one real edition. It is skipped
// unless MYP_LIVE=1 so the normal suite stays offline and Chrome-free.
func TestLiveOneSet(t *testing.T) {
	if os.Getenv("MYP_LIVE") != "1" {
		t.Skip("set MYP_LIVE=1 to run the live MyP Cards integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	c := New(logx.New(os.Stderr), 2, game.OnePiece())
	b := newBrowser(ctx)
	defer b.close()

	if err := b.solve(ctx); err != nil {
		t.Fatalf("cloudflare challenge not cleared: %v", err)
	}

	items := c.setListings(ctx, b, "the-time-of-battle")
	if len(items) == 0 {
		t.Fatal("no listings scraped from the-time-of-battle")
	}
	t.Logf("scraped %d singles from the-time-of-battle", len(items))

	op16, inStock := 0, 0
	for _, it := range items {
		if strings.HasPrefix(it.listing.Number, "OP16-") {
			op16++
		}
		if it.listing.InStock {
			inStock++
		}
		if it.listing.LowBRL <= 0 {
			t.Errorf("%s: non-positive price", it.listing.Number)
		}
	}
	if op16 == 0 {
		t.Errorf("expected OP16-* numbers, got none (sample: %+v)", items[0].listing)
	}
	if inStock == 0 {
		t.Error("expected in-stock listings, got none")
	}
	t.Logf("OP16 singles: %d, in-stock: %d/%d", op16, inStock, len(items))
}

// TestLiveRiftbound covers the second game's layout: a different URL slug, a
// different data-ga-item-id prefix, and the SPF→SFD set alias that set-scoped
// matching depends on. Spiritforged is used precisely because it is the aliased
// set — a regression there silently drops every deal in it.
func TestLiveRiftbound(t *testing.T) {
	if os.Getenv("MYP_LIVE") != "1" {
		t.Skip("set MYP_LIVE=1 to run the live MyP Cards integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	c := New(logx.New(os.Stderr), 2, game.Riftbound())
	b := newBrowser(ctx)
	defer b.close()

	if err := b.solve(ctx); err != nil {
		t.Fatalf("cloudflare challenge not cleared: %v", err)
	}

	editions, err := c.editions(ctx, b)
	if err != nil {
		t.Fatalf("editions: %v", err)
	}
	if len(editions) < 5 {
		t.Fatalf("expected several riftbound editions, got %v", editions)
	}
	t.Logf("editions: %v", editions)

	items := c.setListings(ctx, b, "spiritforged")
	if len(items) == 0 {
		t.Fatal("no listings scraped from spiritforged")
	}

	inStock := 0
	for _, it := range items {
		l := it.listing
		if l.SetCode != "SFD" {
			t.Errorf("%s: setcode = %q, want SFD (SPF alias)", l.Number, l.SetCode)
		}
		if l.LowBRL <= 0 {
			t.Errorf("%s: non-positive price", l.Number)
		}
		if !strings.Contains(l.Number, "/") {
			t.Errorf("%s: expected denominator to survive for the matcher", l.Number)
		}
		if l.InStock {
			inStock++
		}
	}
	t.Logf("spiritforged: %d singles, %d in stock, sample %+v", len(items), inStock, items[0].listing)
}
