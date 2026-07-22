package trades

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"opdeals/internal/logx"
)

var ErrNotFound = errors.New("trade not found")

type Store struct {
	path string
	mu   sync.RWMutex
	log  *logx.Logger
}

func NewStore(path string, log *logx.Logger) *Store {
	return &Store{path: path, log: log}
}

func (s *Store) load() ([]Trade, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Trade{}, nil
		}
		return nil, err
	}
	var out []Trade
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Trade) error {
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

func (s *Store) List() ([]Trade, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	return all, nil
}

func (s *Store) Add(t Trade) (Trade, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Trade{}, err
	}
	now := time.Now()
	t.ID = newID()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Status == "" {
		t.Status = "holding"
	}
	if t.Qty <= 0 {
		t.Qty = 1
	}
	all = append(all, t)
	if err := s.persist(all); err != nil {
		return Trade{}, err
	}
	return t, nil
}

func (s *Store) Update(id string, mut func(*Trade)) (Trade, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Trade{}, err
	}
	for i := range all {
		if all[i].ID == id {
			mut(&all[i])
			all[i].ID = id
			all[i].UpdatedAt = time.Now()
			if all[i].Qty <= 0 {
				all[i].Qty = 1
			}
			if err := s.persist(all); err != nil {
				return Trade{}, err
			}
			return all[i], nil
		}
	}
	return Trade{}, ErrNotFound
}

type Sale struct {
	Qty          int
	SellPrice    float64
	SellCurrency string
	SellDate     string
	Buyer        string
}

func (s *Store) Sell(id string, sale Sale) (Trade, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Trade{}, err
	}
	for i := range all {
		if all[i].ID != id {
			continue
		}
		now := time.Now()
		orig := &all[i]
		if orig.Qty <= 0 {
			orig.Qty = 1
		}
		if sale.Qty <= 0 || sale.Qty >= orig.Qty {
			orig.Status = "sold"
			orig.SellPrice = sale.SellPrice
			orig.SellCurrency = sale.SellCurrency
			orig.SellDate = sale.SellDate
			orig.Buyer = sale.Buyer
			orig.UpdatedAt = now
			result := *orig
			if err := s.persist(all); err != nil {
				return Trade{}, err
			}
			return result, nil
		}
		soldShip := round2(orig.ShippingBRL * float64(sale.Qty) / float64(orig.Qty))
		sold := *orig
		sold.ID = newID()
		sold.Qty = sale.Qty
		sold.ShippingBRL = soldShip
		sold.Status = "sold"
		sold.SellPrice = sale.SellPrice
		sold.SellCurrency = sale.SellCurrency
		sold.SellDate = sale.SellDate
		sold.Buyer = sale.Buyer
		sold.UpdatedAt = now
		orig.Qty -= sale.Qty
		orig.ShippingBRL = round2(orig.ShippingBRL - soldShip)
		orig.UpdatedAt = now
		all = append(all, sold)
		if err := s.persist(all); err != nil {
			return Trade{}, err
		}
		return sold, nil
	}
	return Trade{}, ErrNotFound
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
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
	for _, t := range all {
		if t.ID == id {
			found = true
			continue
		}
		out = append(out, t)
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
