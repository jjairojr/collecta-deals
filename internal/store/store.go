package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"opdeals/internal/logx"
	"opdeals/internal/model"
	"opdeals/internal/pipeline"
)

type Store struct {
	mu         sync.RWMutex
	snap       model.Snapshot
	ready      bool
	refreshing bool

	path      string
	logger    *logx.Logger
	fetchOpts pipeline.Options
}

func New(path string, logger *logx.Logger, fetchOpts pipeline.Options) *Store {
	return &Store{path: path, logger: logger, fetchOpts: fetchOpts}
}

func (s *Store) Load() error {
	if s.path == "" {
		return nil
	}
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read cache: %w", err)
	}
	var snap model.Snapshot
	if err := json.Unmarshal(body, &snap); err != nil {
		return fmt.Errorf("decode cache: %w", err)
	}
	s.mu.Lock()
	s.snap = snap
	s.ready = len(snap.Listings) > 0 && len(snap.Prices) > 0
	s.mu.Unlock()
	s.logger.Printf("cache loaded: %d BR listings, %d US prices from %s", len(snap.Listings), len(snap.Prices), snap.UpdatedAt.Format("2006-01-02 15:04"))
	return nil
}

func (s *Store) Snapshot() model.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

func (s *Store) Status() (ready, refreshing bool, snap model.Snapshot) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready, s.refreshing, s.snap
}

func (s *Store) Refresh(ctx context.Context) error {
	s.mu.Lock()
	if s.refreshing {
		s.mu.Unlock()
		return nil
	}
	s.refreshing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.refreshing = false
		s.mu.Unlock()
	}()

	snap, err := pipeline.Fetch(ctx, s.logger, s.fetchOpts)
	if err != nil {
		s.logger.Printf("refresh failed: %v", err)
		return err
	}

	s.mu.Lock()
	s.snap = snap
	s.ready = true
	s.mu.Unlock()

	if err := s.save(snap); err != nil {
		s.logger.Printf("warning: could not persist cache: %v", err)
	}
	return nil
}

func (s *Store) save(snap model.Snapshot) error {
	if s.path == "" {
		return nil
	}
	if dir := filepath.Dir(s.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	body, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
