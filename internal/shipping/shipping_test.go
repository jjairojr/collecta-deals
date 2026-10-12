package shipping

import (
	"encoding/json"
	"math"
	"testing"
)

func TestPack(t *testing.T) {
	got := Pack([]Item{
		{Kind: "", Name: "Luffy", Qty: 2, UnitBRL: 50},
		{Kind: "", Name: "Zoro", Qty: 1, UnitBRL: 20},
		{Kind: "sealed", Name: "(ING) Booster Box - OP12", Qty: 2, UnitBRL: 800},
		{Kind: "sealed", Name: "Deck Inicial ST-21", Qty: 1, UnitBRL: 90},
		{Kind: "accessory", Name: "Playmat Nami", Qty: 1, UnitBRL: 120},
	})
	if len(got) != 4 {
		t.Fatalf("products = %+v", got)
	}
	if got[0].ID != "sealed-2" || got[0].Weight != 0.9 || got[0].Quantity != 2 || got[0].InsuranceValue != 800 {
		t.Fatalf("booster box = %+v", got[0])
	}
	if got[1].ID != "sealed-3" || got[1].Length != 4 {
		t.Fatalf("deck = %+v", got[1])
	}
	if got[2].ID != "accessory-4" || got[2].Height != 36 {
		t.Fatalf("playmat = %+v", got[2])
	}
	env := got[3]
	if env.ID != "singles" || env.Quantity != 1 || env.InsuranceValue != 120 || math.Abs(env.Weight-0.08) > 1e-9 {
		t.Fatalf("singles envelope = %+v", env)
	}
}

func TestPackNoSingles(t *testing.T) {
	got := Pack([]Item{{Kind: "sealed", Name: "Mystery", Qty: 1, UnitBRL: 10}})
	if len(got) != 1 || got[0].Width != sealedDefault.width {
		t.Fatalf("products = %+v", got)
	}
}

func TestParseServices(t *testing.T) {
	raw := `[
	 {"id":1,"name":"PAC","price":"21.50","custom_price":"19.90","delivery_time":8,"company":{"name":"Correios"}},
	 {"id":2,"name":"SEDEX","price":"35.10","custom_price":"35.10","delivery_time":3,"company":{"name":"Correios"}},
	 {"id":17,"name":"Mini Envios","price":"12.00","delivery_time":9,"company":{"name":"Correios"}},
	 {"id":3,"name":".Package","error":"Dimensões excedem o limite","company":{"name":"Jadlog"}}
	]`
	var rows []calculateResponse
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		t.Fatal(err)
	}
	got := parseServices(rows)
	if len(got) != 3 {
		t.Fatalf("services = %+v", got)
	}
	if got[0].ID != 17 || got[0].PriceBRL != 12 || got[1].ID != 1 || got[1].PriceBRL != 19.9 || got[2].ID != 2 {
		t.Fatalf("order/prices = %+v", got)
	}
	if got[1].Company != "Correios" || got[1].DeliveryDays != 8 {
		t.Fatalf("pac = %+v", got[1])
	}
}
