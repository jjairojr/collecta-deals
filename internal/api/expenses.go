package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"opdeals/internal/expenses"
)

func validExpense(e expenses.Expense) bool {
	return e.Description != "" && e.AmountBRL > 0
}

func (s *Server) handleExpensesList(w http.ResponseWriter, r *http.Request) {
	if s.expenses == nil {
		http.Error(w, "expenses store unavailable", http.StatusServiceUnavailable)
		return
	}
	all, err := s.expenses.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string][]expenses.Expense{"expenses": all})
}

func (s *Server) handleExpensesCreate(w http.ResponseWriter, r *http.Request) {
	if s.expenses == nil {
		http.Error(w, "expenses store unavailable", http.StatusServiceUnavailable)
		return
	}
	var e expenses.Expense
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !validExpense(e) {
		http.Error(w, "description and a positive amount are required", http.StatusBadRequest)
		return
	}
	created, err := s.expenses.Add(e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, created)
}

func (s *Server) handleExpensesUpdate(w http.ResponseWriter, r *http.Request) {
	if s.expenses == nil {
		http.Error(w, "expenses store unavailable", http.StatusServiceUnavailable)
		return
	}
	var e expenses.Expense
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !validExpense(e) {
		http.Error(w, "description and a positive amount are required", http.StatusBadRequest)
		return
	}
	updated, err := s.expenses.Update(r.PathValue("id"), e)
	if errors.Is(err, expenses.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, updated)
}

func (s *Server) handleExpensesDelete(w http.ResponseWriter, r *http.Request) {
	if s.expenses == nil {
		http.Error(w, "expenses store unavailable", http.StatusServiceUnavailable)
		return
	}
	err := s.expenses.Delete(r.PathValue("id"))
	if errors.Is(err, expenses.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}
