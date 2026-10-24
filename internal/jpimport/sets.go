package jpimport

import "strings"

type Set struct {
	JP   string   `json:"jp"`
	Name string   `json:"name"`
	Era  string   `json:"era"`
	West []string `json:"west"`
}

var Sets = []Set{
	{JP: "M6a", Name: "30th Celebration", Era: "mega", West: []string{"30C"}},
	{JP: "M6", Name: "Storm Emeralda", Era: "mega"},
	{JP: "M5", Name: "Abyss Eye", Era: "mega", West: []string{"PBL"}},
	{JP: "M4", Name: "Ninja Spinner", Era: "mega", West: []string{"CRI"}},
	{JP: "M3", Name: "Nihil Zero", Era: "mega", West: []string{"POR"}},
	{JP: "M2a", Name: "MEGA Dream ex", Era: "mega", West: []string{"ASC"}},
	{JP: "m2", Name: "Inferno X", Era: "mega", West: []string{"PFL"}},
	{JP: "M1S", Name: "Mega Symphonia", Era: "mega", West: []string{"MEG"}},
	{JP: "M1L", Name: "Mega Brave", Era: "mega", West: []string{"MEG"}},
	{JP: "SV11W", Name: "White Flare", Era: "sv", West: []string{"WHT"}},
	{JP: "SV11B", Name: "Black Bolt", Era: "sv", West: []string{"BLK"}},
	{JP: "SV10", Name: "Glory of the Rocket Gang", Era: "sv", West: []string{"DRI"}},
	{JP: "SV9A", Name: "Hot Wind Arena", Era: "sv", West: []string{"DRI", "JTG"}},
	{JP: "SV9", Name: "Battle Partners", Era: "sv", West: []string{"JTG"}},
	{JP: "SV8a", Name: "Terastal Fest ex", Era: "sv", West: []string{"PRE"}},
	{JP: "SV8", Name: "Super Electric Breaker", Era: "sv", West: []string{"SSP"}},
	{JP: "SV7a", Name: "Paradise Dragona", Era: "sv", West: []string{"SSP", "SCR"}},
	{JP: "SV7", Name: "Stellar Miracle", Era: "sv", West: []string{"SCR"}},
	{JP: "SV6A", Name: "Night Wanderer", Era: "sv", West: []string{"SFA", "TWM"}},
	{JP: "SV6", Name: "Transformation Mask", Era: "sv", West: []string{"TWM", "SFA"}},
	{JP: "sv5a", Name: "Crimson Haze", Era: "sv", West: []string{"TWM", "SFA"}},
	{JP: "SV5M", Name: "Cyber Judge", Era: "sv", West: []string{"TEF"}},
	{JP: "SV5K", Name: "Wild Force", Era: "sv", West: []string{"TEF"}},
	{JP: "sv4a", Name: "Shiny Treasure ex", Era: "sv", West: []string{"PAF"}},
	{JP: "SV4M", Name: "Future Flash", Era: "sv", West: []string{"PAR"}},
	{JP: "SV4K", Name: "Ancient Roar", Era: "sv", West: []string{"PAR"}},
	{JP: "sv3a", Name: "Raging Surf", Era: "sv", West: []string{"PAR", "OBF"}},
	{JP: "SV3", Name: "Ruler of the Black Flame", Era: "sv", West: []string{"OBF"}},
	{JP: "sV2a", Name: "Pokémon Card 151", Era: "sv", West: []string{"MEW"}},
	{JP: "sv2P", Name: "Snow Hazard", Era: "sv", West: []string{"PAL"}},
	{JP: "sv2D", Name: "Clay Burst", Era: "sv", West: []string{"PAL"}},
	{JP: "sv1a", Name: "Triplet Beat", Era: "sv", West: []string{"PAL", "OBF"}},
	{JP: "sv1V", Name: "Violet ex", Era: "sv", West: []string{"SV1"}},
	{JP: "sv1S", Name: "Scarlet ex", Era: "sv", West: []string{"SV1"}},
}

var sirStart = map[string]int{"30C": 147}

func SetFor(code string) (Set, bool) {
	for _, s := range Sets {
		if strings.EqualFold(s.JP, code) {
			return s, true
		}
	}
	return Set{}, false
}

func LigaSets() []string {
	seen := map[string]bool{}
	var out []string
	add := func(code string) {
		if !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
	}
	for _, s := range Sets {
		add(s.JP)
		for _, w := range s.West {
			add(w)
		}
	}
	return out
}
