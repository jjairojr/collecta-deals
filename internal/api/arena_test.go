package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type arenaCall struct {
	method, path, query, body string
}

type fakeArena struct {
	calls  []arenaCall
	status int
	reply  string
	err    error
}

func (f *fakeArena) Arena(_ context.Context, method, path, rawQuery string, body []byte) (int, []byte, error) {
	f.calls = append(f.calls, arenaCall{method: method, path: path, query: rawQuery, body: string(body)})
	if f.err != nil {
		return 0, nil, f.err
	}
	return f.status, []byte(f.reply), nil
}

func arenaServer(t *testing.T, f *fakeArena) http.Handler {
	t.Helper()
	s, _ := storefrontServer(t)
	s.adminToken = "secret"
	if f != nil {
		s.Arena = f
	}
	return s.Handler()
}

func arenaRequest(h http.Handler, method, target, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if token != "" {
		req.Header.Set("X-Admin-Token", token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestArenaRoutesSitBehindTheGate(t *testing.T) {
	f := &fakeArena{status: http.StatusOK, reply: `{}`}
	h := arenaServer(t, f)
	for _, token := range []string{"", "wrong"} {
		if rec := arenaRequest(h, http.MethodGet, "/api/arena/overview", "", token); rec.Code != http.StatusUnauthorized {
			t.Errorf("token %q: status %d, want 401", token, rec.Code)
		}
		if rec := arenaRequest(h, http.MethodPost, "/api/arena/credit", `{}`, token); rec.Code != http.StatusUnauthorized {
			t.Errorf("token %q: credit status %d, want 401", token, rec.Code)
		}
	}
	if len(f.calls) != 0 {
		t.Fatalf("upstream called without admin token: %+v", f.calls)
	}
	if rec := arenaRequest(h, http.MethodGet, "/api/arena/overview", "", "secret"); rec.Code != http.StatusOK {
		t.Fatalf("with token: status %d", rec.Code)
	}
}

func TestArenaProxyForwardsPathQueryAndBody(t *testing.T) {
	f := &fakeArena{status: http.StatusOK, reply: `{"entries":[],"nextCursor":null}`}
	h := arenaServer(t, f)
	rec := arenaRequest(h, http.MethodGet, "/api/arena/members/u%201/statement?cursor=abc", "", "secret")
	if rec.Code != http.StatusOK || rec.Body.String() != f.reply || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	rec = arenaRequest(h, http.MethodPost, "/api/arena/duels/d1/resolve", `{"outcome":"DRAW","reason":"combinado"}`, "secret")
	if rec.Code != http.StatusOK {
		t.Fatalf("resolve status %d", rec.Code)
	}
	rec = arenaRequest(h, http.MethodPost, "/api/arena/members/u1/reject", "", "secret")
	if rec.Code != http.StatusOK {
		t.Fatalf("reject status %d", rec.Code)
	}
	rec = arenaRequest(h, http.MethodPost, "/api/arena/events/e1/end", `{"first":"u1","second":null,"third":null}`, "secret")
	if rec.Code != http.StatusOK {
		t.Fatalf("end status %d", rec.Code)
	}
	want := []arenaCall{
		{method: http.MethodGet, path: "/store/arena/members/u%201/statement", query: "cursor=abc"},
		{method: http.MethodPost, path: "/store/arena/duels/d1/resolve", body: `{"outcome":"DRAW","reason":"combinado"}`},
		{method: http.MethodPost, path: "/store/arena/members/u1/reject", body: `{}`},
		{method: http.MethodPost, path: "/store/arena/events/e1/end", body: `{"first":"u1","second":null,"third":null}`},
	}
	if len(f.calls) != len(want) {
		t.Fatalf("calls = %+v", f.calls)
	}
	for i := range want {
		if f.calls[i] != want[i] {
			t.Errorf("call %d = %+v, want %+v", i, f.calls[i], want[i])
		}
	}
}

func TestArenaProxyPassesThroughRejections(t *testing.T) {
	f := &fakeArena{status: http.StatusConflict, reply: `{"message":"Esse lançamento já foi estornado"}`}
	h := arenaServer(t, f)
	rec := arenaRequest(h, http.MethodPost, "/api/arena/entries/e1/reverse", `{"reason":"lançado errado"}`, "secret")
	if rec.Code != http.StatusConflict || rec.Body.String() != f.reply {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestArenaProxyErrors(t *testing.T) {
	cases := []struct {
		name    string
		arena   *fakeArena
		method  string
		target  string
		body    string
		status  int
		message string
		calls   int
	}{
		{"disabled", nil, http.MethodGet, "/api/arena/overview", "", http.StatusServiceUnavailable, "COLLECTA_API", 0},
		{"unreachable", &fakeArena{err: errors.New("dial tcp: refused")}, http.MethodGet, "/api/arena/overview", "", http.StatusBadGateway, "indisponível", 1},
		{"store token refused", &fakeArena{status: http.StatusUnauthorized, reply: `{"message":"unauthorized"}`}, http.MethodGet, "/api/arena/games", "", http.StatusBadGateway, "COLLECTA_STORE_TOKEN", 1},
		{"bad json", &fakeArena{status: http.StatusOK}, http.MethodPost, "/api/arena/credit", "{nope", http.StatusBadRequest, "corpo inválido", 0},
		{"dot id", &fakeArena{status: http.StatusOK}, http.MethodPost, "/api/arena/duels/%2E%2E/annul", `{}`, http.StatusBadRequest, "id inválido", 0},
	}
	for _, c := range cases {
		h := arenaServer(t, c.arena)
		rec := arenaRequest(h, c.method, c.target, c.body, "secret")
		if rec.Code != c.status || !strings.Contains(rec.Body.String(), c.message) {
			t.Errorf("%s: status %d body %s", c.name, rec.Code, rec.Body.String())
		}
		if rec.Code == http.StatusUnauthorized {
			t.Errorf("%s: a Collecta failure must not look like a dashboard 401", c.name)
		}
		if c.arena != nil && len(c.arena.calls) != c.calls {
			t.Errorf("%s: %d upstream calls, want %d", c.name, len(c.arena.calls), c.calls)
		}
	}
}
