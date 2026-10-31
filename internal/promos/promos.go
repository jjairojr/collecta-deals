package promos

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound  = errors.New("cupom não encontrado")
	ErrDuplicate = errors.New("já existe um cupom com esse código")
	ErrInvalid   = errors.New("cupom inválido")
)

const (
	KindPercent = "percent"
	KindFixed   = "fixed"
)

var codeRe = regexp.MustCompile(`^[A-Z0-9]{3,30}$`)

type Promo struct {
	Code           string    `json:"code"`
	Kind           string    `json:"kind"`
	PercentBps     int64     `json:"percentBps,omitempty"`
	ValueCents     int64     `json:"valueCents,omitempty"`
	FirstOrderOnly bool      `json:"firstOrderOnly,omitempty"`
	MinItemsCents  int64     `json:"minItemsCents,omitempty"`
	ValidFrom      string    `json:"validFrom,omitempty"`
	ValidUntil     string    `json:"validUntil,omitempty"`
	MaxUses        int       `json:"maxUses,omitempty"`
	Active         bool      `json:"active"`
	Notes          string    `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func Seed() []Promo {
	return []Promo{{Code: "BEMVINDO", Kind: KindPercent, PercentBps: 500, FirstOrderOnly: true, Active: true}}
}

func NormalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func DiscountCents(itemsCents int64, p Promo) int64 {
	if p.Kind == KindFixed {
		return min(p.ValueCents, itemsCents)
	}
	return itemsCents * p.PercentBps / 10000
}

func Validate(p Promo) error {
	if !codeRe.MatchString(p.Code) {
		return errors.New("código precisa ter de 3 a 30 letras ou números, sem espaço")
	}
	switch p.Kind {
	case KindPercent:
		if p.PercentBps <= 0 || p.PercentBps > 10000 {
			return errors.New("percentual precisa estar entre 0,01% e 100%")
		}
	case KindFixed:
		if p.ValueCents <= 0 {
			return errors.New("valor do desconto precisa ser maior que zero")
		}
	default:
		return errors.New("tipo precisa ser percent ou fixed")
	}
	if p.MinItemsCents < 0 || p.MaxUses < 0 {
		return errors.New("mínimo e limite de usos não podem ser negativos")
	}
	for _, d := range []string{p.ValidFrom, p.ValidUntil} {
		if d == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", d); err != nil {
			return errors.New("datas de validade precisam estar no formato AAAA-MM-DD")
		}
	}
	if p.ValidFrom != "" && p.ValidUntil != "" && p.ValidUntil < p.ValidFrom {
		return errors.New("fim da validade antes do início")
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

func (s *Store) load() ([]Promo, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Seed(), nil
		}
		return nil, err
	}
	var out []Promo
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Promo) error {
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

func (s *Store) List() ([]Promo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Code < all[j].Code })
	return all, nil
}

func (s *Store) Get(code string) (Promo, error) {
	all, err := s.List()
	if err != nil {
		return Promo{}, err
	}
	code = NormalizeCode(code)
	for _, p := range all {
		if p.Code == code {
			return p, nil
		}
	}
	return Promo{}, ErrNotFound
}

func (s *Store) Add(p Promo) (Promo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.Code = NormalizeCode(p.Code)
	if err := Validate(p); err != nil {
		return Promo{}, errors.Join(ErrInvalid, err)
	}
	all, err := s.load()
	if err != nil {
		return Promo{}, err
	}
	for _, x := range all {
		if x.Code == p.Code {
			return Promo{}, ErrDuplicate
		}
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	all = append(all, p)
	if err := s.persist(all); err != nil {
		return Promo{}, err
	}
	return p, nil
}

func (s *Store) Update(code string, mutate func(*Promo)) (Promo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Promo{}, err
	}
	code = NormalizeCode(code)
	for i := range all {
		if all[i].Code != code {
			continue
		}
		next := all[i]
		mutate(&next)
		next.Code = all[i].Code
		next.CreatedAt = all[i].CreatedAt
		if err := Validate(next); err != nil {
			return Promo{}, errors.Join(ErrInvalid, err)
		}
		next.UpdatedAt = time.Now()
		all[i] = next
		if err := s.persist(all); err != nil {
			return Promo{}, err
		}
		return next, nil
	}
	return Promo{}, ErrNotFound
}

func (s *Store) Delete(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return err
	}
	code = NormalizeCode(code)
	for i := range all {
		if all[i].Code == code {
			all = append(all[:i], all[i+1:]...)
			return s.persist(all)
		}
	}
	return ErrNotFound
}
