package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"opdeals/internal/metaevents"
)

// panelStub faz de conta que é o painel: devolve as compras e a lista de
// clientes já com hash, como /api/meta/events e /api/meta/customers.
func panelStub(t *testing.T, events []metaevents.Event, rows [][]string) *app {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Admin-Token") != "adm" {
			http.Error(w, "token", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/meta/events":
			_ = json.NewEncoder(w).Encode(map[string]any{"events": events, "skipped": map[string]int{"site_sem_navegador": 1, "liga_sem_telefone": 2, "balcao_sem_contato": 3}})
		case "/api/meta/customers":
			_ = json.NewEncoder(w).Encode(map[string]any{"schema": metaevents.CustomerSchema, "data": rows, "counts": map[string]int{"site": 1, "liga": 1, "balcao": 0}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return newApp(config{OpdealsURL: srv.URL, AdminToken: "adm", DataDir: t.TempDir()}, http.DefaultClient, time.Now)
}

func someEvents() []metaevents.Event {
	p := metaevents.Person{Name: "Maria Souza", Email: "maria@x.com", Phone: "62993399980"}
	return []metaevents.Event{
		{EventName: "Purchase", EventTime: time.Now().Unix(), EventID: "liga-111", ActionSource: "other", UserData: p.User(), CustomData: metaevents.CustomData{Value: 150, Currency: "BRL"}},
		{EventName: "Purchase", EventTime: time.Now().Unix(), EventID: "pedido-abc", ActionSource: "website", EventSourceURL: "https://www.collectatcg.com.br/pedido/abc", UserData: p.User(), CustomData: metaevents.CustomData{Value: 650, Currency: "BRL"}},
	}
}

func TestSendConversionsOncePerPurchase(t *testing.T) {
	_, meta, ex := newAdFixture(t)
	ex.g.cfg.PixelID = "959341777245828"
	a := panelStub(t, someEvents(), nil)
	res, err := sendConversions(context.Background(), a, ex.g, "")
	if err != nil || res.Sent != 2 {
		t.Fatalf("primeira rodada = %+v %v", res, err)
	}
	calls := meta.find("959341777245828/events")
	if len(calls) != 2 || !strings.Contains(calls[0].form["data"], `"event_id":"liga-111"`) || !strings.Contains(calls[1].form["data"], `"action_source":"website"`) || calls[0].form["test_event_code"] != "" {
		t.Fatalf("envios = %+v", calls)
	}
	res, err = sendConversions(context.Background(), a, ex.g, "")
	if err != nil || res.Sent != 0 || res.Already != 2 || len(meta.find("959341777245828/events")) != 2 {
		t.Fatalf("segunda rodada devia não reenviar: %+v %v", res, err)
	}
	// O estado e o log não têm dado de cliente: só ids e contagens.
	for _, name := range []string{capiStateFile, "anuncios.log"} {
		data, err := os.ReadFile(filepath.Join(a.cfg.DataDir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, raw := range []string{"maria", "Maria", "99339", metaevents.Hash("maria@x.com")} {
			if strings.Contains(string(data), raw) {
				t.Errorf("%s tem dado de cliente (%q): %s", name, raw, data)
			}
		}
	}
	log, _ := os.ReadFile(filepath.Join(a.cfg.DataDir, "anuncios.log"))
	if !strings.Contains(string(log), "2 enviadas") || !strings.Contains(string(log), "2 da Liga sem telefone") {
		t.Fatalf("log = %s", log)
	}
}

func TestSendConversionsTestCodeDoesNotCount(t *testing.T) {
	_, meta, ex := newAdFixture(t)
	a := panelStub(t, someEvents(), nil)
	if res, err := sendConversions(context.Background(), a, ex.g, "TEST123"); err != nil || res.Sent != 2 {
		t.Fatalf("teste = %+v %v", res, err)
	}
	if c := meta.find("px/events"); len(c) != 2 || c[0].form["test_event_code"] != "TEST123" {
		t.Fatalf("envios de teste = %+v", c)
	}
	if res, _ := sendConversions(context.Background(), a, ex.g, ""); res.Sent != 2 {
		t.Fatalf("depois do teste, a rodada de verdade manda de novo: %+v", res)
	}
}

func TestSendConversionsFailureRetriesNextRound(t *testing.T) {
	_, meta, ex := newAdFixture(t)
	a := panelStub(t, someEvents(), nil)
	meta.failOn = "px/events"
	res, err := sendConversions(context.Background(), a, ex.g, "")
	if err == nil || res.Failed != 2 || res.Sent != 0 {
		t.Fatalf("falha = %+v %v", res, err)
	}
	meta.failOn = ""
	if res, err := sendConversions(context.Background(), a, ex.g, ""); err != nil || res.Sent != 2 {
		t.Fatalf("a próxima rodada tenta de novo: %+v %v", res, err)
	}
	ex.g.cfg.PixelID = ""
	if _, err := sendConversions(context.Background(), a, ex.g, ""); err == nil || !strings.Contains(err.Error(), "META_PIXEL") {
		t.Fatalf("sem pixel: %v", err)
	}
}

func TestUploadCustomers(t *testing.T) {
	_, meta, ex := newAdFixture(t)
	row, _ := metaevents.Person{Email: "maria@x.com"}.Row()
	row2, _ := metaevents.Person{Phone: "62977776666"}.Row()
	a := panelStub(t, nil, [][]string{row, row2})
	if _, err := uploadCustomers(context.Background(), a, ex.g, "", 180); err == nil || !strings.Contains(err.Error(), "META_LISTA_CLIENTES") {
		t.Fatalf("sem lista: %v", err)
	}
	res, err := uploadCustomers(context.Background(), a, ex.g, "238000", 180)
	if err != nil || res.People != 2 || res.Site != 1 || res.Liga != 1 {
		t.Fatalf("lista = %+v %v", res, err)
	}
	calls := meta.find("238000/users")
	if len(calls) != 1 || !strings.Contains(calls[0].form["payload"], `"schema":["EMAIL","PHONE","FN","LN","CT","ST","ZIP","COUNTRY"]`) || !strings.Contains(calls[0].form["payload"], metaevents.Hash("maria@x.com")) || strings.Contains(calls[0].form["payload"], "maria@") {
		t.Fatalf("envio da lista = %+v", calls)
	}
}
