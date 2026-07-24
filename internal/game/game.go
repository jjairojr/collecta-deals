// Package game holds the per-game configuration that parameterizes the otherwise
// identical Liga (sbrauble) scraping + tracking pipeline. One Piece and Pokémon
// are served by the same platform (ligaonepiece.com.br / ligapokemon.com.br), so
// the only differences are captured here: hosts, the CDN image path, which
// languages define a card's floor, the main-character name tokens, and the set
// list. Everything downstream reads a Game value instead of hard-coded One Piece.
package game

import "regexp"

// SealedCategory is a Liga sealed-product category (booster box, pack, etc.) and
// its numeric id used in the `searchprod=1` catalog query.
type SealedCategory struct {
	ID   string
	Type string
}

// Game is the immutable configuration for one trading-card game.
type Game struct {
	ID          string // stable identifier: "onepiece" | "pokemon"
	Name        string // display name
	LigaBaseURL string // e.g. "https://www.ligaonepiece.com.br/"
	LigaHosts   []string
	SourceName  string // e.g. "ligaonepiece"

	// ImageURLRe matches the card's CDN art URL on its Liga card page.
	ImageURLRe *regexp.Regexp
	// SealedSetCodeRe extracts a set code from a sealed product name; may be nil.
	SealedSetCodeRe *regexp.Regexp

	SealedCategories []SealedCategory

	// FloorLangs restricts which listing languages count toward a card's floor
	// price and buyout ladder (Liga idioma codes: 2=EN, 6=JP, 8=PT). Empty means
	// all languages qualify.
	FloorLangs []string

	// BuyoutLangs narrows the listing languages the buyout/snipe ladders consider,
	// independently of FloorLangs. Empty means fall back to FloorLangs. Pokémon uses
	// this to exclude Japanese from buyout/snipe while JP still counts toward floor.
	BuyoutLangs []string

	// MainCharTokens are lower-cased substrings that flag a "main character" card
	// for the buyout `chars=main` filter.
	MainCharTokens []string

	DefaultSet       string
	DefaultTrackSets []string

	// UniqueCardNumbers reports whether card numbers are globally unique across
	// sets (One Piece's "OP01-025"). It enables number-keyed, set-insensitive
	// fallbacks (card image cache, page-URL index) that tolerate set-code format
	// drift. Games with per-set bare numbers ("238" in LOR9/LOR10/LOR11) must
	// leave it false or those fallbacks serve another set's card.
	UniqueCardNumbers bool

	// Market configures the US (TCGplayer/TCGCSV) side of the deals pipeline;
	// nil means the game is BR-tracking-only (no deals).
	Market *Market

	// Challenged reports whether this game's Liga host sits behind Cloudflare's
	// Managed Challenge and must be fetched through FlareSolverr (a headless
	// browser) rather than the direct HTTP client. onepiece/riftbound/lorcana are
	// challenged; pokemon is currently open. Flip if Liga's protection changes.
	Challenged bool

	// MyP configures mypcards.com as a second Brazilian source; nil means the
	// game is not carried there and the MyP scraper skips it entirely.
	MyP *MyP
}

// MyP describes a game's section of mypcards.com. The site lays every game out
// identically (/{Slug}/edicoes paginated, then /{Slug}/{edition-slug}), so a
// game only needs its URL slug, the prefix its data-ga-item-id values carry,
// and any set codes that disagree with the Liga/TCGCSV vocabulary.
type MyP struct {
	// Slug is the URL path segment, e.g. "onepiece" or "riftbound".
	Slug string

	// GAPrefix is the leading token of data-ga-item-id ("one", "riftbound").
	// Listing grids carry cross-sell cards from other games, so this filter is
	// what keeps another game's cards out of this game's snapshot.
	GAPrefix string

	// SetAliases rewrites MyP's set code to the canonical Liga/TCGCSV one. Set
	// codes are part of the match key for set-scoped games, so an unmapped
	// disagreement silently drops every deal in that set.
	SetAliases map[string]string

	// StripPrintSuffix removes a trailing "p<digits>" print marker from the
	// number (One Piece writes "op16-080p1"). Games whose numbers carry other
	// trailing letters must leave this false and let the Matcher normalize.
	StripPrintSuffix bool
}

