package liga

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"regexp"
	"strconv"
	"strings"
)

const (
	glyphWidth   = 7
	glyphHeight  = 15
	maxGlyphDist = 20
	inkThreshold = 30
)

var (
	stylePosRe = regexp.MustCompile(`\.([A-Za-z0-9_-]+)\s*\{[^}]*background-position:\s*(-?\d+)px\s+(-?\d+)px`)
	quantURLRe = regexp.MustCompile(`url\(([^)]*imgunid[^)]*)\)`)
	priceURLRe = regexp.MustCompile(`url\(([^)]*imgnum[^)]*)\)`)
)

func styleClassPositions(html []byte) map[string][2]int {
	matches := stylePosRe.FindAllSubmatch(html, -1)
	out := make(map[string][2]int, len(matches))
	for _, m := range matches {
		x, err := strconv.Atoi(string(m[2]))
		if err != nil {
			continue
		}
		y, err := strconv.Atoi(string(m[3]))
		if err != nil {
			continue
		}
		out[string(m[1])] = [2]int{abs(x), abs(y)}
	}
	return out
}

func quantAtlasURL(html []byte) string { return atlasURL(html, quantURLRe) }
func priceAtlasURL(html []byte) string { return atlasURL(html, priceURLRe) }

func atlasURL(html []byte, re *regexp.Regexp) string {
	m := re.FindSubmatch(html)
	if m == nil {
		return ""
	}
	u := string(m[1])
	if strings.HasPrefix(u, "//") {
		return "https:" + u
	}
	return u
}

func decodeAtlas(data []byte) (image.Image, bool) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	return img, true
}

func decodeQuantity(atlas image.Image, classPos map[string][2]int, quantCss string) (int, bool) {
	if atlas == nil {
		return 0, false
	}
	num := 0
	found := 0
	for _, seg := range strings.Split(quantCss, ";") {
		for _, tok := range strings.Fields(seg) {
			pos, ok := classPos[tok]
			if !ok {
				continue
			}
			d, dist := classifyGlyph(atlas, pos[0], pos[1], &digitTemplates)
			if dist > maxGlyphDist {
				return 0, false
			}
			num = num*10 + d
			found++
			break
		}
	}
	if found == 0 {
		return 0, false
	}
	return num, true
}

// decodePrice reconstructs a BRL price from a precoCss string. Segments are
// ";"-separated; the "V" segment is the decimal comma. Brazilian prices always
// carry 2 decimals, so all digits are concatenated and divided by 100. Prices
// use the imgnum atlas + priceTemplates (a different font than quantities).
func decodePrice(atlas image.Image, classPos map[string][2]int, precoCss string) (float64, bool) {
	if atlas == nil {
		return 0, false
	}
	num := 0
	found := 0
	for _, seg := range strings.Split(precoCss, ";") {
		seg = strings.TrimSpace(seg)
		if seg == "" || seg == "V" {
			continue
		}
		for _, tok := range strings.Fields(seg) {
			pos, ok := classPos[tok]
			if !ok {
				continue
			}
			d, dist := classifyGlyph(atlas, pos[0], pos[1], &priceTemplates)
			if dist > maxGlyphDist {
				return 0, false
			}
			num = num*10 + d
			found++
			break
		}
	}
	if found == 0 {
		return 0, false
	}
	return float64(num) / 100, true
}

func classifyGlyph(atlas image.Image, x, y int, templates *[10][glyphHeight]uint8) (int, int) {
	q := glyphBits(atlas, x, y)
	best, bestDist := 0, 1<<30
	for d := 0; d < 10; d++ {
		if dist := shiftHamming(q, templates[d]); dist < bestDist {
			bestDist = dist
			best = d
		}
	}
	return best, bestDist
}

func glyphBits(atlas image.Image, x, y int) [glyphHeight]uint8 {
	var rows [glyphHeight]uint8
	b := atlas.Bounds()
	for dy := 0; dy < glyphHeight; dy++ {
		var row uint8
		for dx := 0; dx < glyphWidth; dx++ {
			px, py := x+dx, y+dy
			if px < b.Min.X || px >= b.Max.X || py < b.Min.Y || py >= b.Max.Y {
				continue
			}
			r, g, bl, _ := atlas.At(px, py).RGBA()
			lum := (299*(r>>8) + 587*(g>>8) + 114*(bl>>8)) / 1000
			if lum > inkThreshold {
				row |= 1 << uint(dx)
			}
		}
		rows[dy] = row
	}
	return rows
}

func shiftHamming(q, tmpl [glyphHeight]uint8) int {
	best := 1 << 30
	for _, dy := range []int{-2, -1, 0, 1, 2} {
		for _, dx := range []int{-1, 0, 1} {
			d := 0
			for y := 0; y < glyphHeight; y++ {
				for x := 0; x < glyphWidth; x++ {
					if bit(tmpl, x, y) != bit(q, x+dx, y+dy) {
						d++
					}
				}
			}
			if d < best {
				best = d
			}
		}
	}
	return best
}

func bit(rows [glyphHeight]uint8, x, y int) uint8 {
	if x < 0 || x >= glyphWidth || y < 0 || y >= glyphHeight {
		return 0
	}
	return (rows[y] >> uint(x)) & 1
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
