package conversations

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
)

var ErrNotFound = errors.New("conversation not found")

type State string

const (
	StateCompose State = "compose"
	StateDraft   State = "draft"
	StateQueued  State = "queued"
	StateWait    State = "waiting"
	StateActive  State = "active"
	StateHuman   State = "human"
	StateClosed  State = "closed"
)

const (
	ChannelSite = "site"
	ChannelLiga = "liga"
)

const (
	OutcomeDeclined     = "declined"
	OutcomeExpired      = "expired"
	OutcomeComplemented = "complemented"
	OutcomeSkipped      = "skipped"
	OutcomeFailed       = "failed"
)

const (
	RoleCustomer  = "customer"
	RoleAssistant = "assistant"
	RoleOwner     = "owner"
	RoleSystem    = "system"
)

const (
	MsgDraft    = "draft"
	MsgQueued   = "queued"
	MsgSent     = "sent"
	MsgFailed   = "failed"
	MsgReceived = "received"
)

const (
	KindOffer = "offer"
	KindReply = "reply"
)

type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Text      string    `json:"text"`
	At        time.Time `json:"at"`
	Status    string    `json:"status"`
	WAID      string    `json:"waID,omitempty"`
	Error     string    `json:"error,omitempty"`
	Kind      string    `json:"kind,omitempty"`
	ClaimedAt time.Time `json:"claimedAt,omitzero"`
}

type Offer struct {
	SKU      string  `json:"sku"`
	Name     string  `json:"name"`
	Meta     string  `json:"meta"`
	UnitBRL  float64 `json:"unitBRL"`
	ImageURL string  `json:"imageURL,omitempty"`
}

type Item struct {
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}

type LinkLine struct {
	SKU string `json:"sku"`
	Qty int    `json:"qty"`
}

type Conversation struct {
	ID          string `json:"id"`
	Channel     string `json:"channel,omitempty"`
	Phone       string `json:"phone"`
	Name        string `json:"name"`
	OrderID     string `json:"orderID"`
	OrderNumber int    `json:"orderNumber"`
	Pickup      bool   `json:"pickup,omitempty"`
	Shipped     bool   `json:"shipped,omitempty"`
	Items       []Item `json:"items,omitempty"`
	State       State  `json:"state"`
	Outcome     string `json:"outcome"`
	Offer       *Offer `json:"offer"`
	Pitch       string `json:"pitch"`
	Opening     string `json:"opening,omitempty"`

	LinkToken string     `json:"linkToken,omitempty"`
	LinkLines []LinkLine `json:"linkLines,omitempty"`

	SentAt        time.Time `json:"sentAt"`
	WindowEndsAt  time.Time `json:"windowEndsAt"`
	LastInboundAt time.Time `json:"lastInboundAt"`

	ComplementOrder int     `json:"complementOrder"`
	ComplementBRL   float64 `json:"complementBRL"`

	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (c Conversation) IsLiga() bool {
	return c.Channel == ChannelLiga
}

func (c Conversation) Unsent() bool {
	return c.State == StateCompose || c.State == StateDraft || c.State == StateQueued
}

func (c Conversation) PendingDraft() (int, bool) {
	if n := len(c.Messages); n > 0 && c.Messages[n-1].Status == MsgDraft {
		return n - 1, true
	}
	return -1, false
}

func (c *Conversation) DropDrafts() {
	kept := c.Messages[:0]
	for _, m := range c.Messages {
		if m.Status != MsgDraft {
			kept = append(kept, m)
		}
	}
	c.Messages = kept
}

func (c Conversation) QueuedOffer() (int, bool) {
	for i, m := range c.Messages {
		if m.Status == MsgQueued && m.Kind == KindOffer {
			return i, true
		}
	}
	return -1, false
}

func (c *Conversation) Close(outcome string) {
	c.State = StateClosed
	if c.Outcome == "" {
		c.Outcome = outcome
	}
	for i := range c.Messages {
		if c.Messages[i].Status == MsgQueued {
			c.Messages[i].Status = MsgFailed
			c.Messages[i].Error = "não entregue: conversa encerrada"
		}
	}
}

func NewMessage(role, text, status string, at time.Time) Message {
	return Message{ID: NewID(), Role: role, Text: text, At: at, Status: status}
}

type Store struct {
	path string
	mu   sync.RWMutex
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) load() ([]Conversation, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Conversation{}, nil
		}
		return nil, err
	}
	var out []Conversation
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Conversation) error {
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

func (s *Store) List() ([]Conversation, error) {
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

func (s *Store) Get(id string) (Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return Conversation{}, err
	}
	for _, c := range all {
		if c.ID == id {
			return c, nil
		}
	}
	return Conversation{}, ErrNotFound
}

func (s *Store) Create(build func(existing []Conversation) (Conversation, bool)) (Conversation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Conversation{}, false, err
	}
	c, ok := build(all)
	if !ok {
		return Conversation{}, false, nil
	}
	now := time.Now()
	c.ID = NewID()
	c.CreatedAt = now
	c.UpdatedAt = now
	all = append(all, c)
	if err := s.persist(all); err != nil {
		return Conversation{}, false, err
	}
	return c, true, nil
}

func (s *Store) Transact(id string, fn func(*Conversation) error) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Conversation{}, err
	}
	for i := range all {
		if all[i].ID != id {
			continue
		}
		if err := fn(&all[i]); err != nil {
			return Conversation{}, err
		}
		all[i].ID = id
		all[i].UpdatedAt = time.Now()
		if err := s.persist(all); err != nil {
			return Conversation{}, err
		}
		return all[i], nil
	}
	return Conversation{}, ErrNotFound
}

func NewID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return hex.EncodeToString(b[:])
}
