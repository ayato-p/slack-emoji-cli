package render

import (
	"image"
	"image/color"

	"github.com/fogleman/gg"
)

const (
	canvasSize = 128
	padding    = 8
	drawArea   = canvasSize - padding*2 // 112px
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

	// Determine starting font size based on number of lines
	n := len(lines)
	startSize := 52.0 - float64(n-1)*16.0
	if startSize < 12 {
		startSize = 12
	}

	// Find the largest font size where all lines fit within the draw area
	fontSize := startSize
	for fontSize >= 6 {
		face, err := loadFace(fontSize)
		if err != nil {
			return nil, err
		}
		ctx.SetFontFace(face)

		fits := true
		for _, line := range lines {
			w, _ := ctx.MeasureString(line)
			if w > drawArea {
				fits = false
				break
			}
		}
		if fits {
			break
		}
		fontSize -= 1
	}

	// Draw text centered
	ctx.SetColor(color.White)
	lineHeight := fontSize * 1.3

	for i, line := range lines {
		y := canvasSize/2 + (float64(i)-float64(n-1)/2)*lineHeight
		ctx.DrawStringAnchored(line, canvasSize/2, y, 0.5, 0.5)
	}

	return ctx.Image(), nil
}
