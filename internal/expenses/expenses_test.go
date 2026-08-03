package expenses

import (
	"path/filepath"
	"testing"

	"opdeals/internal/logx"
)

func TestStoreCRUD(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "expenses.json"), logx.New(nil))

	one, err := s.Add(Expense{Date: "2026-08-01", Description: "Envelopes", AmountBRL: 45})
	if err != nil {
		t.Fatalf("add one-off: %v", err)
	}
	fixed, err := s.Add(Expense{Date: "2026-07-01", Recurring: true, Description: "Assinatura Liga", AmountBRL: 99.9})
	if err != nil {
		t.Fatalf("add recurring: %v", err)
	}

	all, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("list len = %d, want 2", len(all))
	}
	if all[0].ID != one.ID {
		t.Fatalf("list order: newest date first, got %q", all[0].Description)
	}

	fixed.EndDate = "2026-09-30"
	upd, err := s.Update(fixed.ID, fixed)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if upd.EndDate != "2026-09-30" {
		t.Fatalf("endDate = %q, want 2026-09-30", upd.EndDate)
	}

	one.Recurring = false
	one.EndDate = "2026-12-31"
	upd, err = s.Update(one.ID, one)
	if err != nil {
		t.Fatalf("update one-off: %v", err)
	}
	if upd.EndDate != "" {
		t.Fatalf("one-off endDate should be cleared, got %q", upd.EndDate)
	}

	if err := s.Delete(one.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := s.Delete(one.ID); err != ErrNotFound {
		t.Fatalf("second delete err = %v, want ErrNotFound", err)
	}
	all, err = s.List()
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(all) != 1 || all[0].ID != fixed.ID {
		t.Fatalf("after delete want only the recurring entry, got %d", len(all))
	}
}
