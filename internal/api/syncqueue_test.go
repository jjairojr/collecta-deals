package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"opdeals/internal/mercadopago"
	"opdeals/internal/orders"
	"opdeals/internal/syncqueue"
	"opdeals/internal/trades"
)

func queueServer(t *testing.T) (*Server, *trades.Store) {
	t.Helper()
	s, op, _ := purchaseServer(t)
	s.SyncQueue = syncqueue.NewStore(filepath.Join(t.TempDir(), "syncqueue.json"))
	return s, op
}

func pendingTasks(t *testing.T, s *Server) []syncqueue.Task {
	t.Helper()
	all, err := s.SyncQueue.List(syncqueue.StatusPending)
	if err != nil {
		t.Fatal(err)
	}
	return all
}

const ligaItem = `{"number":"OP16-080","name":"Teach","set":"OP16","condition":"NM","qty":3,"buyBRL":100,"status":"holding","ligaListed":true,"ligaQty":3}`

func TestSellOutsideLigaQueuesLigaRemoval(t *testing.T) {
	s, _ := queueServer(t)
	item := createTrade(t, s, "onepiece", ligaItem)
	if rec := sellTrade(t, s, item.ID, `{"qty":2,"sellPrice":300,"sellCurrency":"BRL","buyer":"Fulano","channel":"balcao"}`); rec.Code != http.StatusOK {
		t.Fatalf("sell = %d: %s", rec.Code, rec.Body.String())
	}
	got := pendingTasks(t, s)
	if len(got) != 1 {
		t.Fatalf("tasks = %+v, want 1", got)
	}
	task := got[0]
	if task.Target != syncqueue.TargetLiga || task.Action != syncqueue.ActionRemove || task.Qty != 2 ||
		task.Game != "onepiece" || task.Number != "OP16-080" || task.Name != "Teach" || task.Reason != "venda balcao" {
		t.Fatalf("task = %+v", task)
	}
}

func TestWriteOffQueuesLigaRemoval(t *testing.T) {
	s, _ := queueServer(t)
	item := createTrade(t, s, "onepiece", ligaItem)
	if rec := sellTrade(t, s, item.ID, `{"qty":1,"writeOff":"perda"}`); rec.Code != http.StatusOK {
		t.Fatalf("write-off = %d: %s", rec.Code, rec.Body.String())
	}
	got := pendingTasks(t, s)
	if len(got) != 1 || got[0].Qty != 1 || got[0].Reason != "baixa sem venda: perda" {
		t.Fatalf("tasks = %+v", got)
	}
}

func TestLigaSaleOrUnlistedItemQueuesNothing(t *testing.T) {
	s, _ := queueServer(t)
	listed := createTrade(t, s, "onepiece", ligaItem)
	if rec := sellTrade(t, s, listed.ID, `{"qty":1,"sellPrice":150,"buyer":"Ciclano — Liga #123","channel":"liga"}`); rec.Code != http.StatusOK {
		t.Fatalf("liga sell = %d", rec.Code)
	}
	if rec := sellTrade(t, s, listed.ID, `{"qty":1,"sellPrice":150,"buyer":"Ciclano — Liga #124"}`); rec.Code != http.StatusOK {
		t.Fatalf("legacy liga sell = %d", rec.Code)
	}
	unlisted := createTrade(t, s, "onepiece", `{"number":"OP16-001","name":"Luffy","set":"OP16","qty":2,"buyBRL":10,"status":"holding"}`)
	if rec := sellTrade(t, s, unlisted.ID, `{"qty":1,"sellPrice":20,"channel":"balcao"}`); rec.Code != http.StatusOK {
		t.Fatalf("unlisted sell = %d", rec.Code)
	}
	if got := pendingTasks(t, s); len(got) != 0 {
		t.Fatalf("tasks = %+v, want none", got)
	}
}

func TestSellingASoldLotIsConflict(t *testing.T) {
	s, _ := queueServer(t)
	item := createTrade(t, s, "onepiece", ligaItem)
	if rec := sellTrade(t, s, item.ID, `{"sellPrice":450,"channel":"balcao"}`); rec.Code != http.StatusOK {
		t.Fatalf("first sell = %d", rec.Code)
	}
	if rec := sellTrade(t, s, item.ID, `{"sellPrice":10,"channel":"balcao"}`); rec.Code != http.StatusConflict {
		t.Fatalf("second sell = %d, want 409", rec.Code)
	}
	if got := pendingTasks(t, s); len(got) != 1 {
		t.Fatalf("tasks = %d, want 1", len(got))
	}
}

