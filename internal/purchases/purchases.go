// Package purchases keeps the business-wide ledger of buy orders: the header of
// a purchase (date, seller, freight, notes) that one or more holdings were
// bought under. A single order routinely mixes games — the same seller ships
// One Piece and Pokémon in one box — so orders belong to no game and live in a
// single ledger, while the items themselves stay in each game's trade ledger,
// pointing back here through Trade.PurchaseID.
package purchases

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

var ErrNotFound = errors.New("purchase not found")

// Purchase is one buy order. ShippingBRL is the freight charged for the order as
// a whole; per-item freight stays on the item, so an order's real cost is the
// items' cost plus this.
type Purchase struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"` // YYYY-MM-DD
	Store       string  `json:"store,omitempty"`
	ShippingBRL float64 `json:"shippingBRL,omitempty"`
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

func (s *Store) load() ([]Purchase, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Purchase{}, nil
		}
		return nil, err
	}
	var out []Purchase
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Purchase) error {
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

// List returns every order, newest buy date first; orders with no date fall to
// the end, newest created first among ties.
func (s *Store) List() ([]Purchase, error) {
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

func (s *Store) Add(p Purchase) (Purchase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Purchase{}, err
	}
	now := time.Now()
	p.ID = newID()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Date == "" {
		p.Date = now.Format("2006-01-02")
	}
	all = append(all, p)
	if err := s.persist(all); err != nil {
		return Purchase{}, err
	}
	return p, nil
}

func (s *Store) Update(id string, in Purchase) (Purchase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Purchase{}, err
	}
	for i := range all {
		if all[i].ID != id {
			continue
		}
		in.ID = id
		in.CreatedAt = all[i].CreatedAt
		in.UpdatedAt = time.Now()
		if in.Date == "" {
			in.Date = all[i].Date
		}
		all[i] = in
		if err := s.persist(all); err != nil {
			return Purchase{}, err
		}
		return in, nil
	}
	return Purchase{}, ErrNotFound
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
	for _, p := range all {
		if p.ID == id {
			found = true
			continue
		}
		out = append(out, p)
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
