package kits

import (
	"errors"
	"path/filepath"
	"testing"
)

const packSKU = "sealed|pokemon|(jap) booster 30 anos|||"

func TestKitSKUIsDistinctPerSize(t *testing.T) {
	if SKU(packSKU, 2) == SKU(packSKU, 3) || SKU(packSKU, 2) == packSKU {
		t.Fatalf("kit skus must differ from each other and from the base")
	}
}

func TestAddRejectsInvalidAndDuplicate(t *testing.T) {
	st := NewStore(filepath.Join(t.TempDir(), "kits.json"))
	for name, k := range map[string]Kit{
		"single unit": {BaseSKU: packSKU, Units: 1, AskBRL: 10},
		"no price":    {BaseSKU: packSKU, Units: 2},
		"no base":     {Units: 2, AskBRL: 10},
	} {
		if _, err := st.Add(k); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
	if _, err := st.Add(Kit{BaseSKU: packSKU, Units: 2, AskBRL: 50, Listed: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Add(Kit{BaseSKU: packSKU, Units: 2, AskBRL: 45}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("err = %v, want ErrDuplicate", err)
	}
	if _, err := st.Add(Kit{BaseSKU: packSKU, Units: 3, AskBRL: 70}); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateKeepsIdentityAndDeleteRemoves(t *testing.T) {
	st := NewStore(filepath.Join(t.TempDir(), "kits.json"))
	k, err := st.Add(Kit{BaseSKU: packSKU, Units: 3, AskBRL: 70})
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.Update(k.ID, func(e *Kit) {
		e.AskBRL = 65
		e.CardBRL = 72
		e.Listed = true
		e.Units = 9
		e.BaseSKU = "other"
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Units != 3 || got.BaseSKU != packSKU || got.AskBRL != 65 || got.CardPrice() != 72 || !got.Listed {
		t.Fatalf("update = %+v", got)
	}
	if _, err := st.Update(k.ID, func(e *Kit) { e.AskBRL = 0 }); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if err := st.Delete(k.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.Delete(k.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	all, err := st.List()
	if err != nil || len(all) != 0 {
		t.Fatalf("list = %v, %v", all, err)
	}
}
