package render

import (
	"image"
	"image/color"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

const (
	canvasSize = 128
	padding    = 2
	drawArea   = canvasSize - padding*2 // 124px
)

// Render generates a 128x128 PNG-compatible image with the given lines of text
// centered on a solid background color.
func Render(lines []string, bgColor color.Color) (image.Image, error) {
	ctx := gg.NewContext(canvasSize, canvasSize)

	// Fill background
	r, g, b, a := bgColor.RGBA()
	ctx.SetRGBA(
		float64(r)/0xffff,
		float64(g)/0xffff,
		float64(b)/0xffff,
		float64(a)/0xffff,
	)
	ctx.Clear()

	n := len(lines)

	// Find the largest font size where all lines fit within the draw area.
	// Use actual font metrics (ascent+descent) for height, not fontSize*factor,
	// because CJK fonts like Noto Sans JP have larger metrics than the font size.
	fontSize := 120.0
	var face font.Face
	for fontSize >= 6 {
		var err error
		face, err = loadFace(fontSize)
		if err != nil {
			return nil, err
		}
		ctx.SetFontFace(face)

		m := face.Metrics()
		ascent := float64(m.Ascent) / 64
		descent := float64(m.Descent) / 64
		totalHeight := (ascent + descent) * float64(n)

		fits := totalHeight <= drawArea
		if fits {
			for _, line := range lines {
				w, _ := ctx.MeasureString(line)
				if w > drawArea {
					fits = false
					break
				}
			}
		}
		if fits {
			break
		}
		fontSize--
	}

	// Compute baseline positions so glyphs are visually centered.
	//
	// For n lines spaced by lineH = ascent+descent:
	//   visual_top    = baseline_0 - ascent
	//   visual_bottom = baseline_{n-1} + descent
	//                 = baseline_0 + (n-1)*lineH + descent
	//
	// Setting (visual_top + visual_bottom) / 2 == canvasSize/2 gives:
	//   baseline_0 = (canvasSize - (n-1)*lineH + ascent - descent) / 2
	ctx.SetColor(color.White)
	m := face.Metrics()
	ascent := float64(m.Ascent) / 64
	descent := float64(m.Descent) / 64
	lineH := ascent + descent
	baseline0 := (canvasSize - float64(n-1)*lineH + ascent - descent) / 2

	for i, line := range lines {
		baseline := baseline0 + float64(i)*lineH
		// ay=0: y is used directly as the baseline (no vertical shift)
		ctx.DrawStringAnchored(line, canvasSize/2, baseline, 0.5, 0)
	}

	return ctx.Image(), nil
}
