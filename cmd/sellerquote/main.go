package main

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"opdeals/internal/compare"
	"opdeals/internal/game"
	"opdeals/internal/model"
)

var (
	streamItemRe = regexp.MustCompile(`(?s)<li class="stream-item" data-key="(\d+)">(.*?)</li>`)
	gaRe         = regexp.MustCompile(`data-ga-item-id="([^"]*)"`)
	nameRe       = regexp.MustCompile(`<h3 title="([^"]*)"`)
	edicaoRe     = regexp.MustCompile(`class="card-edicao"[^>]*>([^<]+)<`)
	qtyRe        = regexp.MustCompile(`class="quantidade-num">\s*([0-9.]+)\s*<`)
	precoRe      = regexp.MustCompile(`(?s)class="card-preco moeda"[^>]*>\s*(.*?)\s*</span>`)
	qualRe       = regexp.MustCompile(`(?s)card-qualidade".*?/flags/4x3/([a-z]{2})\.svg.*?<span[^>]*title="([^"]*)"`)
)

type card struct {
	number, set, name, idioma, cond string
	qty                             int
	unitBRL                         float64
}

func firstSub(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

func parseBRL(s string) float64 {
	s = html.UnescapeString(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == ',' {
			b.WriteRune(r)
		}
	}
	t := strings.ReplaceAll(b.String(), ".", "")
	t = strings.ReplaceAll(t, ",", ".")
	v, _ := strconv.ParseFloat(t, 64)
	return v
}

var langMap = map[string]string{"us": "EN", "gb": "EN", "br": "PT", "jp": "JP"}

func parsePages(dir string, cfg game.MyP) []card {
	files, _ := filepath.Glob(filepath.Join(dir, "page-*.html"))
	sort.Strings(files)
	seen := map[string]bool{}
	var out []card
	for _, f := range files {
		doc, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, m := range streamItemRe.FindAllStringSubmatch(string(doc), -1) {
			key, inner := m[1], m[2]
			if seen[key] {
				continue
			}
			seen[key] = true

			ga := firstSub(gaRe, inner)
			parts := strings.Split(ga, "_")
			if len(parts) < 3 || parts[0] != cfg.GAPrefix {
				continue
			}
			number := model.NormalizeNumber(strings.Join(parts[2:], "_"))
			price := parseBRL(firstSub(precoRe, inner))
			if number == "" || price <= 0 {
				continue
			}
			qty, _ := strconv.Atoi(strings.ReplaceAll(strings.TrimSpace(firstSub(qtyRe, inner)), ".", ""))
			country, cond := "", ""
			if q := qualRe.FindStringSubmatch(inner); q != nil {
				country, cond = q[1], html.UnescapeString(strings.TrimSpace(q[2]))
			}
			idioma := langMap[country]
			if idioma == "" {
				idioma = strings.ToUpper(country)
			}
			out = append(out, card{
				number:  number,
				set:     cfg.SetCode(strings.ToUpper(strings.TrimSpace(firstSub(edicaoRe, inner)))),
				name:    html.UnescapeString(strings.TrimSpace(firstSub(nameRe, inner))),
				idioma:  idioma,
				cond:    cond,
				qty:     qty,
				unitBRL: price,
			})
		}
	}
	return out
}

type snapshot struct {
	Listings []model.BrazilListing `json:"listings"`
}

func fmtBR(v float64) string {
	if v == 0 {
		return ""
	}
	return strings.ReplaceAll(strconv.FormatFloat(v, 'f', 2, 64), ".", ",")
}

func main() {
	pagesDir := os.Args[1]
	snapPath := os.Args[2]
	outPath := os.Args[3]

	g := game.Riftbound()
	cards := parsePages(pagesDir, *g.MyP)

	raw, err := os.ReadFile(snapPath)
	if err != nil {
		fmt.Println("read snapshot:", err)
		os.Exit(1)
	}
	var snap snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		fmt.Println("parse snapshot:", err)
		os.Exit(1)
	}

	m := compare.MatcherFor(g)
	var liga []model.BrazilListing
	for _, l := range snap.Listings {
		if strings.HasPrefix(strings.ToLower(l.Source), "liga") {
			liga = append(liga, l)
		}
	}
	ligaPrint := compare.BRPrintIndex(liga, m)
	ligaIdx := compare.BRIndex(liga, m)

	sort.SliceStable(cards, func(i, j int) bool {
		return cards[i].unitBRL*float64(cards[i].qty) > cards[j].unitBRL*float64(cards[j].qty)
	})

	var b strings.Builder
	b.WriteString("\ufeff")
	b.WriteString("Nome;Set;Numero;Idioma;Cond;Qtd;Ikenson_unit_BRL;Ikenson_total_BRL;Liga_unit_BRL;Liga_total_BRL;Ikenson_vs_Liga_%;Liga_URL\n")

	included, overSkipped, ligaMatched := 0, 0, 0
	var sumIk, sumLiga float64
	for _, c := range cards {
		if strings.Contains(strings.ToLower(c.name), "overnumbered") {
			overSkipped++
			continue
		}
		included++
		qty := float64(c.qty)

		var ligaUnit float64
		var ligaURL string
		if l, ok := ligaPrint[m.PrintKey(c.number, c.name, c.set)]; ok {
			ligaUnit, ligaURL = l.LowBRL, l.URL
		} else if l, ok := ligaIdx[m.Key(c.number, c.name, c.set)]; ok {
			ligaUnit, ligaURL = l.LowBRL, l.URL
		}

		ikVsLiga := ""
		if ligaUnit > 0 {
			ligaMatched++
			sumLiga += ligaUnit * qty
			ikVsLiga = fmtBR((c.unitBRL - ligaUnit) / ligaUnit * 100)
		}
		sumIk += c.unitBRL * qty

		fields := []string{
			c.name, c.set, c.number, c.idioma, c.cond, strconv.Itoa(c.qty),
			fmtBR(c.unitBRL), fmtBR(c.unitBRL * qty),
			fmtBR(ligaUnit), fmtBR(ligaUnit * qty),
			ikVsLiga, ligaURL,
		}
		b.WriteString(strings.Join(fields, ";"))
		b.WriteString("\n")
	}

	if err := os.WriteFile(outPath, []byte(b.String()), 0644); err != nil {
		fmt.Println("write csv:", err)
		os.Exit(1)
	}
	fmt.Printf("included: %d | overnumbered skipped: %d | liga matched: %d/%d\n", included, overSkipped, ligaMatched, included)
	fmt.Printf("Ikenson total BRL: %.2f | Liga total (matched) BRL: %.2f\n", sumIk, sumLiga)
	fmt.Println("wrote", outPath)
}
