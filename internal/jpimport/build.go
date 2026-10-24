package jpimport

import (
	"html"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"opdeals/internal/tracking"
)

const (
	LangJP = "6"
	LangPT = "8"
	LangEN = "2"
)

const (
	WestOK         = "ok"
	WestMissing    = "missing"
	WestUnreleased = "unreleased"
	WestUntracked  = "untracked"
)

var (
	nearMint    = map[string]bool{"0": true, "1": true, "2": true}
	totalRe     = regexp.MustCompile(`\(#\d+[A-Za-z]*/(\d+)\)`)
	leadingNum  = regexp.MustCompile(`^\d+`)
	nameSuffixR = regexp.MustCompile(`\s*\(#.*$`)
)

type SetData struct {
	Day  tracking.DaySnapshot
	Sold map[string]map[string]int
}

type Input struct {
	Capture    Capture
	JPYBRL     float64
	Sets       map[string]SetData
	OwnStoreID int
}

type Lang struct {
	FloorBRL float64 `json:"floorBRL"`
	Copies   int     `json:"copies"`
	Sold     int     `json:"sold"`
}

type Row struct {
	Set         string  `json:"set"`
	SetName     string  `json:"setName"`
	Era         string  `json:"era"`
	Number      int     `json:"number"`
	Rarity      string  `json:"rarity"`
	Name        string  `json:"name"`
	NameJP      string  `json:"nameJP"`
	Yen         int     `json:"yen"`
	CostBRL     float64 `json:"costBRL"`
	Stock       int     `json:"stock"`
	CardRushURL string  `json:"cardRushURL"`
	LigaURL     string  `json:"ligaURL"`
	JP          Lang    `json:"jp"`
	West        string  `json:"west"`
	WestNumber  int     `json:"westNumber"`
	WestURL     string  `json:"westURL"`
	WestStatus  string  `json:"westStatus"`
	PT          Lang    `json:"pt"`
	EN          Lang    `json:"en"`
	RealBRL     float64 `json:"realBRL"`
	RealX       float64 `json:"realX"`
}

type secretCard struct {
	number int
	name   string
	card   tracking.CardDay
}

type westIndex struct {
	secret   []secretCard
	sirStart int
}

func Build(in Input) []Row {
	jpCards := map[string]map[int]tracking.CardDay{}
	westIdx := map[string]westIndex{}
	for code, data := range in.Sets {
		byNum := map[int]tracking.CardDay{}
		for _, c := range data.Day.Cards {
			if n, ok := cardNumber(c.Number); ok {
				byNum[n] = c
			}
		}
		jpCards[code] = byNum
		westIdx[code] = indexWest(code, data.Day)
	}

	type itemRef struct {
		item Item
		set  Set
		name string
	}
	refs := make([]itemRef, 0, len(in.Capture.Items))
	twins := map[string][]int{}
	for _, it := range in.Capture.Items {
		set, ok := SetFor(it.Set)
		if !ok {
			continue
		}
		name := ""
		if c, ok := jpCards[set.JP][it.Number]; ok {
			name = shortName(c.Name)
		}
		refs = append(refs, itemRef{item: it, set: set, name: name})
		if name != "" {
			k := twinKey(set.JP, name, it.Rarity)
			twins[k] = append(twins[k], it.Number)
		}
	}
	for k := range twins {
		sort.Ints(twins[k])
	}

	rows := make([]Row, 0, len(refs))
	for _, ref := range refs {
		it, set := ref.item, ref.set
		row := Row{
			Set:         set.JP,
			SetName:     set.Name,
			Era:         set.Era,
			Number:      it.Number,
			Rarity:      it.Rarity,
			Name:        ref.name,
			NameJP:      it.NameJP,
			Yen:         it.Yen,
			CostBRL:     round2(float64(it.Yen) * in.JPYBRL),
			Stock:       it.Stock,
			CardRushURL: it.URL,
		}
		if c, ok := jpCards[set.JP][it.Number]; ok {
			row.LigaURL = c.URL
			row.JP = langStats(c, in.Sets[set.JP].Sold, LangJP, in.OwnStoreID)
		}
		row.WestStatus = WestUnreleased
		if len(set.West) > 0 {
			row.WestStatus = WestUntracked
		}
		for _, w := range set.West {
			idx, ok := westIdx[w]
			if !ok {
				continue
			}
			row.WestStatus = WestMissing
			if ref.name == "" {
				continue
			}
			nth := indexOf(twins[twinKey(set.JP, ref.name, it.Rarity)], it.Number)
			if c, ok := idx.match(ref.name, it.Rarity, nth); ok {
				row.West, row.WestNumber, row.WestURL, row.WestStatus = w, c.number, c.card.URL, WestOK
				row.PT = langStats(c.card, in.Sets[w].Sold, LangPT, in.OwnStoreID)
				row.EN = langStats(c.card, in.Sets[w].Sold, LangEN, in.OwnStoreID)
				break
			}
		}
		row.RealBRL = realPrice(row.JP.FloorBRL, row.PT.FloorBRL)
		if row.RealBRL > 0 && row.CostBRL > 0 {
			row.RealX = round2(row.RealBRL / row.CostBRL)
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].RealX > rows[j].RealX })
	return rows
}

