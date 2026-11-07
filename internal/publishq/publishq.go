package publishq

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var (
	ErrNotFound     = errors.New("publicação não encontrada")
	ErrBadToken     = errors.New("código de aprovação inválido")
	ErrNotPending   = errors.New("publicação já foi decidida")
	ErrNotApproved  = errors.New("publicação não está aprovada")
	ErrInvalidInput = errors.New("pedido de publicação inválido")
)

const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusPublished = "published"
	StatusFailed    = "failed"
)

const MaxAttempts = 3

type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type Item struct {
	ID          string    `json:"id"`
	Channel     string    `json:"channel"`
	Format      string    `json:"format"`
	Files       []File    `json:"files"`
	Caption     string    `json:"caption"`
	PostAt      time.Time `json:"postAt"`
	Note        string    `json:"note,omitempty"`
	Status      string    `json:"status"`
	Token       string    `json:"token,omitempty"`
	Attempts    int       `json:"attempts,omitempty"`
	Error       string    `json:"error,omitempty"`
	Link        string    `json:"link,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	DecidedAt   time.Time `json:"decidedAt,omitzero"`
	PublishedAt time.Time `json:"publishedAt,omitzero"`
}

func (it Item) Public() Item {
	it.Token = ""
	return it
}

type Store struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

func NewStore(path string) *Store {
	return &Store{path: path, now: time.Now}
}

func NewStoreWithClock(path string, now func() time.Time) *Store {
	return &Store{path: path, now: now}
}

func (s *Store) load() ([]Item, error) {
	body, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []Item{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Item
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Item) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

var formats = map[string]map[string]bool{
	"instagram": {"reels": true, "post": true, "carrossel": true, "teste": true},
	"tiktok":    {"video": true},
}

func ValidFormat(channel, format string) bool {
	return formats[channel][format]
}

func ViaAPI(channel, format string) bool {
	return channel == "instagram" && format == "teste"
}

func SelfApproved(channel, format string) bool {
	return channel == "tiktok" || ViaAPI(channel, format)
}

func (s *Store) Add(it Item) (Item, error) {
	if !ValidFormat(it.Channel, it.Format) || len(it.Files) == 0 || it.PostAt.IsZero() {
		return Item{}, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Item{}, err
	}
	it.ID = randomHex(8)
	it.Token = randomHex(16)
	it.Status = StatusPending
	it.Attempts = 0
	it.Error = ""
	it.Link = ""
	it.CreatedAt = s.now()
	it.DecidedAt = time.Time{}
	it.PublishedAt = time.Time{}
	all = append(all, it)
	if err := s.persist(all); err != nil {
		return Item{}, err
	}
	return it, nil
}

func (s *Store) List(status string) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(all))
	for _, it := range all {
		if status == "" || it.Status == status {
			out = append(out, it.Public())
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].PostAt.Before(out[j].PostAt) })
	return out, nil
}

func (s *Store) update(id string, change func(*Item) error) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Item{}, err
	}
	for i := range all {
		if all[i].ID != id {
			continue
		}
		if err := change(&all[i]); err != nil {
			return Item{}, err
		}
		if err := s.persist(all); err != nil {
			return Item{}, err
		}
		return all[i].Public(), nil
	}
	return Item{}, ErrNotFound
}

func (s *Store) Decide(id, token string, approve bool) (Item, error) {
	return s.update(id, func(it *Item) error {
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(it.Token)) != 1 {
			return ErrBadToken
		}
		if it.Status != StatusPending {
			return ErrNotPending
		}
		it.Status = StatusRejected
		if approve {
			it.Status = StatusApproved
		}
		it.DecidedAt = s.now()
		return nil
	})
}

func (s *Store) Published(id, link string) (Item, error) {
	return s.update(id, func(it *Item) error {
		if it.Status != StatusApproved {
			return ErrNotApproved
		}
		it.Status = StatusPublished
		it.Link = link
		it.Error = ""
		it.PublishedAt = s.now()
		return nil
	})
}

func (s *Store) Failed(id, msg string, permanent bool) (Item, error) {
	return s.update(id, func(it *Item) error {
		if it.Status != StatusApproved {
			return ErrNotApproved
		}
		it.Attempts++
		it.Error = msg
		if permanent || it.Attempts >= MaxAttempts {
			it.Status = StatusFailed
		}
		return nil
	})
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
