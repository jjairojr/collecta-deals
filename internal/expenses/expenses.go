// Package expenses keeps the business-wide expense ledger: one-off purchases
// (frete, embalagens, taxas…) and recurring monthly costs (aluguel, assinaturas).
// Expenses belong to no game, so there is a single ledger for the whole app.
package expenses

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"opdeals/internal/logx"
)

var ErrNotFound = errors.New("expense not found")

// Expense is one ledger entry. Recurring entries count every month from
// Date's month until EndDate's month (empty EndDate = still active); one-off
// entries count only in Date's month.
type Expense struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"` // YYYY-MM-DD; start month for recurring entries
	Recurring   bool    `json:"recurring,omitempty"`
	EndDate     string  `json:"endDate,omitempty"` // YYYY-MM-DD; last month a recurring entry counts
	Description string  `json:"description"`
	Category    string  `json:"category,omitempty"`
	AmountBRL   float64 `json:"amountBRL"`
	Store       string  `json:"store,omitempty"`
	Notes       string  `json:"notes,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Store struct {
	path string
	mu   sync.RWMutex
	log  *logx.Logger
}

func NewStore(path string, log *logx.Logger) *Store {
	return &Store{path: path, log: log}
}

func (s *Store) load() ([]Expense, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Expense{}, nil
		}
		return nil, err
	}
	var out []Expense
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Expense) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func normalize(e *Expense) {
	if e.Date == "" {
		e.Date = e.CreatedAt.Format("2006-01-02")
	}
	if !e.Recurring {
		e.EndDate = ""
	}
}

func (s *Store) List() ([]Expense, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Date != all[j].Date {
			return all[i].Date > all[j].Date
		}
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	return all, nil
}

func (s *Store) Add(e Expense) (Expense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Expense{}, err
	}
	now := time.Now()
	e.ID = newID()
	e.CreatedAt = now
	e.UpdatedAt = now
	normalize(&e)
	all = append(all, e)
	if err := s.persist(all); err != nil {
		return Expense{}, err
	}
	return e, nil
}

func (s *Store) Update(id string, e Expense) (Expense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Expense{}, err
	}
	for i := range all {
		if all[i].ID == id {
			e.ID = id
			e.CreatedAt = all[i].CreatedAt
			e.UpdatedAt = time.Now()
			normalize(&e)
			all[i] = e
			if err := s.persist(all); err != nil {
				return Expense{}, err
			}
			return all[i], nil
		}
	}
	return Expense{}, ErrNotFound
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return err
	}
	out := all[:0]
	found := false
	for _, e := range all {
		if e.ID == id {
			found = true
			continue
		}
		out = append(out, e)
	}
	if !found {
		return ErrNotFound
	}
	return s.persist(out)
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return hex.EncodeToString(b[:])
}
