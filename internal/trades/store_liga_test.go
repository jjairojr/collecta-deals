package trades

import "testing"

func TestSetLigaStateRecordsWhatWasSent(t *testing.T) {
	s := newTestStore(t)
	a, err := s.Add(Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 3, BuyBRL: 100, AskBRL: 250, Listed: true})
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	n, err := s.SetLigaState(map[string]LigaState{a.ID: {Listed: true, Qty: 3, PriceBRL: 250}})
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	if n != 1 {
		t.Fatalf("updated: got %d want 1", n)
	}

	all, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := all[0]
	if !got.LigaListed || got.LigaQty != 3 || got.LigaPriceBRL != 250 {
		t.Fatalf("liga state not recorded: %+v", got)
	}
	if got.LigaAt.IsZero() {
		t.Fatal("LigaAt should be stamped when the holding is marked")
	}
	if got.AskBRL != 250 || !got.Listed {
		t.Fatalf("storefront fields must be untouched: %+v", got)
	}
}

func TestSetLigaStateUnmarkClearsWhatWasSent(t *testing.T) {
	s := newTestStore(t)
	a, err := s.Add(Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 3, AskBRL: 250})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := s.SetLigaState(map[string]LigaState{a.ID: {Listed: true, Qty: 3, PriceBRL: 250}}); err != nil {
		t.Fatalf("mark: %v", err)
	}
	if _, err := s.SetLigaState(map[string]LigaState{a.ID: {Listed: false}}); err != nil {
		t.Fatalf("unmark: %v", err)
	}

	all, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := all[0]
	if got.LigaListed || got.LigaQty != 0 || got.LigaPriceBRL != 0 || !got.LigaAt.IsZero() {
		t.Fatalf("unmarking must clear the recorded state, got %+v", got)
	}
}

// A holding left out of the payload keeps its recorded Liga state: the Estoque
// tab only sends rows whose flag changed, so an unrelated price edit must not
// refresh what we believe is on the store.
func TestSetLigaStateIgnoresIDsNotInThePayload(t *testing.T) {
	s := newTestStore(t)
	a, err := s.Add(Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 3, AskBRL: 250})
	if err != nil {
		t.Fatalf("add a: %v", err)
	}
	b, err := s.Add(Trade{Number: "OP16-081", Name: "Shanks", Set: "OP16", Qty: 1, AskBRL: 90})
	if err != nil {
		t.Fatalf("add b: %v", err)
	}
	if _, err := s.SetLigaState(map[string]LigaState{a.ID: {Listed: true, Qty: 3, PriceBRL: 250}}); err != nil {
		t.Fatalf("mark: %v", err)
	}

	n, err := s.SetLigaState(map[string]LigaState{b.ID: {Listed: true, Qty: 1, PriceBRL: 90}})
	if err != nil {
		t.Fatalf("mark b: %v", err)
	}
	if n != 1 {
		t.Fatalf("updated: got %d want 1", n)
	}

	all, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, tr := range all {
		if tr.ID == a.ID && (tr.LigaQty != 3 || tr.LigaPriceBRL != 250) {
			t.Fatalf("untouched holding lost its recorded state: %+v", tr)
		}
	}
}
