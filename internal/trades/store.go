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

var ErrAlreadySold = errors.New("trade already sold")

var ErrNotSold = errors.New("trade is not sold")

var ErrSiteSale = errors.New("venda do site: desfaça pelo pedido em Pedidos")

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
	if t.Slug == "" {
		t.Slug = SlugBase(t)
	}
	all = append(all, t)
	if err := s.persist(all); err != nil {
		return Trade{}, err
	}
	return t, nil
}

// Take removes every trade matching pred and returns them unchanged, so records
// can move to another ledger keeping their ids, timestamps and listing state.
func (s *Store) Take(pred func(Trade) bool) ([]Trade, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	taken := make([]Trade, 0)
	kept := make([]Trade, 0, len(all))
	for _, t := range all {
		if pred(t) {
			taken = append(taken, t)
			continue
		}
		kept = append(kept, t)
	}
	if len(taken) == 0 {
		return nil, nil
	}
	if err := s.persist(kept); err != nil {
		return nil, err
	}
	return taken, nil
}

// Append writes trades exactly as given — the counterpart of Take when moving
// records between ledgers. Add is what creates a new trade.
func (s *Store) Append(ts []Trade) error {
	if len(ts) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return err
	}
	return s.persist(append(all, ts...))
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
			slug := all[i].Slug
			if slug == "" {
				slug = SlugBase(all[i])
			}
			mut(&all[i])
			all[i].ID = id
			all[i].Slug = slug
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
	Featured bool
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
		all[i].Featured = u.Featured
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

// LigaState is what a holding looks like on the LigaMagic store: whether it is
// registered there and, if so, the quantity and unit price actually sent.
type LigaState struct {
	Listed   bool
	Qty      int
	PriceBRL float64
}

// SetLigaState records LigaMagic registration state for many trades in a single
// load/persist cycle. Unknown ids are ignored; returns the count updated.
//
// Deliberately separate from SetListings: that one owns the public storefront
// fields and would overwrite price and visibility, so folding the Liga flag into
// it would let a flag toggle clobber a price.
func (s *Store) SetLigaState(updates map[string]LigaState) (int, error) {
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
		all[i].LigaListed = u.Listed
		if u.Listed {
			all[i].LigaQty = u.Qty
			all[i].LigaPriceBRL = u.PriceBRL
			all[i].LigaAt = now
		} else {
			all[i].LigaQty = 0
			all[i].LigaPriceBRL = 0
			all[i].LigaAt = time.Time{}
		}
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
	Channel      string
	WriteOff     string
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
		if all[i].Status == "sold" {
			return Trade{}, ErrAlreadySold
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
			orig.Channel = sale.Channel
			orig.WriteOff = sale.WriteOff
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
		sold.Channel = sale.Channel
		sold.WriteOff = sale.WriteOff
		sold.SplitFrom = orig.ID
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

type Unsold struct {
	Sale     Trade
	Restored Trade
}

func (s *Store) Unsell(id string) (Unsold, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Unsold{}, err
	}
	at := -1
	for i := range all {
		if all[i].ID == id {
			at = i
			break
		}
	}
	if at < 0 {
		return Unsold{}, ErrNotFound
	}
	sale := all[at]
	if sale.Status != "sold" {
		return Unsold{}, ErrNotSold
	}
	if ChannelOf(sale) == ChannelSite {
		return Unsold{}, ErrSiteSale
	}
	now := time.Now()
	back := sale
	back.Status = "holding"
	back.SellPrice = 0
	back.SellCurrency = ""
	back.SellDate = ""
	back.Buyer = ""
	back.Channel = ""
	back.WriteOff = ""
	back.SemLastro = false
	back.SplitFrom = ""
	back.UpdatedAt = now
	if back.Qty <= 0 {
		back.Qty = 1
	}

	origin := originOf(all, sale)
	if origin < 0 {
		all[at] = back
		if err := s.persist(all); err != nil {
			return Unsold{}, err
		}
		return Unsold{Sale: sale, Restored: back}, nil
	}
	o := &all[origin]
	qty := o.Qty + back.Qty
	o.BuyBRL = round2((float64(o.Qty)*o.BuyBRL + float64(back.Qty)*back.BuyBRL) / float64(qty))
	o.Qty = qty
	o.ShippingBRL = round2(o.ShippingBRL + back.ShippingBRL)
	o.UpdatedAt = now
	restored := *o
	out := append(all[:at:at], all[at+1:]...)
	if err := s.persist(out); err != nil {
		return Unsold{}, err
	}
	return Unsold{Sale: sale, Restored: restored}, nil
}

func originOf(all []Trade, sale Trade) int {
	if sale.SplitFrom != "" {
		for i := range all {
			if all[i].ID == sale.SplitFrom && all[i].Status != "sold" {
				return i
			}
		}
	}
	key := SKU("", sale)
	best := -1
	for i := range all {
		t := all[i]
		if t.ID == sale.ID || t.Status == "sold" || t.PurchaseID != sale.PurchaseID || SKU("", t) != key {
			continue
		}
		if best < 0 || t.CreatedAt.Before(all[best].CreatedAt) {
			best = i
		}
	}
	return best
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
		if t.ID != primary.ID {
			merged.Photos = append(merged.Photos, t.Photos...)
		}
	}
	if merged.ImageURL == "" && len(merged.Photos) > 0 {
		merged.ImageURL = merged.Photos[0]
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