// SetCode maps a MyP set code to its canonical Liga/TCGCSV equivalent.
func (m *MyP) SetCode(code string) string {
	if m == nil {
		return code
	}
	if canon, ok := m.SetAliases[code]; ok {
		return canon
	}
	return code
}

// AllowsHost reports whether host is one of this game's Liga hosts.
func (g Game) AllowsHost(host string) bool {
	for _, h := range g.LigaHosts {
		if h == host {
			return true
		}
	}
	return false
}

// FloorLangAllowed reports whether a listing language counts toward the floor.
func (g Game) FloorLangAllowed(lang string) bool {
	if len(g.FloorLangs) == 0 {
		return true
	}
	for _, l := range g.FloorLangs {
		if l == lang {
			return true
		}
	}
	return false
}

// MultiLanguage reports whether this game's Brazil market prices every language
// as the same product (no FloorLangs restriction). Only there is a listing's
// language worth surfacing on its own — elsewhere the floor is English by
// definition and a JP or PT sale is a footnote, not a distinct market.
func (g Game) MultiLanguage() bool {
	return len(g.FloorLangs) == 0
}

// BuyoutFloorLangs returns the listing languages the buyout/snipe ladders count.
// It defaults to FloorLangs but a game may narrow it further (e.g. Pokémon drops
// Japanese from buyout/snipe while JP still counts toward the tracking floor).
func (g Game) BuyoutFloorLangs() []string {
	if len(g.BuyoutLangs) > 0 {
		return g.BuyoutLangs
	}
	return g.FloorLangs
}

var onePieceImageRe = regexp.MustCompile(`//repositorio\.sbrauble\.com/arquivos/in/onepiece/\d+/[^"'\s]+\.jpg`)
var onePieceSealedRe = regexp.MustCompile(`\b(OP|ST|EB|PRB|DP)-?\d{2}\b`)
var pokemonImageRe = regexp.MustCompile(`//repositorio\.sbrauble\.com/arquivos/in/pokemon_bkp/[^"'\s]+\.jpg`)
var riftboundImageRe = regexp.MustCompile(`//repositorio\.sbrauble\.com/arquivos/in/riftbound/[^"'\s]+\.jpg`)
var lorcanaImageRe = regexp.MustCompile(`//repositorio\.sbrauble\.com/arquivos/in/lorcana/[^"'\s]+\.jpg`)
var gundamImageRe = regexp.MustCompile(`//repositorio\.sbrauble\.com/arquivos/in/gundam/[^"'\s]+\.jpg`)

// OnePiece is the original game configuration; its values reproduce the app's
// pre-multi-game behavior exactly (English-only floor, Straw Hat main chars).
func OnePiece() Game {
	return Game{
		ID:                "onepiece",
		Name:              "One Piece",
		LigaBaseURL:       "https://www.ligaonepiece.com.br/",
		LigaHosts:         []string{"www.ligaonepiece.com.br", "ligaonepiece.com.br"},
		SourceName:        "ligaonepiece",
		ImageURLRe:        onePieceImageRe,
		SealedSetCodeRe:   onePieceSealedRe,
		UniqueCardNumbers: true,
		SealedCategories: []SealedCategory{
			{ID: "10", Type: "Booster Box"},
			{ID: "21", Type: "Booster Pack"},
			{ID: "28", Type: "Collector Box"},
			{ID: "36", Type: "Starter Deck"},
		},
		FloorLangs: []string{"2"},
		MainCharTokens: []string{
			"luffy", "zoro", "nami", "usopp", "sogeking", "sanji",
			"chopper", "robin", "franky", "brook", "jinbe", "jimbei",
			"hancock", "trafalgar",
		},
		DefaultSet: "OP-16",
		DefaultTrackSets: []string{
			"OP-16", "OP-15", "OP-14", "OP-13", "OP-12", "OP-11", "OP-10", "OP-09",
			"OP-08", "OP-07", "OP-06", "OP-05", "OP-04", "OP-03", "OP-02", "OP-01",
			"EB-04", "EB03", "EB02", "EB01", "PRB2", "PRB",
			"ST30", "ST29", "ST28", "ST27", "ST26", "ST25", "ST24", "ST23", "ST22",
			"ST21", "ST20", "ST19", "ST18", "ST17", "ST16", "ST15", "ST14",
			"ST-13", "ST-12", "ST-11", "ST-10", "ST-09", "ST-08", "ST-07", "ST-06",
			"ST-05", "ST-04", "ST-03", "ST-02", "ST-01",
		},
		Market:     onePieceMarket(),
		Challenged: true,
		MyP: &MyP{
			Slug:             "onepiece",
			GAPrefix:         "one",
			StripPrintSuffix: true,
		},
	}
}

