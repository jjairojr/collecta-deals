package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"opdeals/internal/publishq"
)

func publishServer(t *testing.T) http.Handler {
	t.Helper()
	s, _, _ := purchaseServer(t)
	s.adminToken = "segredo"
	s.Publish = publishq.NewStore(filepath.Join(t.TempDir(), "publishq.json"))
	return s.Handler()
}

func call(t *testing.T, h http.Handler, method, path, body string, admin bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if admin {
		req.Header.Set("X-Admin-Token", "segredo")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

const publishBody = `{"channel":"instagram","format":"reels","files":[{"path":"/x/v.mp4","sha256":"abc","size":10}],"caption":"Faltam 10 dias","postAt":"2026-11-06T15:00:00-03:00"}`

func TestPublishFlow(t *testing.T) {
	h := publishServer(t)
	if rec := call(t, h, http.MethodPost, "/api/publish", publishBody, false); rec.Code != http.StatusUnauthorized {
		t.Fatalf("criar sem token devia dar 401: %d", rec.Code)
	}
	rec := call(t, h, http.MethodPost, "/api/publish", publishBody, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("criar = %d %s", rec.Code, rec.Body.String())
	}
	var created publishq.Item
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Token == "" || created.Status != publishq.StatusPending {
		t.Fatalf("criado errado: %+v", created)
	}

	if rec := call(t, h, http.MethodPost, "/api/aprovar/"+created.ID+"?acao=aprovar&t=errado", "", false); rec.Code != http.StatusForbidden {
		t.Fatalf("código errado devia dar 403: %d", rec.Code)
	}
	if rec := call(t, h, http.MethodGet, "/api/aprovar/"+created.ID+"?acao=aprovar&t="+created.Token, "", false); rec.Code == http.StatusOK {
		t.Fatal("GET não pode aprovar (prévia de link no celular)")
	}
	rec = call(t, h, http.MethodPost, "/api/aprovar/"+created.ID+"?acao=aprovar&t="+created.Token, "", false)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Aprovado") {
		t.Fatalf("aprovar = %d %s", rec.Code, rec.Body.String())
	}

	rec = call(t, h, http.MethodGet, "/api/publish?status=approved", "", true)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), created.Token) || !strings.Contains(rec.Body.String(), created.ID) {
		t.Fatalf("lista de aprovados errada: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(t, h, http.MethodGet, "/api/publish", "", false); rec.Code != http.StatusUnauthorized {
		t.Fatalf("lista sem token devia dar 401: %d", rec.Code)
	}

	if rec := call(t, h, http.MethodPost, "/api/publish/"+created.ID+"/published", `{"link":""}`, true); rec.Code != http.StatusBadRequest {
		t.Fatalf("publicado sem link devia dar 400: %d", rec.Code)
	}
	rec = call(t, h, http.MethodPost, "/api/publish/"+created.ID+"/published", `{"link":"https://www.instagram.com/reel/abc/"}`, true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"published"`) {
		t.Fatalf("publicado = %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(t, h, http.MethodPost, "/api/publish/"+created.ID+"/failed", `{"error":"x"}`, true); rec.Code != http.StatusConflict {
		t.Fatalf("falha depois de publicado devia dar 409: %d", rec.Code)
	}
}

func TestPublishRejectAndValidation(t *testing.T) {
	h := publishServer(t)
	rec := call(t, h, http.MethodPost, "/api/publish", publishBody, true)
	var created publishq.Item
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	rec = call(t, h, http.MethodPost, "/api/aprovar/"+created.ID+"?acao=rejeitar&t="+created.Token, "", false)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Rejeitado") {
		t.Fatalf("rejeitar = %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(t, h, http.MethodPost, "/api/aprovar/"+created.ID+"?acao=aprovar&t="+created.Token, "", false); rec.Code != http.StatusConflict {
		t.Fatalf("aprovar depois de rejeitar devia dar 409: %d", rec.Code)
	}
	if rec := call(t, h, http.MethodPost, "/api/aprovar/"+created.ID+"?acao=talvez&t="+created.Token, "", false); rec.Code != http.StatusBadRequest {
		t.Fatalf("ação desconhecida devia dar 400: %d", rec.Code)
	}
	bad := strings.Replace(publishBody, `"reels"`, `"story"`, 1)
	if rec := call(t, h, http.MethodPost, "/api/publish", bad, true); rec.Code != http.StatusBadRequest {
		t.Fatalf("formato inválido devia dar 400: %d", rec.Code)
	}
}