func indexWest(code string, day tracking.DaySnapshot) westIndex {
	official := officialCount(day)
	var secret []secretCard
	for _, c := range day.Cards {
		n, ok := cardNumber(c.Number)
		if !ok || official == 0 || n <= official {
			continue
		}
		secret = append(secret, secretCard{number: n, name: shortName(c.Name), card: c})
	}
	sort.Slice(secret, func(i, j int) bool { return secret[i].number < secret[j].number })
	start := sirStart[code]
	if start == 0 {
		seen := map[string]bool{}
		for _, c := range secret {
			if seen[c.name] {
				start = c.number
				break
			}
			seen[c.name] = true
		}
	}
	return westIndex{secret: secret, sirStart: start}
}

func (w westIndex) match(name, rarity string, nth int) (secretCard, bool) {
	var cands []secretCard
	for _, c := range w.secret {
		if c.name != name {
			continue
		}
		inSIR := w.sirStart > 0 && c.number >= w.sirStart
		if (rarity == "SAR") == inSIR {
			cands = append(cands, c)
		}
	}
	if len(cands) == 0 {
		return secretCard{}, false
	}
	if nth >= len(cands) {
		nth = len(cands) - 1
	}
	return cands[nth], true
}

func officialCount(day tracking.DaySnapshot) int {
	counts := map[int]int{}
	best, bestN := 0, 0
	for _, c := range day.Cards {
		m := totalRe.FindStringSubmatch(c.Name)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		counts[n]++
		if counts[n] > bestN {
			best, bestN = n, counts[n]
		}
	}
	return best
}

func langStats(c tracking.CardDay, sold map[string]map[string]int, lang string, ownStoreID int) Lang {
	out := Lang{Sold: sold[c.Number][lang]}
	for _, s := range c.Stores {
		if s.Language != lang || !nearMint[s.Condition] || s.StoreID == ownStoreID {
			continue
		}
		if !s.PriceKnown || s.PriceBRL <= 0 || s.Quantity <= 0 {
			continue
		}
		out.Copies += s.Quantity
		if out.FloorBRL == 0 || s.PriceBRL < out.FloorBRL {
			out.FloorBRL = s.PriceBRL
		}
	}
	return out
}

func SoldByLang(days []tracking.DaySnapshot) map[string]map[string]int {
	type key struct {
		store     int
		condition string
		language  string
	}
	last := map[string]map[key]int{}
	out := map[string]map[string]int{}
	for _, d := range days {
		for _, c := range d.Cards {
			if len(c.Stores) == 0 {
				continue
			}
			cur := map[key]int{}
			for _, s := range c.Stores {
				cur[key{s.StoreID, s.Condition, s.Language}] += s.Quantity
			}
			if prev, ok := last[c.Number]; ok {
				for k, q := range prev {
					if drop := q - cur[k]; drop > 0 {
						if out[c.Number] == nil {
							out[c.Number] = map[string]int{}
						}
						out[c.Number][k.language] += drop
					}
				}
			}
			last[c.Number] = cur
		}
	}
	return out
}

func IsSecret(name, number string) bool {
	m := totalRe.FindStringSubmatch(name)
	if m == nil {
		return false
	}
	total, err := strconv.Atoi(m[1])
	if err != nil {
		return false
	}
	n, ok := cardNumber(number)
	return ok && n > total
}

func cardNumber(s string) (int, bool) {
	n, err := strconv.Atoi(leadingNum.FindString(strings.TrimSpace(s)))
	return n, err == nil
}

func realPrice(jp, pt float64) float64 {
	switch {
	case jp > 0 && pt > 0:
		return math.Min(jp, pt)
	case jp > 0:
		return jp
	default:
		return pt
	}
}

func shortName(name string) string {
	return strings.TrimSpace(html.UnescapeString(nameSuffixR.ReplaceAllString(name, "")))
}

func twinKey(set, name, rarity string) string {
	return set + "|" + name + "|" + rarity
}

func indexOf(nums []int, n int) int {
	for i, v := range nums {
		if v == n {
			return i
		}
	}
	return 0
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