// Riftbound is the ligariftbound.com.br configuration. The BR market carries
// Chinese (idioma 10) listings from the China-first release at much lower
// prices, so the floor is English-only like One Piece.
func Riftbound() Game {
	return Game{
		ID:              "riftbound",
		Name:            "Riftbound",
		LigaBaseURL:     "https://www.ligariftbound.com.br/",
		LigaHosts:       []string{"www.ligariftbound.com.br", "ligariftbound.com.br"},
		SourceName:      "ligariftbound",
		ImageURLRe:      riftboundImageRe,
		SealedSetCodeRe: nil,
		SealedCategories: []SealedCategory{
			{ID: "10", Type: "Booster Box"},
			{ID: "21", Type: "Booster Pack"},
			{ID: "28", Type: "Collector Box"},
			{ID: "36", Type: "Starter Deck"},
		},
		FloorLangs: []string{"2"},
		MainCharTokens: []string{
			"jinx", "vi -", "caitlyn", "ekko", "jayce", "viktor", "ahri", "yasuo",
			"yone", "teemo", "sett", "lee sin", "volibear", "miss fortune",
			"kai'sa", "lux", "garen", "darius", "leona", "diana", "akali",
			"annie", "ashe", "zed", "katarina", "draven", "heimerdinger",
			"warwick", "thresh", "seraphine",
		},
		DefaultSet:       "UNL",
		DefaultTrackSets: []string{"UNL", "SFD", "OGS", "OGN", "ROPP", "OGN-PR"},
		Market:           riftboundMarket(),
		Challenged:       true,
		MyP: &MyP{
			Slug:     "riftbound",
			GAPrefix: "riftbound",
			// MyP shares TCGCSV's abbreviations except Spiritforged, which it
			// transposes to SPF. OPP/PR match riftboundGroupSet's remapping.
			SetAliases: map[string]string{
				"SPF": "SFD",
				"OPP": "ROPP",
				"PR":  "OGN-PR",
			},
		},
	}
}

// Lorcana is the ligalorcana.com.br configuration. Liga's set acronyms are
// LOR1..LOR13 plus Q1/Q2/D100/DLPC1/DLPC2; TCGCSV abbreviates the numbered
// sets as bare digits, mapped in lorcanaGroupSet.
func Lorcana() Game {
	return Game{
		ID:              "lorcana",
		Name:            "Lorcana",
		LigaBaseURL:     "https://www.ligalorcana.com.br/",
		LigaHosts:       []string{"www.ligalorcana.com.br", "ligalorcana.com.br"},
		SourceName:      "ligalorcana",
		ImageURLRe:      lorcanaImageRe,
		SealedSetCodeRe: nil,
		SealedCategories: []SealedCategory{
			{ID: "10", Type: "Booster Box"},
			{ID: "21", Type: "Booster Pack"},
			{ID: "28", Type: "Collector Box"},
			{ID: "36", Type: "Starter Deck"},
		},
		FloorLangs: []string{"2"},
		MainCharTokens: []string{
			"mickey", "minnie", "donald", "goofy", "elsa", "anna -", "olaf",
			"stitch", "maleficent", "ursula", "scar", "hades", "ariel", "belle",
			"beast", "rapunzel", "mulan", "aladdin", "jasmine", "genie", "jafar",
			"simba", "mufasa", "moana", "maui", "tiana", "cinderella", "aurora",
			"peter pan", "tinker bell", "captain hook", "cruella", "gaston",
			"hercules", "alice", "baymax",
		},
		DefaultSet: "LOR12",
		DefaultTrackSets: []string{
			"LOR13", "LOR12", "LOR11", "LOR10", "LOR9", "LOR8", "LOR7", "LOR6",
			"LOR5", "LOR4", "LOR3", "LOR2", "LOR1", "Q2", "Q1", "D100",
			"DLPC2", "DLPC1",
		},
		Market:     lorcanaMarket(),
		Challenged: true,
	}
}

