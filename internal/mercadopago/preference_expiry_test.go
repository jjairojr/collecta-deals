package mercadopago

import (
	"context"
	"testing"
	"time"

	"github.com/mercadopago/sdk-go/pkg/preference"
)

type capturePrefs struct {
	preference.Client
	got preference.Request
}

func (c *capturePrefs) Create(_ context.Context, req preference.Request) (*preference.Response, error) {
	c.got = req
	return &preference.Response{ID: "pref", InitPoint: "https://mp.test/pay"}, nil
}

func TestPreferencePixExpiresWithReservation(t *testing.T) {
	fake := &capturePrefs{}
	c := &Client{prefs: fake}
	expires := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	if _, err := c.CreatePreference(context.Background(), PreferenceInput{
		OrderID: "o1", Number: 7, ExpiresAt: expires, ReturnURL: "https://site.test/pedido/o1",
		Items: []Item{{Title: "Booster", Qty: 1, UnitBRL: 10}},
		Payer: Payer{Name: "Cliente Teste", Phone: "5562999990001"},
	}); err != nil {
		t.Fatal(err)
	}
	r := fake.got
	if r.ExpirationDateTo == nil || !r.ExpirationDateTo.Equal(expires) {
		t.Fatalf("expiration_date_to = %v, quer %v", r.ExpirationDateTo, expires)
	}
	if r.DateOfExpiration == nil || !r.DateOfExpiration.Equal(expires) {
		t.Fatalf("date_of_expiration (Pix) = %v, quer %v", r.DateOfExpiration, expires)
	}
}
