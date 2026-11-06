package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	historyKeep  = 30 * 24 * time.Hour
	historyDedup = 10 * time.Minute
)

type sentMessage struct {
	Text string    `json:"texto"`
	At   time.Time `json:"em"`
	Kind string    `json:"tipo,omitempty"`
}

type history struct {
	path string
	mu   sync.Mutex
}

func newHistory(dir string) *history {
	return &history{path: filepath.Join(dir, "historico.json")}
}

func (h *history) load() (map[string][]sentMessage, error) {
	data, err := os.ReadFile(h.path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string][]sentMessage{}, nil
	}
	if err != nil {
		return nil, err
	}
	all := map[string][]sentMessage{}
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}
	return all, nil
}

func (h *history) add(key, text, kind string, at time.Time) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	all, err := h.load()
	if err != nil {
		return err
	}
	if prev := all[key]; len(prev) > 0 {
		last := prev[len(prev)-1]
		if last.Text == text && at.Sub(last.At) <= historyDedup {
			return nil
		}
	}
	all[key] = append(all[key], sentMessage{Text: text, At: at, Kind: kind})
	for k, msgs := range all {
		kept := msgs[:0]
		for _, m := range msgs {
			if at.Sub(m.At) <= historyKeep {
				kept = append(kept, m)
			}
		}
		if len(kept) == 0 {
			delete(all, k)
			continue
		}
		all[k] = kept
	}
	data, err := json.Marshal(all)
	if err != nil {
		return err
	}
	tmp := h.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, h.path)
}

func (h *history) keysOfKindOn(kind string, day time.Time) (map[string]bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	all, err := h.load()
	if err != nil {
		return nil, err
	}
	date := day.In(saoPaulo).Format(time.DateOnly)
	out := map[string]bool{}
	for key, msgs := range all {
		for _, m := range msgs {
			if m.Kind == kind && m.At.In(saoPaulo).Format(time.DateOnly) == date {
				out[key] = true
				break
			}
		}
	}
	return out, nil
}

func (h *history) since(key string, from time.Time) ([]sentMessage, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	all, err := h.load()
	if err != nil {
		return nil, err
	}
	out := []sentMessage{}
	for _, m := range all[key] {
		if !m.At.Before(from) {
			out = append(out, m)
		}
	}
	return out, nil
}
