package tracking

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"opdeals/internal/logx"
	"opdeals/internal/model"
)

var (
	unsafePath = regexp.MustCompile(`[^A-Za-z0-9._-]`)
	dateFile   = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}(?:T\d{2})?)\.json$`)
)

const indexTTL = 5 * time.Minute

type Store struct {
	dir string
	mu  sync.RWMutex
	log *logx.Logger

	idxMu  sync.Mutex
	urlIdx map[string]string
	idxAt  time.Time
}

func NewStore(dir string, log *logx.Logger) *Store {
	return &Store{dir: dir, log: log}
}

func (s *Store) setDir(set string) string {
	return filepath.Join(s.dir, unsafePath.ReplaceAllString(set, "_"))
}

func (s *Store) pathFor(set, date string) string {
	return filepath.Join(s.setDir(set), date+".json")
}

func (s *Store) HasDay(set, date string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, err := os.Stat(s.pathFor(set, date))
	return err == nil
}

func (s *Store) SaveDay(d DaySnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := s.setDir(d.Set)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	body, err := json.Marshal(d)
	if err != nil {
		return err
	}
	path := s.pathFor(d.Set, d.Date)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) LoadDay(set, date string) (DaySnapshot, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadFile(s.pathFor(set, date))
}

func (s *Store) loadFile(path string) (DaySnapshot, bool, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DaySnapshot{}, false, nil
		}
		return DaySnapshot{}, false, err
	}
	var d DaySnapshot
	if err := json.Unmarshal(body, &d); err != nil {
		return DaySnapshot{}, false, fmt.Errorf("decode %s: %w", path, err)
	}
	return d, true, nil
}

func (s *Store) ListSets() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var sets []string
	for _, e := range entries {
		if e.IsDir() {
			sets = append(sets, e.Name())
		}
	}
	sort.Strings(sets)
	return sets, nil
}

func (s *Store) LatestDay(set string) (DaySnapshot, bool, error) {
	dates, err := s.ListDates(set)
	if err != nil {
		return DaySnapshot{}, false, err
	}
	if len(dates) == 0 {
		return DaySnapshot{}, false, nil
	}
	return s.LoadDay(set, dates[len(dates)-1])
}

func (s *Store) LatestSnapshot() (DaySnapshot, bool, error) {
	sets, err := s.ListSets()
	if err != nil {
		return DaySnapshot{}, false, err
	}
	bestSet, bestDate := "", ""
	for _, set := range sets {
		dates, err := s.ListDates(set)
		if err != nil || len(dates) == 0 {
			continue
		}
		if d := dates[len(dates)-1]; d > bestDate {
			bestDate = d
			bestSet = set
		}
	}
	if bestSet == "" {
		return DaySnapshot{}, false, nil
	}
	return s.LoadDay(bestSet, bestDate)
}

func (s *Store) ListDates(set string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.setDir(set))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var dates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if m := dateFile.FindStringSubmatch(e.Name()); m != nil {
			dates = append(dates, m[1])
		}
	}
	sort.Strings(dates)
	return dates, nil
}

func (s *Store) PageURLByNumber(number string) (string, bool) {
	number = model.NormalizeNumber(number)
	if number == "" {
		return "", false
	}
	s.idxMu.Lock()
	defer s.idxMu.Unlock()
	if s.urlIdx == nil || time.Since(s.idxAt) > indexTTL {
		s.urlIdx = s.buildURLIndex()
		s.idxAt = time.Now()
	}
	u, ok := s.urlIdx[number]
	return u, ok
}

func (s *Store) buildURLIndex() map[string]string {
	idx := map[string]string{}
	sets, err := s.ListSets()
	if err != nil {
		return idx
	}
	for _, set := range sets {
		day, ok, err := s.LatestDay(set)
		if err != nil || !ok {
			continue
		}
		for _, c := range day.Cards {
			key := model.NormalizeNumber(c.Number)
			if key == "" || c.URL == "" {
				continue
			}
			if _, exists := idx[key]; !exists {
				idx[key] = c.URL
			}
		}
	}
	return idx
}

func (s *Store) LoadRange(set, from, to string) ([]DaySnapshot, error) {
	dates, err := s.ListDates(set)
	if err != nil {
		return nil, err
	}
	var out []DaySnapshot
	for _, date := range dates {
		if from != "" && date < from {
			continue
		}
		if to != "" && date > to {
			continue
		}
		d, ok, err := s.LoadDay(set, date)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, d)
		}
	}
	return out, nil
}
