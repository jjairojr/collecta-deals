package quotes

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

var ErrNotFound = errors.New("quote not found")

type Item struct {
	Number     string  `json:"number"`
	Name       string  `json:"name"`
	Set        string  `json:"set"`
	Variant    string  `json:"variant,omitempty"`
	Qty        int     `json:"qty"`
	UnitBRL    float64 `json:"unitBRL"`
	Pct        float64 `json:"pct,omitempty"`
	MarketUSD  float64 `json:"marketUSD,omitempty"`
	LigaLowBRL float64 `json:"ligaLowBRL,omitempty"`
	LigaAvgBRL float64 `json:"ligaAvgBRL,omitempty"`
	LigaURL    string  `json:"ligaUrl,omitempty"`
	ProductID  int     `json:"productID,omitempty"`
}

type Quote struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Pct       float64   `json:"pct"`
	Market    string    `json:"market,omitempty"`
	FXRate    float64   `json:"fxRate,omitempty"`
	Items     []Item    `json:"items"`
	Notes     string    `json:"notes,omitempty"`
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

func (s *Store) load() ([]Quote, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Quote{}, nil
		}
		return nil, err
	}
	var out []Quote
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Quote) error {
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

func normalize(q *Quote) {
	if q.Name == "" {
		q.Name = q.CreatedAt.Format("02/01/2006")
	}
	if q.Pct <= 0 {
		q.Pct = 60
	}
	if q.Market != "liga" {
		q.Market = "tcg"
	}
	if q.Items == nil {
		q.Items = []Item{}
	}
	for i := range q.Items {
		if q.Items[i].Qty < 1 {
			q.Items[i].Qty = 1
		}
		if q.Items[i].Pct < 0 {
			q.Items[i].Pct = 0
		}
	}
}

func (s *Store) List() ([]Quote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].UpdatedAt.After(all[j].UpdatedAt)
	})
	return all, nil
}

func (s *Store) Add(q Quote) (Quote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Quote{}, err
	}
	now := time.Now()
	q.ID = newID()
	q.CreatedAt = now
	q.UpdatedAt = now
	normalize(&q)
	all = append(all, q)
	if err := s.persist(all); err != nil {
		return Quote{}, err
	}
	return q, nil
}

func (s *Store) Update(id string, q Quote) (Quote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Quote{}, err
	}
	for i := range all {
		if all[i].ID == id {
			q.ID = id
			q.CreatedAt = all[i].CreatedAt
			q.UpdatedAt = time.Now()
			normalize(&q)
			all[i] = q
			if err := s.persist(all); err != nil {
				return Quote{}, err
			}
			return all[i], nil
		}
	}
	return Quote{}, ErrNotFound
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
	for _, q := range all {
		if q.ID == id {
			found = true
			continue
		}
		out = append(out, q)
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
