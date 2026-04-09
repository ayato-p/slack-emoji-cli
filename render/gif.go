package render

import (
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"

	"github.com/fogleman/gg"
)

const (
	numFrames  = 24 // frames per full animation cycle
	frameDelay = 4  // 40ms per frame → ~1 second per revolution
)

// RenderGIF generates an animated GIF with the given lines of text on a solid
// background color. Each frame has all transformers applied in order, enabling
// composable animations.
func RenderGIF(lines []string, bgColor color.Color, transformers []Transformer) (*gif.GIF, error) {
	face, ascent, descent, err := findFontAndMetrics(lines)
	if err != nil {
		return nil, err
	}

	n := len(lines)
	lineH := ascent + descent
	firstAscent, _ := glyphBounds(face, lines[0])
	_, lastDescent := glyphBounds(face, lines[n-1])
	baseline0 := (canvasSize - float64(n-1)*lineH + firstAscent - lastDescent) / 2

	g := &gif.GIF{LoopCount: 0}

	for i := 0; i < numFrames; i++ {
		ctx := gg.NewContext(canvasSize, canvasSize)

		// Fill background
		cr, cg, cb, ca := bgColor.RGBA()
		ctx.SetRGBA(
			float64(cr)/0xffff,
			float64(cg)/0xffff,
			float64(cb)/0xffff,
			float64(ca)/0xffff,
		)
		ctx.Clear()

		ctx.SetFontFace(face)
		ctx.SetColor(color.White)

		// Apply transformers (e.g. rotation) then draw text
		ctx.Push()
		for _, t := range transformers {
			t(ctx, i, numFrames)
		}
		for j, line := range lines {
			baseline := baseline0 + float64(j)*lineH
			ctx.DrawStringAnchored(line, canvasSize/2, baseline, 0.5, 0)
		}
		ctx.Pop()

		// Convert RGBA frame to paletted image for GIF encoding
		img := ctx.Image()
		paletted := image.NewPaletted(img.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(paletted, img.Bounds(), img, image.Point{})

		g.Image = append(g.Image, paletted)
		g.Delay = append(g.Delay, frameDelay)
	}

	return g, nil
}