// Gundam is the ligagundam.com.br configuration. Like One Piece, its card numbers
// are globally unique and set-prefixed ("GD01-001", "ST01-001") on both Liga and
// TCGCSV, so the deals match runs by number alone (no set-scoping). The BR market
// carries Japanese imports below the English card, so the floor is English-only.
func Gundam() Game {
	return Game{
		ID:                "gundam",
		Name:              "Gundam",
		LigaBaseURL:       "https://www.ligagundam.com.br/",
		LigaHosts:         []string{"www.ligagundam.com.br", "ligagundam.com.br"},
		SourceName:        "ligagundam",
		ImageURLRe:        gundamImageRe,
		SealedSetCodeRe:   nil,
		UniqueCardNumbers: true,
		SealedCategories: []SealedCategory{
			{ID: "10", Type: "Booster Box"},
			{ID: "21", Type: "Booster Pack"},
			{ID: "28", Type: "Collector Box"},
			{ID: "36", Type: "Starter Deck"},
		},
		FloorLangs: []string{"2"},
		MainCharTokens: []string{
			"gundam", "amuro", "char", "zaku", "rx-78", "nu gundam", "sazabi",
			"unicorn", "banshee", "wing", "freedom", "strike", "justice", "aerial",
			"suletta", "barbatos", "exia", "zeon", "gouf", "dom", "qubeley",
			"kshatriya", "sinanju", "nightingale",
		},
		DefaultSet: "GD04",
		DefaultTrackSets: []string{
			"GD04", "GD03", "GD02", "GD01", "EB01",
			"ST10", "ST09", "ST08", "ST07", "ST06",
			"ST05", "ST04", "ST03", "ST02", "ST01", "BETA",
		},
		Market:     gundamMarket(),
		Challenged: true,
	}
}

// Pokemon is the ligapokemon.com.br configuration. The Liga market for Pokémon is
// multi-language (EN/JP/PT), so the floor is NOT restricted to English. Set codes
// are Liga's own acronyms, not official TCG codes. Defaults track recent, liquid
// expansions; users widen via -pkm-track-set.
func Pokemon() Game {
	return Game{
		ID:              "pokemon",
		Name:            "Pokémon",
		LigaBaseURL:     "https://www.ligapokemon.com.br/",
		LigaHosts:       []string{"www.ligapokemon.com.br", "ligapokemon.com.br"},
		SourceName:      "ligapokemon",
		ImageURLRe:      pokemonImageRe,
		SealedSetCodeRe: nil,
		BuyoutLangs:     []string{"2", "8"},
		SealedCategories: []SealedCategory{
			{ID: "10", Type: "Booster Box"},
			{ID: "14", Type: "Bundle"},
			{ID: "21", Type: "Booster Pack"},
			{ID: "25", Type: "Blister"},
			{ID: "27", Type: "Elite Trainer Box"},
			{ID: "28", Type: "Collector Box"},
			{ID: "36", Type: "Starter Deck"},
		},
		FloorLangs: nil,
		MainCharTokens: []string{
			"pikachu", "charizard", "mewtwo", "mew", "eevee", "umbreon", "sylveon",
			"espeon", "vaporeon", "jolteon", "flareon", "glaceon", "leafeon",
			"rayquaza", "lucario", "gengar", "gardevoir", "greninja", "lugia",
			"giratina", "dragonite", "gyarados", "snorlax", "tyranitar", "garchomp",
			"blastoise", "venusaur", "arcanine", "sylveon", "dialga", "palkia",
		},
		DefaultSet: "DRI",
		DefaultTrackSets: []string{
			"PBL", "M5", "CRI", "POR", "M4", "ASC", "M3", "M2a", "PFL", "m2", "MEG",
			"M1S", "M1L", "WHT", "BLK", "DRI", "SV10", "JTG",
		},
	}
}

// ByID returns the configured game for an id, defaulting to One Piece.
func ByID(id string) Game {
	switch id {
	case "pokemon":
		return Pokemon()
	case "riftbound":
		return Riftbound()
	case "lorcana":
		return Lorcana()
	case "gundam":
		return Gundam()
	}
	return OnePiece()
}
