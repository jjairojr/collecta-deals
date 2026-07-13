package cardimg

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"opdeals/internal/game"
)

func TestImageURLRegex(t *testing.T) {
	imageURLRe := game.OnePiece().ImageURLRe
	html := []byte(`<div><img class="card-image" src="//repositorio.sbrauble.com/arquivos/in/onepiece/81/6a1dca76e46e6-yfslg-grx0m-200fffdb00ce250934ce05ce833b12f7.jpg" alt="x"></div>`)
	m := imageURLRe.Find(html)
	if m == nil {
		t.Fatal("expected a match")
	}
	want := "//repositorio.sbrauble.com/arquivos/in/onepiece/81/6a1dca76e46e6-yfslg-grx0m-200fffdb00ce250934ce05ce833b12f7.jpg"
	if string(m) != want {
		t.Errorf("got %q, want %q", m, want)
	}
}

func TestImageURLRegexNoMatch(t *testing.T) {
	imageURLRe := game.OnePiece().ImageURLRe
	html := []byte(`<img src="//www.lmcorp.com.br/arquivos/img/logo_new_tcg_11.jpg">`)
	if imageURLRe.Find(html) != nil {
		t.Error("logo image should not match the card-art pattern")
	}
}

func TestPokemonImageURLRegex(t *testing.T) {
	imageURLRe := game.Pokemon().ImageURLRe
	html := []byte(`<img src="//repositorio.sbrauble.com/arquivos/in/pokemon_bkp/cd/700/6810f63a66a9f-g7hu2-24bo6-03670ba8f67d0ef732dfa5ddf8068a49.jpg">`)
	if imageURLRe.Find(html) == nil {
		t.Fatal("expected a Pokemon card-art match")
	}
}

func solidJPEG(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 350, 489))
	for y := 0; y < 489; y++ {
		for x := 0; x < 350; x++ {
			im.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, im, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestGrid(t *testing.T) {
	imgs := [][]byte{
		solidJPEG(t, color.RGBA{200, 30, 30, 255}),
		solidJPEG(t, color.RGBA{30, 200, 30, 255}),
		solidJPEG(t, color.RGBA{30, 30, 200, 255}),
	}
	out, err := Grid(imgs, 0)
	if err != nil {
		t.Fatalf("Grid: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("output is not a valid PNG: %v", err)
	}
	// 3 images -> defaultCols=2, rows=2
	wantW := 2*cellWidth + 3*gridPad
	wantH := 2*cellHeight + 3*gridPad
	if b := img.Bounds(); b.Dx() != wantW || b.Dy() != wantH {
		t.Errorf("dimensions %dx%d, want %dx%d", b.Dx(), b.Dy(), wantW, wantH)
	}
}

func TestGridEmpty(t *testing.T) {
	if _, err := Grid([][]byte{[]byte("not a jpeg")}, 0); err == nil {
		t.Error("expected error when no images decode")
	}
}
