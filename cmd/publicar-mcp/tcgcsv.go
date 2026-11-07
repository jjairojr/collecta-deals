package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	tcgcsvTTL     = 6 * time.Hour
	tcgcsvMax     = 30 << 20
	releaseLayout = "2006-01-02T15:04:05"
)

type listaGame struct {
	name     string
	category int
	liga     string
}

var (
	listaGames = map[string]listaGame{
		"pokemon":   {"Pokémon", 3, "tracking-pkm"},
		"onepiece":  {"One Piece", 68, "tracking"},
		"riftbound": {"Riftbound", 89, "tracking-rft"},
		"magic":     {"Magic", 1, ""},
		"yugioh":    {"Yu-Gi-Oh!", 2, ""},
		"lorcana":   {"Lorcana", 71, ""},
	}
	notReleaseSet = regexp.MustCompile(`(?i)promo|release event|prerelease|pre-release|playtest|oversize|art series|world championship|trainer kit`)
	boxPriority   = []*regexp.Regexp{
		regexp.MustCompile(`(?i)booster (box|display)$`),
		regexp.MustCompile(`(?i)(booster box|display)$`),
		regexp.MustCompile(`(?i)\bbox$`),
		regexp.MustCompile(`(?i)booster pack$`),
	}
	smallImage = regexp.MustCompile(`_\d+w\.jpg$`)
)

type tcgGroup struct {
	ID           int    `json:"groupId"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Supplemental bool   `json:"isSupplemental"`
	PublishedOn  string `json:"publishedOn"`
}

type tcgProduct struct {
	ID       int    `json:"productId"`
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl"`
	Extended []struct {
		Name string `json:"name"`
	} `json:"extendedData"`
}

type tcgRelease struct {
	Group tcgGroup
	Date  time.Time
}

type tcgCache struct {
	at   time.Time
	body []byte
}

func (a *app) tcgcsv(ctx context.Context, path string, out any) error {
	a.tcgMu.Lock()
	cached, ok := a.tcgCache[path]
	a.tcgMu.Unlock()
	if !ok || a.now().Sub(cached.at) >= tcgcsvTTL {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.TCGCSVURL+path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "Collecta/1.0")
		resp, err := a.client.Do(req)
		if err != nil {
			return fmt.Errorf("TCGCSV fora do ar: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("TCGCSV respondeu %d em %s", resp.StatusCode, path)
		}
		var raw json.RawMessage
		if err := json.NewDecoder(io.LimitReader(resp.Body, tcgcsvMax)).Decode(&raw); err != nil {
			return fmt.Errorf("TCGCSV ilegível em %s: %v", path, err)
		}
		cached = tcgCache{at: a.now(), body: raw}
		a.tcgMu.Lock()
		a.tcgCache[path] = cached
		a.tcgMu.Unlock()
	}
	var envelope struct {
		Success bool            `json:"success"`
		Results json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(cached.body, &envelope); err != nil {
		return fmt.Errorf("TCGCSV ilegível em %s: %v", path, err)
	}
	if !envelope.Success {
		return fmt.Errorf("TCGCSV sem resultado em %s", path)
	}
	return json.Unmarshal(envelope.Results, out)
}

func (a *app) releases(ctx context.Context, game listaGame) ([]tcgRelease, error) {
	var groups []tcgGroup
	if err := a.tcgcsv(ctx, fmt.Sprintf("/tcgplayer/%d/groups", game.category), &groups); err != nil {
		return nil, err
	}
	out := []tcgRelease{}
	for _, g := range groups {
		if g.Supplemental || notReleaseSet.MatchString(g.Name) || !strings.HasSuffix(g.PublishedOn, "T00:00:00") {
			continue
		}
		date, err := time.ParseInLocation(releaseLayout, g.PublishedOn, saoPaulo)
		if err != nil {
			continue
		}
		out = append(out, tcgRelease{Group: g, Date: date})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out, nil
}

func (a *app) boxImage(ctx context.Context, game listaGame, group tcgGroup) (string, error) {
	var products []tcgProduct
	if err := a.tcgcsv(ctx, fmt.Sprintf("/tcgplayer/%d/%d/products", game.category, group.ID), &products); err != nil {
		return "", err
	}
	sealed := []tcgProduct{}
	for _, p := range products {
		if !isSingle(p) && p.ImageURL != "" && !strings.Contains(strings.ToLower(p.Name), "case") {
			sealed = append(sealed, p)
		}
	}
	for _, re := range boxPriority {
		for _, p := range sealed {
			if re.MatchString(p.Name) {
				return largeImage(p.ImageURL), nil
			}
		}
	}
	if len(sealed) > 0 {
		return largeImage(sealed[0].ImageURL), nil
	}
	return "", errors.New("set sem foto de produto lacrado no TCGplayer")
}

func isSingle(p tcgProduct) bool {
	for _, e := range p.Extended {
		if e.Name == "Number" {
			return true
		}
	}
	return false
}

func largeImage(raw string) string {
	return smallImage.ReplaceAllString(raw, "_in_1000x1000.jpg")
}
