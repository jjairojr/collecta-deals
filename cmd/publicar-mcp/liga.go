package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"opdeals/internal/logx"
	"opdeals/internal/tracking"
)

const (
	ligaStale      = 60 * time.Hour
	ligaWindow     = 7 * 24 * time.Hour
	ligaWindowMax  = 21 * 24 * time.Hour
	ligaSetLag     = 24 * time.Hour
	moverFloorBRL  = 10
	moverMinStores = 3
	hotFloorBRL    = 5
	sealedSet      = "SEALED"
)

var cardSuffix = regexp.MustCompile(`\s*\([^()]*\d[^()]*\)\s*$`)

type ligaCard struct {
	Set     string
	Number  string
	Name    string
	Image   string
	Before  float64
	Now     float64
	Change  float64
	Units   int
	Average float64
}

type ligaReport struct {
	From  time.Time
	To    time.Time
	Cards []ligaCard
}

type ligaSet struct {
	name   string
	base   string
	latest string
	at     time.Time
}

func parseSnapshotDate(raw string) (time.Time, error) {
	if len(raw) == len("2006-01-02") {
		return time.ParseInLocation("2006-01-02", raw, saoPaulo)
	}
	return time.ParseInLocation("2006-01-02T15", raw, saoPaulo)
}

func cardName(raw string) string {
	name := html.UnescapeString(raw)
	for cardSuffix.MatchString(name) {
		name = cardSuffix.ReplaceAllString(name, "")
	}
	return cleanLine(name)
}

func (a *app) ligaSets(store *tracking.Store, game listaGame) ([]ligaSet, error) {
	names, err := store.ListSets()
	if err != nil {
		return nil, fmt.Errorf("snapshots da Liga ilegíveis: %v", err)
	}
	found := []ligaSet{}
	var newest time.Time
	for _, name := range names {
		if name == sealedSet {
			continue
		}
		dates, err := store.ListDates(name)
		if err != nil || len(dates) < 2 {
			continue
		}
		latest := dates[len(dates)-1]
		at, err := parseSnapshotDate(latest)
		if err != nil {
			continue
		}
		base := ""
		for _, d := range dates {
			t, err := parseSnapshotDate(d)
			if err == nil && !t.After(at.Add(-ligaWindow)) && !t.Before(at.Add(-ligaWindowMax)) {
				base = d
			}
		}
		if base == "" {
			continue
		}
		found = append(found, ligaSet{name: name, base: base, latest: latest, at: at})
		if at.After(newest) {
			newest = at
		}
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("sem snapshots da Liga de %s com 7 dias de histórico", game.name)
	}
	if a.now().Sub(newest) > ligaStale {
		return nil, fmt.Errorf("os dados da Liga de %s estão parados desde %s: sem dado novo não dá para dizer o que subiu ou vendeu. Anote em Pendências do Jairo (o container opdeals no Docker coleta a cada 48h e não completou o último ciclo)", game.name, newest.Format("02/01 15h"))
	}
	fresh := found[:0]
	for _, s := range found {
		if newest.Sub(s.at) <= ligaSetLag {
			fresh = append(fresh, s)
		}
	}
	return fresh, nil
}

func ligaImages(dir string) map[string]string {
	images := map[string]string{}
	data, err := os.ReadFile(filepath.Join(dir, "images.json"))
	if err != nil {
		return images
	}
	_ = json.Unmarshal(data, &images)
	return images
}

func storesWithStock(card tracking.CardDay) int {
	seen := map[int]bool{}
	for _, s := range card.Stores {
		if s.Quantity > 0 {
			seen[s.StoreID] = true
		}
	}
	return len(seen)
}

