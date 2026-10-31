package syncqueue

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var ErrNotFound = errors.New("sync task not found")

const (
	TargetLiga = "liga"
	TargetMyP  = "myp"
)

const (
	ActionRemove  = "remove"
	ActionRestore = "restore"
)

const (
	StatusPending = "pending"
	StatusDone    = "done"
	StatusError   = "error"
)

type Task struct {
	ID        string    `json:"id"`
	Game      string    `json:"game"`
	TradeID   string    `json:"tradeID"`
	Name      string    `json:"name"`
	Number    string    `json:"number,omitempty"`
	Variant   string    `json:"variant,omitempty"`
	Condition string    `json:"condition,omitempty"`
	Target    string    `json:"target"`
	Action    string    `json:"action"`
	Qty       int       `json:"qty"`
	Reason    string    `json:"reason,omitempty"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	DoneAt    time.Time `json:"doneAt,omitempty"`
}

type Store struct {
	path string
	mu   sync.RWMutex
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) load() ([]Task, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil
		}
		return nil, err
	}
	var out []Task
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Task) error {
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

func (s *Store) List(status string) ([]Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]Task, 0, len(all))
	for _, t := range all {
		if status == "" || t.Status == status {
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) Add(t Task) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Task{}, err
	}
	t.ID = newID()
	t.Status = StatusPending
	t.Error = ""
	t.CreatedAt = time.Now()
	t.DoneAt = time.Time{}
	all = append(all, t)
	if err := s.persist(all); err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s *Store) SetStatus(id, status, errMsg string) (Task, error) {
	switch status {
	case StatusPending, StatusDone, StatusError:
	default:
		return Task{}, fmt.Errorf("unknown status %q", status)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Task{}, err
	}
	for i := range all {
		if all[i].ID != id {
			continue
		}
		all[i].Status = status
		all[i].Error = ""
		all[i].DoneAt = time.Time{}
		switch status {
		case StatusError:
			all[i].Error = errMsg
		case StatusDone:
			all[i].DoneAt = time.Now()
		}
		if err := s.persist(all); err != nil {
			return Task{}, err
		}
		return all[i], nil
	}
	return Task{}, ErrNotFound
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return hex.EncodeToString(b[:])
}
