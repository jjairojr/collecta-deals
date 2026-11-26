package packs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound  = errors.New("vínculo não encontrado")
	ErrDuplicate = errors.New("produto já está vinculado a outra embalagem")
	ErrInvalid   = errors.New("vínculo inválido")
)

const (
	MinUnits = 2
	MaxUnits = 1000
)

type Link struct {
	ID        string    `json:"id"`
	Game      string    `json:"game"`
	ParentSKU string    `json:"parentSKU"`
	ChildSKU  string    `json:"childSKU"`
	Units     int       `json:"units"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func Validate(l Link, existing []Link) error {
	switch {
	case strings.TrimSpace(l.Game) == "":
		return fmt.Errorf("%w: jogo obrigatório", ErrInvalid)
	case strings.TrimSpace(l.ParentSKU) == "" || strings.TrimSpace(l.ChildSKU) == "":
		return fmt.Errorf("%w: embalagem e conteúdo obrigatórios", ErrInvalid)
	case l.ParentSKU == l.ChildSKU:
		return fmt.Errorf("%w: embalagem e conteúdo são o mesmo produto", ErrInvalid)
	case l.Units < MinUnits || l.Units > MaxUnits:
		return fmt.Errorf("%w: quantidade precisa estar entre %d e %d", ErrInvalid, MinUnits, MaxUnits)
	}
	parentOf := make(map[string]string, len(existing))
	for _, e := range existing {
		if e.ID == l.ID || e.Game != l.Game {
			continue
		}
		if e.ChildSKU == l.ChildSKU || e.ParentSKU == l.ParentSKU {
			return ErrDuplicate
		}
		parentOf[e.ChildSKU] = e.ParentSKU
	}
	for sku, ok := l.ParentSKU, true; ok; sku, ok = parentOf[sku] {
		if sku == l.ChildSKU {
			return fmt.Errorf("%w: vínculo em círculo", ErrInvalid)
		}
	}
	return nil
}

func ForGame(links []Link, game string) []Link {
	out := make([]Link, 0, len(links))
	for _, l := range links {
		if l.Game == game {
			out = append(out, l)
		}
	}
	return out
}

func ByChild(links []Link, game, childSKU string) (Link, bool) {
	for _, l := range links {
		if l.Game == game && l.ChildSKU == childSKU {
			return l, true
		}
	}
	return Link{}, false
}

func ByParent(links []Link, game, parentSKU string) (Link, bool) {
	for _, l := range links {
		if l.Game == game && l.ParentSKU == parentSKU {
			return l, true
		}
	}
	return Link{}, false
}

type Store struct {
	path string
	mu   sync.RWMutex
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) load() ([]Link, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Link{}, nil
		}
		return nil, err
	}
	var out []Link
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Link) error {
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

func (s *Store) List() ([]Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].CreatedAt.Before(all[j].CreatedAt) })
	return all, nil
}

func (s *Store) Add(l Link) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Link{}, err
	}
	l.ID = newID()
	if err := Validate(l, all); err != nil {
		return Link{}, err
	}
	now := time.Now()
	l.CreatedAt = now
	l.UpdatedAt = now
	all = append(all, l)
	if err := s.persist(all); err != nil {
		return Link{}, err
	}
	return l, nil
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
