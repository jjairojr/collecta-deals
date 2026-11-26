package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"opdeals/internal/orders"
	"opdeals/internal/packs"
	"opdeals/internal/syncqueue"
	"opdeals/internal/trades"
)

const (
	linkCaseSKU = "sealed|onepiece|case op18|||"
	linkBoxSKU  = "sealed|onepiece|box op18|||"
)

type packFixture struct {
	*kitFixture
	cases trades.Trade
	boxes trades.Trade
}

func packServer(t *testing.T, looseBoxes int) *packFixture {
	t.Helper()
	k := &kitFixture{checkoutFixture: checkoutServer(t)}
	k.s.Packs = packs.NewStore(filepath.Join(t.TempDir(), "packs.json"))
	k.s.SyncQueue = syncqueue.NewStore(filepath.Join(t.TempDir(), "syncqueue.json"))
	cases, err := k.store.Add(trades.Trade{Kind: "sealed", Name: "Case OP18", Set: "SEALED", Qty: 2, BuyBRL: 6192, BuyDate: "2026-10-07", Status: "holding"})
	if err != nil {
		t.Fatal(err)
	}
	boxes, err := k.store.Add(trades.Trade{Kind: "sealed", Name: "Box OP18", Set: "SEALED", Qty: looseBoxes, BuyBRL: 516, BuyDate: "2026-10-07", Status: "holding"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.store.SetListings(map[string]trades.Listing{
		cases.ID: {AskBRL: 8000, Listed: true},
		boxes.ID: {AskBRL: 700, Listed: true},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := k.store.SetLigaState(map[string]trades.LigaState{
		cases.ID: {Listed: true, Qty: 2, PriceBRL: 8000},
		boxes.ID: {Listed: true, Qty: looseBoxes + 24, PriceBRL: 700},
	}); err != nil {
		t.Fatal(err)
	}
	p := &packFixture{kitFixture: k, cases: cases, boxes: boxes}
	rec := p.do(t, http.MethodPost, "/api/packs", map[string]interface{}{"parentTradeID": cases.ID, "childTradeID": boxes.ID, "units": 12}, adminHeaders)
	if rec.Code != http.StatusOK {
		t.Fatalf("create link: status %d body %s", rec.Code, rec.Body.String())
	}
	return p
}

func (p *packFixture) held(t *testing.T, sku string) int {
	t.Helper()
	all, err := p.store.List()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, tr := range all {
		if tr.Status == "holding" && trades.SKU("onepiece", tr) == sku {
			n += tr.Qty
		}
	}
	return n
}

func (p *packFixture) shelf(t *testing.T, sku string) int {
	t.Helper()
	n := 0
	for _, it := range p.storefront(t).Items {
		if it.SKU == sku {
			n += it.Qty
		}
	}
	return n
}

func (p *packFixture) pending(t *testing.T) map[string]int {
	t.Helper()
	all, err := p.s.SyncQueue.List(syncqueue.StatusPending)
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]int)
	for _, task := range all {
		qty := task.Qty
		if task.Action == syncqueue.ActionRemove {
			qty = -qty
		}
		out[task.Name] += qty
	}
	return out
}

func TestPackLinkValidation(t *testing.T) {
	p := packServer(t, 1)
	rec := p.do(t, http.MethodPost, "/api/packs", map[string]interface{}{"parentTradeID": p.cases.ID, "childTradeID": p.older.ID, "units": 3}, adminHeaders)
	if rec.Code != http.StatusConflict {
		t.Fatalf("second content for the same pack: status %d", rec.Code)
	}
	rec = p.do(t, http.MethodPost, "/api/packs", map[string]interface{}{"parentTradeID": "nope", "childTradeID": p.boxes.ID, "units": 3}, adminHeaders)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown pack: status %d", rec.Code)
	}
	rec = p.do(t, http.MethodGet, "/api/packs", nil, adminHeaders)
	var list struct {
		Packs []packView `json:"packs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Packs) != 1 {
		t.Fatalf("packs = %+v", list.Packs)
	}
	v := list.Packs[0]
	if v.ParentSKU != linkCaseSKU || v.ChildSKU != linkBoxSKU || v.ParentQty != 2 || v.ChildQty != 1 || v.ParentName != "Case OP18" || v.ChildName != "Box OP18" {
		t.Fatalf("view = %+v", v)
	}
	if rec := p.do(t, http.MethodDelete, "/api/packs/"+v.ID, nil, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("delete: status %d", rec.Code)
	}
	if got := p.shelf(t, linkBoxSKU); got != 1 {
		t.Fatalf("without the link only the loose box is for sale: %d", got)
	}
}

func TestStorefrontCountsBoxesInsideClosedCases(t *testing.T) {
	p := packServer(t, 1)
	if got := p.shelf(t, linkBoxSKU); got != 25 {
		t.Fatalf("boxes on the shelf = %d want 25", got)
	}
	if got := p.shelf(t, linkCaseSKU); got != 2 {
		t.Fatalf("cases on the shelf = %d want 2", got)
	}
	p.placeOrder(t, kitOrderBody(0, orderLine(linkBoxSKU, 3, 70000)))
	if box, cs := p.shelf(t, linkBoxSKU), p.shelf(t, linkCaseSKU); box != 22 || cs != 1 {
		t.Fatalf("3 held boxes need one case: boxes %d cases %d", box, cs)
	}
	rec := p.do(t, http.MethodPost, "/api/orders", kitOrderBody(1, orderLine(linkCaseSKU, 2, 800000)), storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"available":1`) {
		t.Fatalf("second buyer wants both cases: status %d body %s", rec.Code, rec.Body.String())
	}
	p.placeOrder(t, kitOrderBody(1, orderLine(linkCaseSKU, 1, 800000)))
	if box, cs := p.shelf(t, linkBoxSKU), p.shelf(t, linkCaseSKU); box != 10 || cs != 0 {
		t.Fatalf("a held case takes its boxes: boxes %d cases %d", box, cs)
	}
}

func TestPaidBoxOpensACaseOnlyWhenLooseRunsOut(t *testing.T) {
	p := packServer(t, 1)
	got := p.pay(t, p.placeOrder(t, kitOrderBody(0, orderLine(linkBoxSKU, 1, 70000))))
	if got.Status != orders.StatusPaid {
		t.Fatalf("order = %s notes %q", got.Status, got.Notes)
	}
	if cs, box := p.held(t, linkCaseSKU), p.held(t, linkBoxSKU); cs != 2 || box != 0 {
		t.Fatalf("the loose box goes first: cases %d boxes %d", cs, box)
	}
	if q := p.pending(t); q["Box OP18"] != -1 || q["Case OP18"] != 0 {
		t.Fatalf("queue = %+v", q)
	}
}

func TestPaidBoxWithoutLooseStockOpensOneCase(t *testing.T) {
	p := packServer(t, 1)
	got := p.pay(t, p.placeOrder(t, kitOrderBody(0, orderLine(linkBoxSKU, 3, 70000))))
	if got.Status != orders.StatusPaid {
		t.Fatalf("order = %s notes %q", got.Status, got.Notes)
	}
	if cs, box := p.held(t, linkCaseSKU), p.held(t, linkBoxSKU); cs != 1 || box != 10 {
		t.Fatalf("cases %d boxes %d, want 1 and 10", cs, box)
	}
	all, _ := p.store.List()
	cost, sold := 0.0, 0
	for _, tr := range all {
		if tr.Kind != "sealed" {
			continue
		}
		cost += float64(tr.Qty) * tr.BuyBRL
		if tr.Status == "sold" {
			sold += tr.Qty
			if tr.BuyBRL != 516 || tr.Channel != trades.ChannelSite {
				t.Fatalf("sold box = %+v", tr)
			}
		}
	}
	if sold != 3 || round2(cost) != 2*6192+516 {
		t.Fatalf("sold %d cost %v", sold, cost)
	}
	if q := p.pending(t); q["Box OP18"] != -3 || q["Case OP18"] != -1 {
		t.Fatalf("queue = %+v", q)
	}
	if got := p.shelf(t, linkBoxSKU); got != 22 {
		t.Fatalf("boxes on the shelf = %d want 22", got)
	}
}

func TestSellingACaseTakesItsBoxesOffTheLiga(t *testing.T) {
	p := packServer(t, 1)
	rec := p.do(t, http.MethodPost, "/api/trades/"+p.cases.ID+"/sell", map[string]interface{}{"qty": 1, "sellPrice": 8000, "sellCurrency": "BRL", "buyer": "Balcão", "channel": "balcao"}, adminHeaders)
	if rec.Code != http.StatusOK {
		t.Fatalf("sell: status %d body %s", rec.Code, rec.Body.String())
	}
	var sold trades.Trade
	if err := json.Unmarshal(rec.Body.Bytes(), &sold); err != nil {
		t.Fatal(err)
	}
	if q := p.pending(t); q["Case OP18"] != -1 || q["Box OP18"] != -12 {
		t.Fatalf("queue = %+v", q)
	}
	if rec := p.do(t, http.MethodPost, "/api/trades/"+sold.ID+"/unsell", nil, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("unsell: status %d body %s", rec.Code, rec.Body.String())
	}
	if q := p.pending(t); len(q) != 0 {
		t.Fatalf("undone sale must clear the queue: %+v", q)
	}
}

func TestOpenEndpointCreatesContentAndLink(t *testing.T) {
	p := packServer(t, 1)
	rec := p.do(t, http.MethodPost, "/api/trades/"+p.cases.ID+"/open", map[string]interface{}{"qty": 1}, adminHeaders)
	if rec.Code != http.StatusOK {
		t.Fatalf("open: status %d body %s", rec.Code, rec.Body.String())
	}
	if cs, box := p.held(t, linkCaseSKU), p.held(t, linkBoxSKU); cs != 1 || box != 13 {
		t.Fatalf("cases %d boxes %d", cs, box)
	}
	if q := p.pending(t); q["Case OP18"] != -1 || q["Box OP18"] != 0 {
		t.Fatalf("opening keeps the boxes on the Liga: %+v", q)
	}
	if rec := p.do(t, http.MethodPost, "/api/trades/"+p.boxes.ID+"/open", map[string]interface{}{"qty": 1}, adminHeaders); rec.Code != http.StatusBadRequest {
		t.Fatalf("box without a link: status %d", rec.Code)
	}
	rec = p.do(t, http.MethodPost, "/api/trades/"+p.boxes.ID+"/open", map[string]interface{}{"qty": 2, "units": 24, "childName": "Booster OP18"}, adminHeaders)
	if rec.Code != http.StatusOK {
		t.Fatalf("open into a new product: status %d body %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Child trades.Trade `json:"child"`
		Link  packs.Link   `json:"link"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Child.Name != "Booster OP18" || out.Child.Qty != 48 || out.Child.BuyBRL != 21.5 || out.Child.Listed || out.Child.LigaListed || out.Link.Units != 24 {
		t.Fatalf("child = %+v link = %+v", out.Child, out.Link)
	}
	if got := p.held(t, linkBoxSKU); got != 11 {
		t.Fatalf("boxes = %d", got)
	}
	if got := p.shelf(t, linkBoxSKU); got != 11+12 {
		t.Fatalf("boxes on the shelf = %d", got)
	}
}
