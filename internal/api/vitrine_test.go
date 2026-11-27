package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"opdeals/internal/trades"
	"opdeals/internal/vitrine"
)

func vitrineServer(t *testing.T) (*Server, string) {
	t.Helper()
	s, store := storefrontServer(t)
	s.adminToken = "secret"
	s.Vitrine = vitrine.NewStore(filepath.Join(t.TempDir(), "vitrine.json"))
	day := time.Date(2026, 12, 1, 15, 0, 0, 0, saoPaulo)
	s.postSale.now = func() time.Time { return day }
	tr, err := store.Add(trades.Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 2, BuyBRL: 100, Status: "holding"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetListings(map[string]trades.Listing{tr.ID: {AskBRL: 250, Listed: true}}); err != nil {
		t.Fatal(err)
	}
	return s, trades.SKU("onepiece", tr)
}

func vitrineDo(t *testing.T, s *Server, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("X-Admin-Token", token)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func TestVitrineEditNeedsAdminAndLiveIsPublic(t *testing.T) {
	s, _ := vitrineServer(t)
	if rec := vitrineDo(t, s, http.MethodGet, "/api/vitrine", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET sem token = %d", rec.Code)
	}
	if rec := vitrineDo(t, s, http.MethodPut, "/api/vitrine", "", vitrine.Example()); rec.Code != http.StatusUnauthorized {
		t.Fatalf("PUT sem token = %d", rec.Code)
	}
	rec := vitrineDo(t, s, http.MethodGet, "/api/storefront/vitrine", "", nil)
	var live vitrine.Live
	if err := json.Unmarshal(rec.Body.Bytes(), &live); err != nil || rec.Code != http.StatusOK || len(live.Phrases) != 0 || live.Banner != nil || live.Popup != nil {
		t.Fatalf("sem config, nada aparece: %d %s", rec.Code, rec.Body)
	}
}

func TestVitrinePutAndLive(t *testing.T) {
	s, sku := vitrineServer(t)
	c := vitrine.Example()
	c.Popup = &vitrine.Popup{Enabled: true, SKU: sku, Title: "Leve o Teach", Text: "Da coleção OP16"}
	rec := vitrineDo(t, s, http.MethodPut, "/api/vitrine", "secret", c)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT = %d %s", rec.Code, rec.Body)
	}
	rec = vitrineDo(t, s, http.MethodGet, "/api/vitrine", "secret", nil)
	var full vitrine.Config
	if err := json.Unmarshal(rec.Body.Bytes(), &full); err != nil || len(full.Phrases) != 2 || full.Popup == nil {
		t.Fatalf("GET admin: %s", rec.Body)
	}

	rec = vitrineDo(t, s, http.MethodGet, "/api/storefront/vitrine", "", nil)
	var live vitrine.Live
	if err := json.Unmarshal(rec.Body.Bytes(), &live); err != nil {
		t.Fatal(err)
	}
	// 01/12: só a frase do prazo de 09/12; banner no ar; pop-up com o produto à venda.
	if len(live.Phrases) != 1 || !strings.Contains(live.Phrases[0], "09/12") || live.Banner == nil || live.Popup == nil || live.Popup.SKU != sku {
		t.Fatalf("live 01/12 = %+v", live)
	}
	if rec.Header().Get("Cache-Control") != "public, max-age=60" {
		t.Errorf("cache = %q", rec.Header().Get("Cache-Control"))
	}
}

func TestVitrineRejectsUnlistedSKUAndBadContent(t *testing.T) {
	s, _ := vitrineServer(t)
	c := vitrine.Config{Popup: &vitrine.Popup{Enabled: true, SKU: "single|onepiece|nao existe||", Title: "Leve agora"}}
	rec := vitrineDo(t, s, http.MethodPut, "/api/vitrine", "secret", c)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "não está à venda") {
		t.Fatalf("SKU fora da vitrine = %d %s", rec.Code, rec.Body)
	}
	bad := vitrine.Config{Phrases: []vitrine.Phrase{{Text: "Frete grátis no Natal"}}}
	if rec := vitrineDo(t, s, http.MethodPut, "/api/vitrine", "secret", bad); rec.Code != http.StatusBadRequest {
		t.Fatalf("frete grátis = %d", rec.Code)
	}
}

func TestVitrineLiveHidesPopupWhenProductLeaves(t *testing.T) {
	s, sku := vitrineServer(t)
	c := vitrine.Config{Popup: &vitrine.Popup{Enabled: true, SKU: sku, Title: "Leve o Teach"}}
	if rec := vitrineDo(t, s, http.MethodPut, "/api/vitrine", "secret", c); rec.Code != http.StatusOK {
		t.Fatalf("PUT = %d %s", rec.Code, rec.Body)
	}
	// O produto sai da vitrine (vendido ou despublicado) depois de configurado.
	gs := s.games["onepiece"]
	all, _ := gs.Trades.List()
	if _, err := gs.Trades.SetListings(map[string]trades.Listing{all[0].ID: {AskBRL: 250, Listed: false}}); err != nil {
		t.Fatal(err)
	}
	rec := vitrineDo(t, s, http.MethodGet, "/api/storefront/vitrine", "", nil)
	var live vitrine.Live
	if err := json.Unmarshal(rec.Body.Bytes(), &live); err != nil || live.Popup != nil {
		t.Fatalf("pop-up de produto fora da vitrine apareceu: %s", rec.Body)
	}
}
