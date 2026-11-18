package kits

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound  = errors.New("kit não encontrado")
	ErrDuplicate = errors.New("já existe um kit com essa quantidade para o produto")
	ErrInvalid   = errors.New("kit inválido")
)

const (
	MinUnits = 2
	MaxUnits = 100
)

type Kit struct {
	ID        string    `json:"id"`
	BaseSKU   string    `json:"baseSKU"`
	Units     int       `json:"units"`
	AskBRL    float64   `json:"askBRL"`
	CardBRL   float64   `json:"cardBRL,omitempty"`
	Listed    bool      `json:"listed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func SKU(baseSKU string, units int) string {
	return baseSKU + "#x" + strconv.Itoa(units)
}

func (k Kit) SKU() string {
	return SKU(k.BaseSKU, k.Units)
}

func (k Kit) CardPrice() float64 {
	if k.CardBRL > 0 {
		return k.CardBRL
	}
	return k.AskBRL
}

func Validate(k Kit) error {
	switch {
	case strings.TrimSpace(k.BaseSKU) == "":
		return fmt.Errorf("%w: produto base obrigatório", ErrInvalid)
	case k.Units < MinUnits || k.Units > MaxUnits:
		return fmt.Errorf("%w: quantidade precisa estar entre %d e %d", ErrInvalid, MinUnits, MaxUnits)
	case k.AskBRL <= 0:
		return fmt.Errorf("%w: preço precisa ser maior que zero", ErrInvalid)
	case k.CardBRL < 0:
		return fmt.Errorf("%w: preço no cartão não pode ser negativo", ErrInvalid)
	}
	return nil
}

type Store struct {
	path string
	mu   sync.RWMutex
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) load() ([]Kit, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Kit{}, nil
		}
		return nil, err
	}
	var out []Kit
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Kit) error {
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

func (s *Store) List() ([]Kit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].BaseSKU != all[j].BaseSKU {
			return all[i].BaseSKU < all[j].BaseSKU
		}
		return all[i].Units < all[j].Units
	})
	return all, nil
}

func (s *Store) Add(k Kit) (Kit, error) {
	if err := Validate(k); err != nil {
		return Kit{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Kit{}, err
	}
	for _, e := range all {
		if e.BaseSKU == k.BaseSKU && e.Units == k.Units {
			return Kit{}, ErrDuplicate
		}
	}
	now := time.Now()
	k.ID = newID()
	k.CreatedAt = now
	k.UpdatedAt = now
	all = append(all, k)
	if err := s.persist(all); err != nil {
		return Kit{}, err
	}
	return k, nil
}

func (s *Store) Update(id string, mut func(*Kit)) (Kit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Kit{}, err
	}
	for i := range all {
		if all[i].ID != id {
			continue
		}
		next := all[i]
		mut(&next)
		next.ID = id
		next.BaseSKU = all[i].BaseSKU
		next.Units = all[i].Units
		next.CreatedAt = all[i].CreatedAt
		next.UpdatedAt = time.Now()
		if err := Validate(next); err != nil {
			return Kit{}, err
		}
		all[i] = next
		if err := s.persist(all); err != nil {
			return Kit{}, err
		}
		return next, nil
	}
	return Kit{}, ErrNotFound
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return err
	}
	for i := range all {
		if all[i].ID == id {
			return s.persist(append(all[:i], all[i+1:]...))
		}
	}
	return ErrNotFound
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return hex.EncodeToString(b[:])
}
