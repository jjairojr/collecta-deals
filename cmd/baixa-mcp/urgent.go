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
	urgentKeep   = 14 * 24 * time.Hour
	reminderHour = 9
)

type urgentItem struct {
	Pedido   string    `json:"pedido"`
	Item     string    `json:"item"`
	Motivo   string    `json:"motivo"`
	ComBaixa bool      `json:"com_baixa"`
	Criado   time.Time `json:"criado"`
}

type urgentFile struct {
	Itens    []urgentItem `json:"itens"`
	Lembrete string       `json:"lembrete,omitempty"`
}

type urgentStore struct {
	path string
	mu   sync.Mutex
}

func newUrgentStore(dir string) *urgentStore {
	return &urgentStore{path: filepath.Join(dir, "urgentes.json")}
}

func (u *urgentStore) load() (urgentFile, error) {
	data, err := os.ReadFile(u.path)
	if errors.Is(err, fs.ErrNotExist) {
		return urgentFile{Itens: []urgentItem{}}, nil
	}
	if err != nil {
		return urgentFile{}, err
	}
	var f urgentFile
	if err := json.Unmarshal(data, &f); err != nil {
		return urgentFile{}, err
	}
	if f.Itens == nil {
		f.Itens = []urgentItem{}
	}
	return f, nil
}

func (u *urgentStore) save(f urgentFile) error {
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	tmp := u.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, u.path)
}

func sameUrgent(a, b urgentItem) bool {
	return a.Pedido == b.Pedido && compact(a.Item) == compact(b.Item)
}

func (u *urgentStore) register(item urgentItem) (bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	f, err := u.load()
	if err != nil {
		return false, err
	}
	for _, it := range f.Itens {
		if sameUrgent(it, item) {
			return true, nil
		}
	}
	f.Itens = append(f.Itens, item)
	return false, u.save(f)
}

func (u *urgentStore) prune(now time.Time, withSale map[string]bool) ([]urgentItem, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	f, err := u.load()
	if err != nil {
		return nil, err
	}
	kept := make([]urgentItem, 0, len(f.Itens))
	for _, it := range f.Itens {
		if now.Sub(it.Criado) > urgentKeep || (it.ComBaixa && !withSale[it.Pedido]) {
			continue
		}
		kept = append(kept, it)
	}
	if len(kept) != len(f.Itens) {
		f.Itens = kept
		if err := u.save(f); err != nil {
			return nil, err
		}
	}
	return kept, nil
}

func (u *urgentStore) takeReminder(now time.Time) (bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	f, err := u.load()
	if err != nil {
		return false, err
	}
	local := now.In(saoPaulo)
	today := local.Format(time.DateOnly)
	if len(f.Itens) == 0 || local.Hour() < reminderHour || f.Lembrete == today {
		return false, nil
	}
	f.Lembrete = today
	return true, u.save(f)
}
