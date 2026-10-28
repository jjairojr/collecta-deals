package orders

import (
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"

	"opdeals/internal/logx"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "orders.json"), logx.New(io.Discard))
}

func TestCreateNumbersAndReserves(t *testing.T) {
	s := testStore(t)
	first, err := s.Create(func(reserved map[string]int, _ []Order) (Order, error) {
		if len(reserved) != 0 {
			t.Fatalf("reserved = %v want empty", reserved)
		}
		return Order{Lines: []Line{{SKU: "a", Qty: 2}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Number != 1 || first.Status != StatusPending || first.ID == "" {
		t.Fatalf("unexpected order: %+v", first)
	}
	if first.ExpiresAt.Sub(first.CreatedAt) != ReservationTTL {
		t.Fatalf("expires = %v", first.ExpiresAt)
	}
	second, err := s.Create(func(reserved map[string]int, _ []Order) (Order, error) {
		if reserved["a"] != 2 {
			t.Fatalf("reserved = %v want a:2", reserved)
		}
		return Order{Lines: []Line{{SKU: "a", Qty: 1}, {SKU: "b", Qty: 3}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Number != 2 {
		t.Fatalf("number = %d want 2", second.Number)
	}
	got, err := s.Reserved(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != 3 || got["b"] != 3 {
		t.Fatalf("reserved = %v", got)
	}
	if _, err := s.Update(first.ID, func(o *Order) { o.Status = StatusPaid }); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Reserved(time.Now())
	if got["a"] != 1 {
		t.Fatalf("reserved after paid = %v", got)
	}
	got, _ = s.Reserved(time.Now().Add(ReservationTTL + time.Second))
	if len(got) != 0 {
		t.Fatalf("reserved after expiry = %v", got)
	}
}

func TestCreateBuildErrorPersistsNothing(t *testing.T) {
	s := testStore(t)
	want := errors.New("no stock")
	if _, err := s.Create(func(map[string]int, []Order) (Order, error) { return Order{}, want }); !errors.Is(err, want) {
		t.Fatalf("err = %v", err)
	}
	all, err := s.List()
	if err != nil || len(all) != 0 {
		t.Fatalf("list = %v %v", all, err)
	}
}

func TestTransactRollsBackOnError(t *testing.T) {
	s := testStore(t)
	o, _ := s.Create(func(map[string]int, []Order) (Order, error) { return Order{}, nil })
	if _, err := s.Transact(o.ID, func(o *Order) error {
		o.Status = StatusPaid
		return errors.New("boom")
	}); err == nil {
		t.Fatal("expected error")
	}
	got, _ := s.Get(o.ID)
	if got.Status != StatusPending {
		t.Fatalf("status = %s want pending", got.Status)
	}
	if _, err := s.Transact("missing", func(*Order) error { return nil }); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestEffectiveStatus(t *testing.T) {
	now := time.Now()
	o := Order{Status: StatusPending, ExpiresAt: now.Add(-time.Second)}
	if EffectiveStatus(o, now) != StatusExpired {
		t.Fatal("expected expired")
	}
	o.Status = StatusPaid
	if EffectiveStatus(o, now) != StatusPaid {
		t.Fatal("paid must not expire")
	}
}

func TestSetStatus(t *testing.T) {
	now := time.Now()
	o := Order{Status: StatusOversold}
	if err := SetStatus(&o, StatusDelivered, now); err != nil {
		t.Fatal(err)
	}
	if o.Status != StatusDelivered || o.PaidAt.IsZero() || o.DeliveredAt.IsZero() {
		t.Fatalf("order = %+v", o)
	}
	if err := SetStatus(&o, StatusPaid, now); err != nil {
		t.Fatal(err)
	}
	if !o.DeliveredAt.IsZero() || !o.ShippedAt.IsZero() {
		t.Fatalf("rollback kept stamps: %+v", o)
	}
	if err := SetStatus(&o, StatusCancelled, now); !errors.Is(err, ErrTransition) {
		t.Fatalf("cancel paid: %v", err)
	}
	p := Order{Status: StatusPending, ExpiresAt: now.Add(time.Hour)}
	if err := SetStatus(&p, StatusShipped, now); !errors.Is(err, ErrTransition) {
		t.Fatalf("ship pending: %v", err)
	}
	if err := SetStatus(&p, StatusCancelled, now); err != nil || p.Status != StatusCancelled {
		t.Fatalf("cancel pending: %v %+v", err, p)
	}
}
