package collecta

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArenaForwardsMethodPathQueryBodyAndToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.URL.Path != "/store/arena/members/u1/remove" || r.URL.RawQuery != "x=1" {
			t.Errorf("request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		if r.Header.Get("X-Store-Token") != "secret" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("headers = %v", r.Header)
		}
		if string(raw) != `{"reason":"trapaça"}` {
			t.Errorf("body = %s", raw)
		}
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"message":"Esse lançamento já foi estornado"}`))
	}))
	defer srv.Close()
	status, data, err := New(srv.URL+"/", "secret").Arena(context.Background(), http.MethodPost, "/store/arena/members/u1/remove", "x=1", []byte(`{"reason":"trapaça"}`))
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusConflict || string(data) != `{"message":"Esse lançamento já foi estornado"}` {
		t.Fatalf("status %d body %s", status, data)
	}
}

func TestArenaGetSendsNoBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodGet || len(raw) != 0 || r.Header.Get("Content-Type") != "" {
			t.Errorf("request = %s body %q content-type %q", r.Method, raw, r.Header.Get("Content-Type"))
		}
		w.Write([]byte(`{"members":3}`))
	}))
	defer srv.Close()
	status, data, err := New(srv.URL, "secret").Arena(context.Background(), http.MethodGet, "/store/arena/overview", "", nil)
	if err != nil || status != http.StatusOK || string(data) != `{"members":3}` {
		t.Fatalf("status %d body %s err %v", status, data, err)
	}
}

func TestArenaRefusesPathsOutsideArena(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
	}))
	defer srv.Close()
	c := New(srv.URL, "secret")
	for _, path := range []string{"/store/coupons/reserve", "/store/arena/", "/store/arena/members/../../coupons/reserve", "/store/arena/members//approve", "/store/arena/./overview"} {
		if _, _, err := c.Arena(context.Background(), http.MethodGet, path, "", nil); err == nil {
			t.Errorf("path %q was accepted", path)
		}
	}
	if calls != 0 {
		t.Fatalf("upstream called %d times", calls)
	}
}
