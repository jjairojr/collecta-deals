package api

import (
	"net/http"
	"regexp"
	"time"

	"opdeals/internal/model"
	"opdeals/internal/tracking"
)

const (
	marketWindow   = 30 * 24 * time.Hour
	marketMaxStale = 14 * 24 * time.Hour
)

var marketLangRe = regexp.MustCompile(`^\d{1,2}$`)

func (s *Server) handleStorefrontMarket(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	w.Header().Set("Cache-Control", "public, max-age=3600")
	gs, ok := s.games[q.Get("game")]
	set := q.Get("set")
	number := model.NormalizeNumber(q.Get("number"))
	lang := q.Get("lang")
	if !ok || gs.Track == nil || set == "" || number == "" || !marketLangRe.MatchString(lang) {
		writeJSON(w, struct{}{})
		return
	}
	now := time.Now()
	days, err := gs.Track.LoadRange(set, now.Add(-marketWindow).Format("2006-01-02"), "")
	if err != nil {
		writeJSON(w, struct{}{})
		return
	}
	price, found := tracking.CardMarket(days, number, lang, s.OwnLigaStoreID)
	if !found {
		writeJSON(w, struct{}{})
		return
	}
	if at, ok := snapshotTime(price.Date); !ok || now.Sub(at) > marketMaxStale {
		writeJSON(w, struct{}{})
		return
	}
	writeJSON(w, price)
}
