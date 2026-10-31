package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"opdeals/internal/syncqueue"
	"opdeals/internal/trades"
)

func unsellTrade(t *testing.T, s *Server, id string) *httptest.ResponseRecorder {
	t.Helper()
	return queueCall(t, http.MethodPost, "/api/trades/"+id+"/unsell?game=onepiece", id, "", s.handleTradesUnsell)
}

func ledgerTrades(t *testing.T, st *trades.Store) map[string]trades.Trade {
	t.Helper()
	all, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]trades.Trade, len(all))
	for _, tr := range all {
		out[tr.ID] = tr
	}
	return out
}

func soldIDOther(t *testing.T, st *trades.Store, holdingID string) string {
	t.Helper()
	for id, tr := range ledgerTrades(t, st) {
		if id != holdingID && tr.Status == "sold" {
			return id
		}
	}
	t.Fatal("no sold lot found")
	return ""
}

func TestUnsellWholeLotReturnsToHoldingAndClearsSale(t *testing.T) {
	s, op := queueServer(t)
	item := createTrade(t, s, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":2,"buyBRL":100,"status":"holding"}`)
	if rec := sellTrade(t, s, item.ID, `{"sellPrice":300,"sellCurrency":"BRL","sellDate":"2026-10-30","buyer":"Fulano","channel":"balcao"}`); rec.Code != http.StatusOK {
		t.Fatalf("sell = %d", rec.Code)
	}
	if rec := unsellTrade(t, s, item.ID); rec.Code != http.StatusOK {
		t.Fatalf("unsell = %d: %s", rec.Code, rec.Body.String())
	}
	got := ledgerTrades(t, op)
	if len(got) != 1 {
		t.Fatalf("ledger = %+v", got)
	}
	tr := got[item.ID]
	if tr.Status != "holding" || tr.Qty != 2 || tr.SellPrice != 0 || tr.SellCurrency != "" || tr.SellDate != "" ||
		tr.Buyer != "" || tr.Channel != "" || tr.WriteOff != "" || tr.SemLastro {
		t.Fatalf("after unsell = %+v", tr)
	}
}

func TestUnsellSplitMergesBackIntoOrigin(t *testing.T) {
	s, op := queueServer(t)
	item := createTrade(t, s, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":4,"buyBRL":100,"shippingBRL":40,"status":"holding"}`)
	sellTrade(t, s, item.ID, `{"qty":1,"sellPrice":150,"channel":"balcao"}`)
	sold := soldIDOther(t, op, item.ID)
	if rec := unsellTrade(t, s, sold); rec.Code != http.StatusOK {
		t.Fatalf("unsell = %d: %s", rec.Code, rec.Body.String())
	}
	got := ledgerTrades(t, op)
	if len(got) != 1 {
		t.Fatalf("ledger = %+v, want origin only", got)
	}
	tr := got[item.ID]
	if tr.Status != "holding" || tr.Qty != 4 || tr.ShippingBRL != 40 || tr.BuyBRL != 100 {
		t.Fatalf("origin after unsell = %+v", tr)
	}
}

func TestUnsellWithoutOriginStaysAsOwnHolding(t *testing.T) {
	s, op := queueServer(t)
	item := createTrade(t, s, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":3,"buyBRL":100,"status":"holding"}`)
	sellTrade(t, s, item.ID, `{"qty":1,"sellPrice":150,"channel":"balcao"}`)
	sold := soldIDOther(t, op, item.ID)
	sellTrade(t, s, item.ID, `{"sellPrice":300,"channel":"balcao"}`)
	if rec := unsellTrade(t, s, sold); rec.Code != http.StatusOK {
		t.Fatalf("unsell = %d: %s", rec.Code, rec.Body.String())
	}
	got := ledgerTrades(t, op)
	if len(got) != 2 || got[sold].Status != "holding" || got[sold].Qty != 1 || got[item.ID].Status != "sold" {
		t.Fatalf("ledger = %+v", got)
	}
}

func TestUnsellRefusesSiteAndHolding(t *testing.T) {
	s, _ := queueServer(t)
	item := createTrade(t, s, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":2,"buyBRL":100,"status":"holding"}`)
	if rec := unsellTrade(t, s, item.ID); rec.Code != http.StatusConflict {
		t.Fatalf("unsell holding = %d, want 409", rec.Code)
	}
	sellTrade(t, s, item.ID, `{"sellPrice":300,"buyer":"Maria — Site #7","channel":"site"}`)
	if rec := unsellTrade(t, s, item.ID); rec.Code != http.StatusConflict {
		t.Fatalf("unsell site = %d, want 409", rec.Code)
	}
	if rec := unsellTrade(t, s, "nope"); rec.Code != http.StatusNotFound {
		t.Fatalf("unsell unknown = %d, want 404", rec.Code)
	}
}

func TestUnsellLigaSaleQueuesNothing(t *testing.T) {
	s, _ := queueServer(t)
	item := createTrade(t, s, "onepiece", ligaItem)
	sellTrade(t, s, item.ID, `{"sellPrice":450,"buyer":"Ciclano — Liga #1","channel":"liga"}`)
	if rec := unsellTrade(t, s, item.ID); rec.Code != http.StatusOK {
		t.Fatalf("unsell = %d", rec.Code)
	}
	all, err := s.SyncQueue.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Fatalf("tasks = %+v, want none", all)
	}
}

func TestUnsellCancelsPendingRemoval(t *testing.T) {
	s, _ := queueServer(t)
	item := createTrade(t, s, "onepiece", ligaItem)
	sellTrade(t, s, item.ID, `{"sellPrice":450,"channel":"balcao"}`)
	if rec := unsellTrade(t, s, item.ID); rec.Code != http.StatusOK {
		t.Fatalf("unsell = %d", rec.Code)
	}
	if got := pendingTasks(t, s); len(got) != 0 {
		t.Fatalf("pending = %+v, want removal cancelled", got)
	}
	all, err := s.SyncQueue.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Action != syncqueue.ActionRemove || all[0].Status != syncqueue.StatusDone {
		t.Fatalf("tasks = %+v", all)
	}
}

func TestUnsellAfterDoneRemovalQueuesRestore(t *testing.T) {
	s, _ := queueServer(t)
	item := createTrade(t, s, "onepiece", ligaItem)
	sellTrade(t, s, item.ID, `{"qty":2,"sellPrice":300,"channel":"balcao"}`)
	task := pendingTasks(t, s)[0]
	if _, err := s.SyncQueue.SetStatus(task.ID, syncqueue.StatusDone, ""); err != nil {
		t.Fatal(err)
	}
	if rec := unsellTrade(t, s, task.TradeID); rec.Code != http.StatusOK {
		t.Fatalf("unsell = %d: %s", rec.Code, rec.Body.String())
	}
	got := pendingTasks(t, s)
	if len(got) != 1 || got[0].Action != syncqueue.ActionRestore || got[0].Qty != 2 || got[0].Target != syncqueue.TargetLiga {
		t.Fatalf("pending = %+v, want one restore of 2", got)
	}
}
