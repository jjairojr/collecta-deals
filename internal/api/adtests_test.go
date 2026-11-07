package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"opdeals/internal/adtestq"
)

const adTestBody = `{"objective":"seguidores","audience":"goiania","budgetCents":10000,"days":4,"creatives":[
{"path":"/x/a.mp4","sha256":"a","size":1,"kind":"video","thumb":"/x/a.png","text":"oi","hook":"preço"},
{"path":"/x/b.png","sha256":"b","size":1,"kind":"imagem","text":"oi","hook":"loja"}]}`

func TestAdTestFlow(t *testing.T) {
	s, _, _ := purchaseServer(t)
	s.adminToken = "segredo"
	s.AdTests = adtestq.NewStore(filepath.Join(t.TempDir(), "adtests.json"))
	h := s.Handler()
	if rec := call(t, h, http.MethodPost, "/api/adtests", adTestBody, false); rec.Code != http.StatusUnauthorized {
		t.Fatalf("sem token = %d", rec.Code)
	}
	rec := call(t, h, http.MethodPost, "/api/adtests", adTestBody, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("criar = %d %s", rec.Code, rec.Body)
	}
	var created adtestq.Test
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if rec := call(t, h, http.MethodPost, "/api/adtests", adTestBody, true); rec.Code != http.StatusConflict {
		t.Fatalf("teto = %d", rec.Code)
	}
	if rec := call(t, h, http.MethodPost, "/api/aprovar-anuncio/"+created.ID+"?acao=aprovar&t=errado", "", false); rec.Code != http.StatusForbidden {
		t.Fatalf("token errado = %d", rec.Code)
	}
	if rec := call(t, h, http.MethodPost, "/api/aprovar-anuncio/"+created.ID+"?acao=aprovar&t="+created.Token, "", false); rec.Code != http.StatusOK {
		t.Fatalf("aprovar = %d %s", rec.Code, rec.Body)
	}
	rec = call(t, h, http.MethodGet, "/api/adtests?status=approved", "", true)
	var list struct {
		Items          []adtestq.Test `json:"items"`
		AvailableCents int64          `json:"availableCents"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Token != "" || list.AvailableCents != 5000 {
		t.Fatalf("lista = %+v", list)
	}
	started := `{"meta":{"campaignId":"c","adSetId":"s","ads":[{"id":"a1"}]},"endsAt":"2026-11-10T23:59:00-03:00"}`
	if rec := call(t, h, http.MethodPost, "/api/adtests/"+created.ID+"/started", started, true); rec.Code != http.StatusOK {
		t.Fatalf("started = %d %s", rec.Code, rec.Body)
	}
}
