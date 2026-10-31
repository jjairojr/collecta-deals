package syncqueue

import (
	"errors"
	"path/filepath"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "syncqueue.json"))
}

func TestAddStartsPendingAndListFilters(t *testing.T) {
	s := newStore(t)
	a, err := s.Add(Task{Game: "onepiece", TradeID: "t1", Name: "Nami", Target: TargetLiga, Action: ActionRemove, Qty: 2})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == "" || a.Status != StatusPending || a.CreatedAt.IsZero() {
		t.Fatalf("added = %+v", a)
	}
	if _, err := s.Add(Task{Game: "pokemon", TradeID: "t2", Name: "Pikachu", Target: TargetLiga, Action: ActionRemove, Qty: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetStatus(a.ID, StatusDone, ""); err != nil {
		t.Fatal(err)
	}
	pending, err := s.List(StatusPending)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].TradeID != "t2" {
		t.Fatalf("pending = %+v", pending)
	}
	all, err := s.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("all = %d, want 2", len(all))
	}
}

func TestSetStatusTracksErrorAndDoneAt(t *testing.T) {
	s := newStore(t)
	a, err := s.Add(Task{TradeID: "t1", Target: TargetLiga, Action: ActionRemove, Qty: 1})
	if err != nil {
		t.Fatal(err)
	}
	failed, err := s.SetStatus(a.ID, StatusError, "oferta não achada")
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != StatusError || failed.Error != "oferta não achada" || !failed.DoneAt.IsZero() {
		t.Fatalf("failed = %+v", failed)
	}
	retried, err := s.SetStatus(a.ID, StatusPending, "")
	if err != nil {
		t.Fatal(err)
	}
	if retried.Status != StatusPending || retried.Error != "" {
		t.Fatalf("retried = %+v", retried)
	}
	done, err := s.SetStatus(a.ID, StatusDone, "")
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != StatusDone || done.DoneAt.IsZero() {
		t.Fatalf("done = %+v", done)
	}
}

func TestSetStatusUnknownID(t *testing.T) {
	s := newStore(t)
	if _, err := s.SetStatus("nope", StatusDone, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestSetStatusRejectsUnknownStatus(t *testing.T) {
	s := newStore(t)
	a, err := s.Add(Task{TradeID: "t1", Target: TargetLiga, Action: ActionRemove, Qty: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetStatus(a.ID, "weird", ""); err == nil {
		t.Fatal("unknown status accepted")
	}
}
