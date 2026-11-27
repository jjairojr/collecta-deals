package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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
	// O pop-up único (formato antigo) volta como lista de um.
	if err := json.Unmarshal(rec.Body.Bytes(), &full); err != nil || len(full.Phrases) != 2 || full.Popup != nil || len(full.Popups) != 1 || full.Popups[0].SKU != sku {
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

func vitrineUpload(t *testing.T, s *Server, token string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "arte.webp")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(data)
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/vitrine/image", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("X-Admin-Token", token)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func TestVitrineImageUploadAndServe(t *testing.T) {
	s, _ := vitrineServer(t)
	webp := []byte("RIFF\x10\x00\x00\x00WEBPVP8 arte")
	if rec := vitrineUpload(t, s, "", webp); rec.Code != http.StatusUnauthorized {
		t.Fatalf("upload sem token = %d", rec.Code)
	}
	if rec := vitrineUpload(t, s, "secret", []byte("<svg/>")); rec.Code != http.StatusBadRequest {
		t.Fatalf("svg = %d %s", rec.Code, rec.Body)
	}
	big := append([]byte("RIFF\x10\x00\x00\x00WEBPVP8 "), make([]byte, vitrine.MaxImageBytes)...)
	if rec := vitrineUpload(t, s, "secret", big); rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("acima de 1 MB = %d", rec.Code)
	}
	rec := vitrineUpload(t, s, "secret", webp)
	var out struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || rec.Code != http.StatusOK || !vitrine.IsImageURL(out.URL) {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body)
	}
	get := vitrineDo(t, s, http.MethodGet, out.URL, "", nil)
	if get.Code != http.StatusOK || !bytes.Equal(get.Body.Bytes(), webp) || get.Header().Get("Content-Type") != "image/webp" ||
		!strings.Contains(get.Header().Get("Cache-Control"), "immutable") || get.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("GET imagem = %d %v", get.Code, get.Header())
	}
	for _, bad := range []string{"/api/storefront/vitrine/img/..%2Fvitrine.json", "/api/storefront/vitrine/img/0123456789abcdef.png", "/api/storefront/vitrine/img/x.svg"} {
		if rec := vitrineDo(t, s, http.MethodGet, bad, "", nil); rec.Code != http.StatusNotFound {
			t.Errorf("%s = %d", bad, rec.Code)
		}
	}
}

func TestVitrineLivePopupSkipsUnlisted(t *testing.T) {
	s, sku := vitrineServer(t)
	c := vitrine.Config{Phrases: []vitrine.Phrase{}, Popups: []vitrine.Popup{
		{Enabled: true, SKU: sku, Title: "Leve o Teach", Window: vitrine.Window{From: "2026-11-27", Until: "2026-12-01"}},
		{Enabled: true, SKU: "single|onepiece|nao-existe||", Title: "Para depois", Window: vitrine.Window{From: "2026-12-02"}},
	}}
	if rec := vitrineDo(t, s, http.MethodPut, "/api/vitrine", "secret", c); rec.Code != http.StatusOK {
		t.Fatalf("pop-up agendado para depois com produto ainda fora da vitrine devia salvar: %d %s", rec.Code, rec.Body)
	}
	var live vitrine.Live
	rec := vitrineDo(t, s, http.MethodGet, "/api/storefront/vitrine", "", nil)
	if err := json.Unmarshal(rec.Body.Bytes(), &live); err != nil || live.Popup == nil || live.Popup.SKU != sku {
		t.Fatalf("01/12 = %s", rec.Body)
	}
	s.postSale.now = func() time.Time { return time.Date(2026, 12, 3, 15, 0, 0, 0, saoPaulo) }
	rec = vitrineDo(t, s, http.MethodGet, "/api/storefront/vitrine", "", nil)
	live = vitrine.Live{}
	if err := json.Unmarshal(rec.Body.Bytes(), &live); err != nil || live.Popup != nil {
		t.Fatalf("03/12: o produto não está à venda, então sem pop-up: %s", rec.Body)
	}
	c.Popups[0].Window = vitrine.Window{}
	c.Popups[0].SKU = "single|onepiece|nao-existe||"
	if rec := vitrineDo(t, s, http.MethodPut, "/api/vitrine", "secret", c); rec.Code != http.StatusBadRequest {
		t.Fatalf("pop-up de hoje com produto fora da vitrine devia ser recusado: %d", rec.Code)
	}
}

func TestVitrineHeroSlidesLiveDropUnlisted(t *testing.T) {
	s, sku := vitrineServer(t)
	c := vitrine.Config{Phrases: []vitrine.Phrase{}, HeroStyle: vitrine.HeroStyleSpotlight, HeroSlides: []vitrine.HeroSlide{
		{Enabled: true, Kind: vitrine.SlideProduct, SKU: sku, Badge: "Novo", Phrase: "Teach chegou"},
		{Enabled: true, Kind: vitrine.SlideProduct, SKU: "single|onepiece|nao-existe||", Window: vitrine.Window{From: "2026-12-05"}},
	}}
	if rec := vitrineDo(t, s, http.MethodPut, "/api/vitrine", "secret", c); rec.Code != http.StatusOK {
		t.Fatalf("salvar: %d %s", rec.Code, rec.Body)
	}
	today := c
	today.HeroSlides = append(today.HeroSlides, vitrine.HeroSlide{Enabled: true, Kind: vitrine.SlideProduct, SKU: "single|onepiece|nao-existe||"})
	if rec := vitrineDo(t, s, http.MethodPut, "/api/vitrine", "secret", today); rec.Code != http.StatusBadRequest {
		t.Fatalf("slide de hoje sem estoque devia recusar: %d", rec.Code)
	}
	var live vitrine.Live
	rec := vitrineDo(t, s, http.MethodGet, "/api/storefront/vitrine", "", nil)
	if err := json.Unmarshal(rec.Body.Bytes(), &live); err != nil || len(live.HeroSlides) != 1 || live.HeroSlides[0].SKU != sku || live.HeroStyle != vitrine.HeroStyleSpotlight {
		t.Fatalf("01/12 = %s", rec.Body)
	}
	s.postSale.now = func() time.Time { return time.Date(2026, 12, 6, 15, 0, 0, 0, saoPaulo) }
	live = vitrine.Live{}
	rec = vitrineDo(t, s, http.MethodGet, "/api/storefront/vitrine", "", nil)
	if err := json.Unmarshal(rec.Body.Bytes(), &live); err != nil || len(live.HeroSlides) != 1 || live.HeroSlides[0].SKU != sku {
		t.Fatalf("06/12: o slide sem estoque some e fica só o Teach: %s", rec.Body)
	}
}
