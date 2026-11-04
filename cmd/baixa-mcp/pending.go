package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"opdeals/internal/trades"
)

const pendingKeep = 14 * 24 * time.Hour

type pendingItem struct {
	Pedido    string    `json:"pedido"`
	Jogo      string    `json:"jogo"`
	Numero    string    `json:"numero,omitempty"`
	Nome      string    `json:"nome"`
	Qty       int       `json:"qty"`
	TotalBRL  float64   `json:"total_brl"`
	Data      string    `json:"data"`
	Comprador string    `json:"comprador"`
	Motivo    string    `json:"motivo"`
	Criada    time.Time `json:"criada"`
}

type pendingStore struct {
	path string
	mu   sync.Mutex
}

func newPendingStore(dir string) *pendingStore {
	return &pendingStore{path: filepath.Join(dir, "pendencias.json")}
}

func (p *pendingStore) load() ([]pendingItem, error) {
	data, err := os.ReadFile(p.path)
	if errors.Is(err, fs.ErrNotExist) {
		return []pendingItem{}, nil
	}
	if err != nil {
		return nil, err
	}
	var all []pendingItem
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}
	return all, nil
}

func (p *pendingStore) save(all []pendingItem) error {
	data, err := json.Marshal(all)
	if err != nil {
		return err
	}
	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p.path)
}

func samePending(a, b pendingItem) bool {
	return a.Pedido == b.Pedido && a.Jogo == b.Jogo && compact(a.Numero) == compact(b.Numero) && compact(a.Nome) == compact(b.Nome)
}

func (p *pendingStore) add(item pendingItem) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	all, err := p.load()
	if err != nil {
		return err
	}
	for i := range all {
		if samePending(all[i], item) {
			item.Criada = all[i].Criada
			all[i] = item
			return p.save(all)
		}
	}
	return p.save(append(all, item))
}

func resolvedBy(item pendingItem, ledger []trades.Trade) bool {
	re := regexp.MustCompile(`Liga #` + regexp.QuoteMeta(item.Pedido) + `\b`)
	for _, t := range ledger {
		if t.Status == "sold" && re.MatchString(t.Buyer) && compact(t.Name) == compact(item.Nome) && compact(t.Number) == compact(item.Numero) {
			return true
		}
	}
	return false
}

func (p *pendingStore) open(now time.Time, ledgers map[string][]trades.Trade) ([]pendingItem, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	all, err := p.load()
	if err != nil {
		return nil, err
	}
	kept := make([]pendingItem, 0, len(all))
	for _, it := range all {
		if now.Sub(it.Criada) > pendingKeep || resolvedBy(it, ledgers[it.Jogo]) {
			continue
		}
		kept = append(kept, it)
	}
	if len(kept) != len(all) {
		if err := p.save(kept); err != nil {
			return nil, err
		}
	}
	return kept, nil
}

func cleanText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