func TestSiteSettleQueuesLigaRemoval(t *testing.T) {
	s, op := queueServer(t)
	item := createTrade(t, s, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","condition":"NM","qty":3,"buyBRL":100,"status":"holding","ligaListed":true,"listed":true,"askBRL":150}`)
	all, err := op.List()
	if err != nil {
		t.Fatal(err)
	}
	var stored trades.Trade
	for _, tr := range all {
		if tr.ID == item.ID {
			stored = tr
		}
	}
	o := orders.Order{
		Number: 7,
		Buyer:  orders.Buyer{Name: "Maria"},
		Lines:  []orders.Line{{SKU: trades.SKU("onepiece", stored), Game: "onepiece", Name: "Teach", Qty: 2, UnitBRL: 150}},
		Totals: orders.Totals{TotalBRL: 300},
	}
	if err := s.settle(&o, mercadopago.Payment{AmountBRL: 300}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if o.Status != orders.StatusPaid {
		t.Fatalf("status = %s, notes %s", o.Status, o.Notes)
	}
	got := pendingTasks(t, s)
	if len(got) != 1 || got[0].Qty != 2 || got[0].Reason != "venda site" {
		t.Fatalf("tasks = %+v", got)
	}
}

func queueCall(t *testing.T, method, path, id, body string, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if id != "" {
		req.SetPathValue("id", id)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestSyncQueueEndpoints(t *testing.T) {
	s, _ := queueServer(t)
	item := createTrade(t, s, "onepiece", ligaItem)
	sellTrade(t, s, item.ID, `{"qty":1,"sellPrice":150,"channel":"balcao"}`)
	task := pendingTasks(t, s)[0]

	rec := queueCall(t, http.MethodGet, "/api/syncqueue?status=pending", "", "", s.handleSyncQueueList)
	var listed struct {
		Tasks []syncqueue.Task `json:"tasks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil || len(listed.Tasks) != 1 {
		t.Fatalf("list = %d %s", rec.Code, rec.Body.String())
	}

	rec = queueCall(t, http.MethodPost, "/api/syncqueue/"+task.ID+"/error", task.ID, `{"error":"oferta não achada"}`, s.handleSyncQueueError)
	var failed syncqueue.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &failed); err != nil || failed.Status != syncqueue.StatusError || failed.Error != "oferta não achada" {
		t.Fatalf("error = %d %s", rec.Code, rec.Body.String())
	}

	rec = queueCall(t, http.MethodPost, "/api/syncqueue/"+task.ID+"/retry", task.ID, "", s.handleSyncQueueRetry)
	var retried syncqueue.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &retried); err != nil || retried.Status != syncqueue.StatusPending {
		t.Fatalf("retry = %d %s", rec.Code, rec.Body.String())
	}

	rec = queueCall(t, http.MethodPost, "/api/syncqueue/"+task.ID+"/done", task.ID, "", s.handleSyncQueueDone)
	var done syncqueue.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &done); err != nil || done.Status != syncqueue.StatusDone {
		t.Fatalf("done = %d %s", rec.Code, rec.Body.String())
	}

	if rec := queueCall(t, http.MethodPost, "/api/syncqueue/nope/done", "nope", "", s.handleSyncQueueDone); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown id = %d, want 404", rec.Code)
	}
}

func TestAttachClearsSemLastro(t *testing.T) {
	s, op := queueServer(t)
	order := newOrder(t, s, `{"date":"2026-10-30","store":"Legends"}`)
	sale := createTrade(t, s, "onepiece", `{"number":"OP16-080","name":"Teach","qty":1,"status":"sold","sellPrice":150,"buyer":"Fulano — Liga #9"}`)
	if !sale.SemLastro {
		t.Fatalf("sale created as sold should be sem lastro: %+v", sale)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/purchases/"+order.ID+"/attach",
		strings.NewReader(`{"items":[{"id":"`+sale.ID+`","game":"onepiece"}]}`))
	req.SetPathValue("id", order.ID)
	s.handlePurchaseAttach(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("attach = %d: %s", rec.Code, rec.Body.String())
	}
	all, err := op.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range all {
		if tr.ID == sale.ID && (tr.SemLastro || tr.PurchaseID != order.ID) {
			t.Fatalf("after attach = %+v", tr)
		}
	}
}