func (a *app) ligaReport(game listaGame, tipo string) (ligaReport, error) {
	if game.liga == "" {
		return ligaReport{}, fmt.Errorf("a Liga não tem dados de %s: subiram, em_alta, mais_caras, caiu, chuta e duelo só valem para pokemon, onepiece e riftbound", game.name)
	}
	dir := filepath.Join(a.cfg.LigaDir, game.liga)
	store := tracking.NewStore(dir, logx.New(io.Discard))
	sets, err := a.ligaSets(store, game)
	if err != nil {
		return ligaReport{}, err
	}
	images := ligaImages(dir)
	report := ligaReport{}
	seen := map[string]bool{}
	for _, s := range sets {
		cards, err := a.ligaSetCards(store, s, tipo)
		if err != nil {
			return ligaReport{}, err
		}
		from, _ := parseSnapshotDate(s.base)
		if report.From.IsZero() || from.Before(report.From) {
			report.From = from
		}
		if s.at.After(report.To) {
			report.To = s.at
		}
		for _, c := range cards {
			if seen[c.Number] {
				continue
			}
			c.Image = images[s.name+"/"+c.Number]
			if c.Image == "" {
				continue
			}
			seen[c.Number] = true
			report.Cards = append(report.Cards, c)
		}
	}
	sort.SliceStable(report.Cards, func(i, j int) bool {
		switch tipo {
		case "subiram":
			return report.Cards[i].Change > report.Cards[j].Change
		case "caiu":
			return report.Cards[i].Change < report.Cards[j].Change
		case "mais_caras", "chuta", "duelo":
			return report.Cards[i].Now > report.Cards[j].Now
		}
		if report.Cards[i].Units != report.Cards[j].Units {
			return report.Cards[i].Units > report.Cards[j].Units
		}
		return report.Cards[i].Average > report.Cards[j].Average
	})
	if len(report.Cards) == 0 {
		return ligaReport{}, errors.New("nenhuma carta passou nos filtros da Liga nesta janela: escolha outro jogo ou formato")
	}
	return report, nil
}

func (a *app) ligaSetCards(store *tracking.Store, s ligaSet, tipo string) ([]ligaCard, error) {
	if tipo == "em_alta" {
		days, err := store.LoadRange(s.name, s.base, s.latest)
		if err != nil {
			return nil, fmt.Errorf("snapshots da Liga ilegíveis em %s: %v", s.name, err)
		}
		out := []ligaCard{}
		for _, sale := range tracking.TopSoldCards(days) {
			avg := sale.RevenueBRL / float64(sale.Units)
			if sale.Units < 2 || avg < hotFloorBRL {
				continue
			}
			out = append(out, ligaCard{Set: s.name, Number: sale.Number, Name: cardName(sale.Name), Units: sale.Units, Average: math.Round(avg*100) / 100})
		}
		return out, nil
	}
	prev, okPrev, err := store.LoadDay(s.name, s.base)
	if err != nil {
		return nil, fmt.Errorf("snapshot da Liga ilegível em %s: %v", s.name, err)
	}
	cur, okCur, err := store.LoadDay(s.name, s.latest)
	if err != nil {
		return nil, fmt.Errorf("snapshot da Liga ilegível em %s: %v", s.name, err)
	}
	if !okPrev || !okCur {
		return nil, nil
	}
	before := map[string]float64{}
	for _, c := range prev.Cards {
		before[c.Number] = c.LowBRL
	}
	out := []ligaCard{}
	for _, c := range cur.Cards {
		if c.LowBRL < moverFloorBRL || storesWithStock(c) < moverMinStores {
			continue
		}
		card := ligaCard{Set: s.name, Number: c.Number, Name: cardName(c.Name), Now: c.LowBRL}
		old := before[c.Number]
		switch tipo {
		case "subiram":
			if old < moverFloorBRL || c.LowBRL <= old {
				continue
			}
		case "caiu":
			if old < moverFloorBRL || c.LowBRL >= old {
				continue
			}
		default:
			out = append(out, card)
			continue
		}
		card.Before, card.Change = old, math.Round((c.LowBRL-old)/old*100)
		out = append(out, card)
	}
	return out, nil
}
