package promos

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestSeedWhenFileMissing(t *testing.T) {
	st := NewStore(filepath.Join(t.TempDir(), "promos.json"))
	p, err := st.Get("bemvindo")
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != KindPercent || p.PercentBps != 500 || !p.FirstOrderOnly || !p.Active {
		t.Fatalf("seed = %+v", p)
	}
}

func TestDiscountPercentAndFixedCapped(t *testing.T) {
	if got := DiscountCents(50066, Promo{Kind: KindPercent, PercentBps: 500}); got != 2503 {
		t.Fatalf("percent = %d", got)
	}
	if got := DiscountCents(10000, Promo{Kind: KindFixed, ValueCents: 2500}); got != 2500 {
		t.Fatalf("fixed = %d", got)
	}
	if got := DiscountCents(1000, Promo{Kind: KindFixed, ValueCents: 2500}); got != 1000 {
		t.Fatalf("fixed capped = %d", got)
	}
}

func TestAddValidatesAndRejectsDuplicates(t *testing.T) {
	st := NewStore(filepath.Join(t.TempDir(), "promos.json"))
	if _, err := st.Add(Promo{Code: "natal 10", Kind: KindPercent, PercentBps: 1000, Active: true}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("space in code: %v", err)
	}
	if _, err := st.Add(Promo{Code: "X10", Kind: KindFixed, Active: true}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("fixed without value: %v", err)
	}
	if _, err := st.Add(Promo{Code: "X10", Kind: KindPercent, PercentBps: 1000, ValidFrom: "2026-12-10", ValidUntil: "2026-12-01"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("until before from: %v", err)
	}
	p, err := st.Add(Promo{Code: " natal10 ", Kind: KindPercent, PercentBps: 1000, Active: true})
	if err != nil || p.Code != "NATAL10" {
		t.Fatalf("add = %+v %v", p, err)
	}
	if _, err := st.Add(Promo{Code: "NATAL10", Kind: KindPercent, PercentBps: 500}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate: %v", err)
	}
	all, err := st.List()
	if err != nil || len(all) != 2 {
		t.Fatalf("list = %+v %v", all, err)
	}
}
