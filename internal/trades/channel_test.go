package trades

import "testing"

func TestChannelOfResolvesLegacyBuyers(t *testing.T) {
	cases := []struct {
		buyer string
		want  string
	}{
		{"Guilherme de Araujo — Liga #11641862", ChannelLiga},
		{"Carlos Eduardo — Liga", ChannelLiga},
		{"Liga", ChannelLiga},
		{"zapiti — MyP #812", ChannelMyP},
		{"David Candelero — Site #12", ChannelSite},
		{"site", ChannelSite},
		{"Site", ChannelSite},
		{"Balcão", ChannelBalcao},
		{"balcao", ChannelBalcao},
		{"MYP", ChannelBalcao},
		{"MYP Cards", ChannelBalcao},
		{"Venda por fora", ChannelBalcao},
		{"Abertos em Live", ChannelBalcao},
		{"Case Aberta", ChannelBalcao},
		{"Mercado Livre", ChannelOutros},
		{"shopee", ChannelOutros},
		{"Allen", ChannelOutros},
		{"", ChannelOutros},
		{"----", ChannelOutros},
	}
	for _, c := range cases {
		got := ChannelOf(Trade{Status: "sold", Buyer: c.buyer})
		if got != c.want {
			t.Errorf("ChannelOf(%q) = %q, want %q", c.buyer, got, c.want)
		}
	}
}

func TestChannelOfKeepsExplicitChannel(t *testing.T) {
	got := ChannelOf(Trade{Status: "sold", Channel: ChannelMyP, Buyer: "Fulano — Liga #1"})
	if got != ChannelMyP {
		t.Fatalf("explicit channel overridden: %q", got)
	}
}

func TestWriteOffOfResolvesOpenedCases(t *testing.T) {
	cases := []struct {
		t    Trade
		want string
	}{
		{Trade{Buyer: "Abertos em Live", SellPrice: 0.02}, WriteOffAberto},
		{Trade{Buyer: "Case Aberta", SellPrice: 1}, WriteOffAberto},
		{Trade{Buyer: "Guilherme — Liga #1", SellPrice: 0.27}, ""},
		{Trade{Buyer: "Balcão", SellPrice: 0, WriteOff: WriteOffBrinde}, WriteOffBrinde},
	}
	for _, c := range cases {
		c.t.Status = "sold"
		if got := WriteOffOf(c.t); got != c.want {
			t.Errorf("WriteOffOf(%q) = %q, want %q", c.t.Buyer, got, c.want)
		}
	}
}

func TestValidChannelAndWriteOff(t *testing.T) {
	for _, c := range []string{"", ChannelLiga, ChannelMyP, ChannelSite, ChannelBalcao, ChannelOutros} {
		if !ValidChannel(c) {
			t.Errorf("ValidChannel(%q) = false", c)
		}
	}
	if ValidChannel("shopee") {
		t.Error("ValidChannel(shopee) = true")
	}
	for _, w := range []string{"", WriteOffAberto, WriteOffPerda, WriteOffBrinde, WriteOffUso} {
		if !ValidWriteOff(w) {
			t.Errorf("ValidWriteOff(%q) = false", w)
		}
	}
	if ValidWriteOff("roubo") {
		t.Error("ValidWriteOff(roubo) = true")
	}
}

func TestPortfolioResolvesChannelOfSoldOnly(t *testing.T) {
	p := BuildPortfolio([]Trade{
		{ID: "a", Name: "x", Qty: 1, Status: "sold", Buyer: "Fulano — MyP #3", SellPrice: 10},
		{ID: "b", Name: "y", Qty: 1, Status: "holding"},
	}, 90, 5, nil)
	if p.Trades[0].Channel != ChannelMyP {
		t.Fatalf("sold channel = %q, want myp", p.Trades[0].Channel)
	}
	if p.Trades[1].Channel != "" {
		t.Fatalf("holding channel = %q, want empty", p.Trades[1].Channel)
	}
}

func TestWriteOffCostsButIsNotASale(t *testing.T) {
	p := BuildPortfolio([]Trade{
		{ID: "a", Name: "Box", Kind: "sealed", Qty: 1, BuyBRL: 300, Status: "sold", WriteOff: WriteOffAberto},
		{ID: "b", Name: "Box", Kind: "sealed", Qty: 1, BuyBRL: 300, Status: "sold", SellPrice: 400, Channel: ChannelLiga},
	}, 90, 5, nil)
	if p.Summary.Sold != 1 {
		t.Fatalf("sold = %d, want 1 (write-off is not a sale)", p.Summary.Sold)
	}
	if p.Summary.CostOfSoldBRL != 600 || p.Summary.ProceedsBRL != 400 {
		t.Fatalf("cost %v proceeds %v, want 600 and 400", p.Summary.CostOfSoldBRL, p.Summary.ProceedsBRL)
	}
}
