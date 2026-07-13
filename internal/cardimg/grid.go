package cardimg

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
)

const (
	cellWidth  = 200
	cellHeight = 279
	gridPad    = 14
)

var gridBackground = color.RGBA{R: 15, G: 23, B: 42, A: 255}

// Grid composes the decoded JPEG card images into a padded PNG grid on a dark
// background. Images that fail to decode are skipped. cols <= 0 auto-picks a
// column count from the image total.
func Grid(images [][]byte, cols int) ([]byte, error) {
	decoded := make([]image.Image, 0, len(images))
	for _, b := range images {
		im, err := jpeg.Decode(bytes.NewReader(b))
		if err != nil {
			continue
		}
		decoded = append(decoded, im)
	}
	if len(decoded) == 0 {
		return nil, fmt.Errorf("no decodable images")
	}
	if cols <= 0 {
		cols = defaultCols(len(decoded))
	}
	rows := (len(decoded) + cols - 1) / cols

	canvasW := cols*cellWidth + (cols+1)*gridPad
	canvasH := rows*cellHeight + (rows+1)*gridPad
	canvas := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	fill(canvas, gridBackground)

	for i, im := range decoded {
		r, c := i/cols, i%cols
		x0 := gridPad + c*(cellWidth+gridPad)
		y0 := gridPad + r*(cellHeight+gridPad)
		drawScaled(canvas, im, x0, y0, cellWidth, cellHeight)
	}

	var out bytes.Buffer
	if err := png.Encode(&out, canvas); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func defaultCols(n int) int {
	c := int(math.Ceil(math.Sqrt(float64(n))))
	if c > 6 {
		c = 6
	}
	if c < 1 {
		c = 1
	}
	return c
}

func fill(dst *image.RGBA, c color.RGBA) {
	b := dst.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.SetRGBA(x, y, c)
		}
	}
}

// drawScaled area-averages src down into a tw x th cell at (x0,y0), preserving
// the card aspect ratio and centering it within the cell.
func drawScaled(dst *image.RGBA, src image.Image, x0, y0, tw, th int) {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw == 0 || sh == 0 {
		return
	}
	scale := math.Min(float64(tw)/float64(sw), float64(th)/float64(sh))
	fw := int(float64(sw) * scale)
	fh := int(float64(sh) * scale)
	if fw < 1 {
		fw = 1
	}
	if fh < 1 {
		fh = 1
	}
	offX := x0 + (tw-fw)/2
	offY := y0 + (th-fh)/2

	for y := 0; y < fh; y++ {
		sy0 := b.Min.Y + y*sh/fh
		sy1 := b.Min.Y + (y+1)*sh/fh
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < fw; x++ {
			sx0 := b.Min.X + x*sw/fw
			sx1 := b.Min.X + (x+1)*sw/fw
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var rr, gg, bb, count uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					cr, cg, cb, _ := src.At(sx, sy).RGBA()
					rr += uint64(cr)
					gg += uint64(cg)
					bb += uint64(cb)
					count++
				}
			}
			if count == 0 {
				count = 1
			}
			dst.SetRGBA(offX+x, offY+y, color.RGBA{
				R: uint8((rr / count) >> 8),
				G: uint8((gg / count) >> 8),
				B: uint8((bb / count) >> 8),
				A: 255,
			})
		}
	}
}
