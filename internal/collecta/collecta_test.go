package collecta

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReserveSendsTokenAndBody(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/store/coupons/reserve" || r.Header.Get("X-Store-Token") != "secret" {
			t.Errorf("request = %s %s token %q", r.Method, r.URL.Path, r.Header.Get("X-Store-Token"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"coupons":[{"id":"c1","code":"IND-AAAAAA","valueCents":1000}]}`))
	}))
	defer srv.Close()
	coupons, err := New(srv.URL+"/", "secret").Reserve(context.Background(), "u1", []string{"c1"}, 25000)
	if err != nil {
		t.Fatal(err)
	}
	if len(coupons) != 1 || coupons[0].Code != "IND-AAAAAA" || coupons[0].ValueCents != 1000 {
		t.Fatalf("coupons = %+v", coupons)
	}
	if got["userId"] != "u1" || got["totalCents"] != float64(25000) {
		t.Fatalf("body = %+v", got)
	}
}

func TestReserveMapsRejections(t *testing.T) {
	cases := []struct {
		status   int
		body     string
		message  string
		rejected bool
	}{
		{http.StatusNotFound, `{"message":"Cupom não encontrado"}`, "Cupom não encontrado", true},
		{http.StatusBadRequest, `not json`, "cupom recusado", true},
		{http.StatusUnauthorized, `{"message":"unauthorized"}`, "unauthorized", false},
	}
	for _, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(c.status)
			w.Write([]byte(c.body))
		}))
		_, err := New(srv.URL, "secret").Reserve(context.Background(), "u1", []string{"c1"}, 25000)
		srv.Close()
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("status %d: err = %v", c.status, err)
		}
		if apiErr.Status != c.status || apiErr.Message != c.message || apiErr.Rejected() != c.rejected {
			t.Fatalf("status %d: apiErr = %+v rejected %v", c.status, apiErr, apiErr.Rejected())
		}
	}
}

func TestServerErrorIsNotAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	err := New(srv.URL, "secret").Release(context.Background(), []string{"IND-AAAAAA"})
	var apiErr *APIError
	if err == nil || errors.As(err, &apiErr) {
		t.Fatalf("err = %v", err)
	}
}

func TestReleaseRequiresAck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Codes []string `json:"codes"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if len(body.Codes) != 2 {
			t.Errorf("codes = %v", body.Codes)
		}
		w.Write([]byte(`{"ok":true,"restored":1}`))
	}))
	defer srv.Close()
	if err := New(srv.URL, "secret").Release(context.Background(), []string{"IND-A", "IND-B"}); err != nil {
		t.Fatal(err)
	}
}
