package postsale

import (
	"strings"
	"testing"
	"time"

	"opdeals/internal/conversations"
	"opdeals/internal/orders"
)

var sp = func() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		panic(err)
	}
	return loc
}()

func TestCandidatesPriceBandAndEnvelope(t *testing.T) {
	bought := []orders.Line{{SKU: "a", Game: "onepiece", Name: "Nami", Meta: "OP12 · OP12-001", Qty: 1, UnitBRL: 100}}
	stock := []Item{
		{SKU: "a", Game: "onepiece", Name: "Nami", Set: "OP12", UnitBRL: 50, Available: 1},
		{SKU: "cheap", Game: "onepiece", Name: "X", UnitBRL: 9, Available: 1},
		{SKU: "below", Game: "onepiece", Name: "Y", UnitBRL: 29, Available: 1},
		{SKU: "above", Game: "onepiece", Name: "Z", UnitBRL: 101, Available: 1},
		{SKU: "box", Game: "onepiece", Kind: "sealed", Name: "Booster Box", UnitBRL: 60, Available: 1},
		{SKU: "zero", Game: "onepiece", Name: "W", UnitBRL: 60, Available: 0},
		{SKU: "ok", Game: "onepiece", Name: "Robin", UnitBRL: 30, Available: 1},
	}
	got := Candidates(bought, 100, stock, nil)
	if len(got) != 1 || got[0].SKU != "ok" {
		t.Fatalf("got %+v", got)
	}
}

func TestCandidatesMinimumTenReais(t *testing.T) {
	bought := []orders.Line{{SKU: "a", Game: "pokemon", Name: "Pikachu", UnitBRL: 20, Qty: 1}}
	stock := []Item{
		{SKU: "nine", Game: "pokemon", Name: "B", UnitBRL: 9.99, Available: 1},
		{SKU: "ten", Game: "pokemon", Name: "C", UnitBRL: 10, Available: 1},
	}
	got := Candidates(bought, 20, stock, nil)
	if len(got) != 1 || got[0].SKU != "ten" {
		t.Fatalf("got %+v", got)
	}
}

func TestCandidatesPriorityRelatedThenVelocityThenFeatured(t *testing.T) {
	bought := []orders.Line{{SKU: "b", Game: "onepiece", Name: "Roronoa Zoro", Meta: "OP06 · OP06-118", UnitBRL: 200, Qty: 1}}
	stock := []Item{
		{SKU: "featured", Game: "onepiece", Name: "Luffy", Set: "OP01", UnitBRL: 100, Available: 1, Featured: true},
		{SKU: "fast", Game: "onepiece", Name: "Sanji", Set: "OP02", UnitBRL: 100, Available: 1},
		{SKU: "other-game", Game: "pokemon", Name: "Zoro Pokemon", UnitBRL: 100, Available: 1},
		{SKU: "same-set", Game: "onepiece", Name: "Kaido", Set: "OP06", UnitBRL: 100, Available: 1},
		{SKU: "same-char", Game: "onepiece", Name: "Roronoa Zoro (Parallel)", Set: "OP01", UnitBRL: 100, Available: 1},
	}
	got := Candidates(bought, 200, stock, map[string]float64{"fast": 5})
	order := []string{"same-char", "same-set", "fast", "featured"}
	if len(got) != MaxCandidates {
		t.Fatalf("len %d", len(got))
	}
	for i, sku := range order {
		if got[i].SKU != sku {
			t.Fatalf("pos %d = %s, want %s (all %+v)", i, got[i].SKU, sku, got)
		}
	}
	if got[0].Reasons[0] != "combina com Roronoa Zoro" {
		t.Fatalf("reasons %v", got[0].Reasons)
	}
}

func TestQuietHoursAndNextAllowed(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 10, 31, h, m, 0, 0, sp) }
	if Quiet(at(21, 59), sp) || !Quiet(at(22, 0), sp) || !Quiet(at(7, 59), sp) || Quiet(at(8, 0), sp) {
		t.Fatal("quiet boundaries wrong")
	}
	if got := NextAllowed(at(23, 10), sp); !got.Equal(time.Date(2026, 11, 1, 8, 0, 0, 0, sp)) {
		t.Fatalf("23h → %v", got)
	}
	if got := NextAllowed(at(3, 0), sp); !got.Equal(at(8, 0)) {
		t.Fatalf("3h → %v", got)
	}
	if got := NextAllowed(at(14, 0), sp); !got.Equal(at(14, 0)) {
		t.Fatalf("14h → %v", got)
	}
}

