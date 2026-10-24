package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"opdeals/internal/fx"
	"opdeals/internal/httpx"
	"opdeals/internal/jpimport"
	"opdeals/internal/tracking"
)

const (
	jpImportGame     = "pokemon"
	jpImportSoldDays = 14
	jpImportFXTTL    = 6 * time.Hour
)

type JPImport struct {
	Store        *tracking.Store
	CardRushPath string

	mu       sync.Mutex
	key      string
	resp     jpImportResponse
	fxRate   float64
	fxAt     time.Time
	fxClient *httpx.Client
}

type jpImportResponse struct {
	CapturedAt   time.Time      `json:"capturedAt"`
	FXJPYBRL     float64        `json:"fxJPYBRL"`
	TrackingDate string         `json:"trackingDate"`
	Sets         []jpimport.Set `json:"sets"`
	Count        int            `json:"count"`
	Rows         []jpimport.Row `json:"rows"`
}

func (s *Server) handleJPImport(w http.ResponseWriter, r *http.Request) {
	gs, ok := s.games[jpImportGame]
	if !ok || s.JPImport == nil {
		http.Error(w, "jp import not configured", http.StatusNotFound)
		return
	}
	resp, err := s.JPImport.build(r.Context(), gs, s.OwnLigaStoreID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, resp)
}

func (j *JPImport) build(ctx context.Context, gs *GameStack, ownStoreID int) (jpImportResponse, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	capture, modTime, err := jpimport.LoadCapture(j.CardRushPath)
	if err != nil {
		return jpImportResponse{}, err
	}
	rate, err := j.rate(ctx)
	if err != nil {
		return jpImportResponse{}, err
	}

	stores := []*tracking.Store{gs.Track, j.Store}
	type source struct {
		day   tracking.DaySnapshot
		store *tracking.Store
	}
	latest := map[string]source{}
	trackingDate := ""
	var stamp strings.Builder
	for _, code := range jpimport.LigaSets() {
		day, st, err := latestDay(stores, code)
		if err != nil {
			return jpImportResponse{}, err
		}
		if st == nil {
			continue
		}
		latest[code] = source{day: day, store: st}
		stamp.WriteString(code + "@" + day.Date + ";")
		if day.Date > trackingDate {
			trackingDate = day.Date
		}
	}

	key := fmt.Sprintf("%d|%s|%.6f", modTime.UnixNano(), stamp.String(), rate)
	if key == j.key {
		return j.resp, nil
	}

	sets := make(map[string]jpimport.SetData, len(latest))
	for code, src := range latest {
		days, err := rangeFor(src.store, src.day)
		if err != nil {
			return jpImportResponse{}, err
		}
		sets[code] = jpimport.SetData{Day: src.day, Sold: jpimport.SoldByLang(days)}
	}
	rows := jpimport.Build(jpimport.Input{Capture: capture, JPYBRL: rate, Sets: sets, OwnStoreID: ownStoreID})
	j.key = key
	j.resp = jpImportResponse{
		CapturedAt:   capture.CapturedAt,
		FXJPYBRL:     rate,
		TrackingDate: trackingDate,
		Sets:         jpimport.Sets,
		Count:        len(rows),
		Rows:         rows,
	}
	return j.resp, nil
}

func (j *JPImport) rate(ctx context.Context) (float64, error) {
	if j.fxRate > 0 && time.Since(j.fxAt) < jpImportFXTTL {
		return j.fxRate, nil
	}
	if j.fxClient == nil {
		c, err := httpx.New(15 * time.Second)
		if err != nil {
			return 0, err
		}
		j.fxClient = c
	}
	rate, err := fx.RateJPY(ctx, j.fxClient)
	if err != nil {
		if j.fxRate > 0 {
			return j.fxRate, nil
		}
		return 0, err
	}
	j.fxRate, j.fxAt = rate, time.Now()
	return rate, nil
}

func latestDay(stores []*tracking.Store, set string) (tracking.DaySnapshot, *tracking.Store, error) {
	var best tracking.DaySnapshot
	var from *tracking.Store
	for _, st := range stores {
		if st == nil {
			continue
		}
		day, ok, err := st.LatestDayWithDepth(set, importDepthLookback)
		if err != nil {
			return tracking.DaySnapshot{}, nil, err
		}
		if ok && day.Date > best.Date {
			best, from = day, st
		}
	}
	return best, from, nil
}

func rangeFor(st *tracking.Store, latest tracking.DaySnapshot) ([]tracking.DaySnapshot, error) {
	end, err := time.Parse("2006-01-02", latest.Date[:10])
	if err != nil {
		return nil, err
	}
	from := end.AddDate(0, 0, -jpImportSoldDays).Format("2006-01-02")
	return st.LoadRange(latest.Set, from, latest.Date)
}
