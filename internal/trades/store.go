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
	"strings"
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

// Listing is the storefront state for one holding: its per-unit asking price and
// whether it is shown on the public catalog.
type Listing struct {
	AskBRL   float64
	Listed   bool
	ImageURL string
}

// SetListings applies storefront listing state to many trades in a single
// load/persist cycle. Unknown ids are ignored; returns the count updated.
func (s *Store) SetListings(updates map[string]Listing) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return 0, err
	}
	now := time.Now()
	n := 0
	for i := range all {
		u, ok := updates[all[i].ID]
		if !ok {
			continue
		}
		all[i].AskBRL = u.AskBRL
		all[i].Listed = u.Listed
		all[i].ImageURL = u.ImageURL
		all[i].UpdatedAt = now
		n++
	}
	if n == 0 {
		return 0, nil
	}
	if err := s.persist(all); err != nil {
		return 0, err
	}
	return n, nil
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

// Merge collapses two or more holdings into a single record. Quantity and
// shipping are summed; the per-unit buy price becomes the quantity-weighted
// average, so the combined cost basis equals the sum of the originals (portfolio
// P&L is unchanged). The oldest selected holding is the primary: it keeps its id
// (so its asking price, listed flag and any references survive) along with its
// card identity, store, price and listed state. The buy date becomes the
// earliest, notes are concatenated, and the lot counts as delivered only when
// every part was. Sold trades cannot be merged.
func (s *Store) Merge(ids []string) (Trade, error) {
	if len(ids) < 2 {
		return Trade{}, errors.New("select at least two holdings to merge")
	}
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Trade{}, err
	}
	parts := make([]Trade, 0, len(ids))
	for _, t := range all {
		if !want[t.ID] {
			continue
		}
		if t.Status == "sold" {
			return Trade{}, errors.New("cannot merge a sold trade")
		}
		parts = append(parts, t)
	}
	if len(parts) != len(want) {
		return Trade{}, ErrNotFound
	}

	primary := parts[0]
	for _, t := range parts[1:] {
		if t.CreatedAt.Before(primary.CreatedAt) {
			primary = t
		}
	}

	merged := primary
	totalQty := 0
	weightedBuy := 0.0
	shipping := 0.0
	delivered := true
	notes := make([]string, 0, len(parts))
	seenNotes := make(map[string]bool, len(parts))
	for _, t := range parts {
		qty := t.Qty
		if qty <= 0 {
			qty = 1
		}
		totalQty += qty
		weightedBuy += float64(qty) * t.BuyBRL
		shipping += t.ShippingBRL
		if !t.Delivered {
			delivered = false
		}
		if t.BuyDate != "" && (merged.BuyDate == "" || t.BuyDate < merged.BuyDate) {
			merged.BuyDate = t.BuyDate
		}
		if n := t.Notes; n != "" && !seenNotes[n] {
			seenNotes[n] = true
			notes = append(notes, n)
		}
	}

	merged.ID = primary.ID
	merged.Qty = totalQty
	merged.BuyBRL = round2(weightedBuy / float64(totalQty))
	merged.ShippingBRL = round2(shipping)
	merged.Delivered = delivered
	merged.CreatedAt = primary.CreatedAt
	merged.UpdatedAt = time.Now()
	if len(notes) > 0 {
		merged.Notes = strings.Join(notes, "; ")
	}

	out := make([]Trade, 0, len(all)-len(parts)+1)
	for _, t := range all {
		if t.ID == primary.ID {
			out = append(out, merged)
			continue
		}
		if want[t.ID] {
			continue
		}
		out = append(out, t)
	}
	if err := s.persist(out); err != nil {
		return Trade{}, err
	}
	return merged, nil
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