func TestEligibleForOffer(t *testing.T) {
	now := time.Date(2026, 10, 31, 14, 0, 0, 0, sp)
	base := orders.Order{
		ID:             "o1",
		Status:         orders.StatusPaid,
		PaidAt:         now.Add(-time.Hour),
		MarketingOptIn: true,
		Buyer:          orders.Buyer{Phone: "11987654321"},
		Shipping:       orders.Shipping{ServiceID: 1},
	}
	if !EligibleForOffer(base, nil, now) {
		t.Fatal("base should be eligible")
	}
	noOpt := base
	noOpt.MarketingOptIn = false
	pickup := base
	pickup.Shipping.ServiceID = -1
	for name, o := range map[string]orders.Order{"noOpt": noOpt, "pickup": pickup} {
		if !EligibleForOffer(o, nil, now) {
			t.Errorf("%s should be eligible", name)
		}
	}
	join := base
	join.Shipping.ServiceID = -2
	fresh := base
	fresh.PaidAt = now.Add(-5 * time.Minute)
	old := base
	old.PaidAt = now.Add(-25 * time.Hour)
	shipped := base
	shipped.Status = orders.StatusShipped
	for name, o := range map[string]orders.Order{"join": join, "fresh": fresh, "old": old, "shipped": shipped} {
		if EligibleForOffer(o, nil, now) {
			t.Errorf("%s should not be eligible", name)
		}
	}
	recent := []conversations.Conversation{{OrderID: "other", Phone: "5511987654321", CreatedAt: now.Add(-6 * 24 * time.Hour)}}
	if EligibleForOffer(base, recent, now) {
		t.Error("cooldown ignored")
	}
	skipped := []conversations.Conversation{{OrderID: "other", Phone: "5511987654321", CreatedAt: now.Add(-time.Hour), Outcome: conversations.OutcomeSkipped}}
	if !EligibleForOffer(base, skipped, now) {
		t.Error("a skipped draft must not block the next offer")
	}
	same := []conversations.Conversation{{OrderID: "o1", Phone: "5521900000000", CreatedAt: now.Add(-30 * 24 * time.Hour)}}
	if EligibleForOffer(base, same, now) {
		t.Error("order already has a conversation")
	}
}

func TestEligibleLigaOrder(t *testing.T) {
	now := time.Date(2026, 11, 2, 14, 0, 0, 0, sp)
	if !EligibleLigaOrder("liga:1", "(62) 99827-4848", now.Add(-2*time.Hour), nil, now) {
		t.Fatal("fresh liga order should be eligible")
	}
	if EligibleLigaOrder("liga:1", "(62) 99827-4848", now.Add(-25*time.Hour), nil, now) {
		t.Error("paid more than 24h ago")
	}
	if EligibleLigaOrder("liga:1", "9982", now.Add(-time.Hour), nil, now) {
		t.Error("invalid phone")
	}
	recent := []conversations.Conversation{{OrderID: "site-1", Phone: "5562998274848", CreatedAt: now.Add(-2 * 24 * time.Hour)}}
	if EligibleLigaOrder("liga:1", "(62) 99827-4848", now.Add(-time.Hour), recent, now) {
		t.Error("cooldown ignored")
	}
	same := []conversations.Conversation{{OrderID: "liga:1", Phone: "5511900000000", CreatedAt: now.Add(-30 * 24 * time.Hour)}}
	if EligibleLigaOrder("liga:1", "(62) 99827-4848", now.Add(-time.Hour), same, now) {
		t.Error("liga order imported twice")
	}
}

func TestCleanOpeningKeepsLineBreaks(t *testing.T) {
	if got := CleanOpening("  Oi Ana!  Valeu 💙\r\n\r\nQuer  mais algo?  "); got != "Oi Ana! Valeu 💙\n\nQuer mais algo?" {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("a", 700)
	if got := []rune(CleanOpening(long)); len(got) != MaxOpeningRunes {
		t.Fatalf("len %d", len(got))
	}
}

func TestCleanPitch(t *testing.T) {
	long := ""
	for range 400 {
		long += "a"
	}
	if got := []rune(CleanPitch(long)); len(got) != MaxPitchRunes {
		t.Fatalf("len %d", len(got))
	}
	if got := CleanPitch("linha 1\nlinha 2"); got != "linha 1 linha 2" {
		t.Fatalf("got %q", got)
	}
}
