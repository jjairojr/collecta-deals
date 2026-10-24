package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGateRequiresAdminTokenOnPrivateRoutes(t *testing.T) {
	s, _ := storefrontServer(t)
	s.adminToken = "secret"
	h := s.Handler()

	cases := []struct {
		method, path, token string
		want                int
	}{
		{http.MethodGet, "/api/trades?game=onepiece", "", http.StatusUnauthorized},
		{http.MethodGet, "/api/trades?game=onepiece", "wrong", http.StatusUnauthorized},
		{http.MethodGet, "/api/trades?game=onepiece", "secret", http.StatusOK},
		{http.MethodPost, "/api/trades/x/sell?game=onepiece", "", http.StatusUnauthorized},
		{http.MethodGet, "/api/tracking/card?game=onepiece", "", http.StatusUnauthorized},
		{http.MethodGet, "/api/storefront", "", http.StatusOK},
		{http.MethodOptions, "/api/trades", "", http.StatusNoContent},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		if c.token != "" {
			req.Header.Set("X-Admin-Token", c.token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s %s token=%q: status %d, want %d", c.method, c.path, c.token, rec.Code, c.want)
		}
	}
}

func TestGateOpenWithoutConfiguredToken(t *testing.T) {
	s, _ := storefrontServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/trades?game=onepiece", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200 when no admin token is configured", rec.Code)
	}
}
