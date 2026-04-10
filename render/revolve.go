package render

import (
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"math"

	"github.com/fogleman/gg"
)

// RevolveGIF generates an animated GIF where each character revolves clockwise
// around the canvas center. Characters remain upright throughout the animation.
// One full 360° revolution equals one GIF cycle.
func RevolveGIF(lines []string, bgColor color.Color) (*gif.GIF, error) {
	face, ascent, descent, err := findFontAndMetrics(lines)
	if err != nil {
		return nil, err
	}

	// Compute baseline positions (same formula as Render and RenderGIF)
	n := len(lines)
	lineH := ascent + descent
	firstAscent, _ := glyphBounds(face, lines[0])
	_, lastDescent := glyphBounds(face, lines[n-1])
	baseline0 := (canvasSize - float64(n-1)*lineH + firstAscent - lastDescent) / 2

	// Measure individual character widths using a temporary context
	tmpCtx := gg.NewContext(canvasSize, canvasSize)
	tmpCtx.SetFontFace(face)

	type charOrbit struct {
		s      string
		theta0 float64 // initial angle from canvas center (radians)
		radius float64 // orbit radius from canvas center
	}

	cx, cy := canvasSize/2.0, canvasSize/2.0
	halfMetricSpan := (ascent - descent) / 2 // offset from visual center to baseline

	var orbits []charOrbit
	for row, line := range lines {
		baseline := baseline0 + float64(row)*lineH
		yOrbit := baseline - halfMetricSpan // visual vertical center of this row

		lineWidth, _ := tmpCtx.MeasureString(line)
		xCursor := cx - lineWidth/2

		for _, r := range []rune(line) {
			s := string(r)
			charWidth, _ := tmpCtx.MeasureString(s)
			xCenter := xCursor + charWidth/2

			dx := xCenter - cx
			dy := yOrbit - cy
			orbits = append(orbits, charOrbit{
				s:      s,
				theta0: math.Atan2(dy, dx),
				radius: math.Hypot(dx, dy),
			})
			xCursor += charWidth
		}
	}

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

		offset := float64(i) / float64(numFrames) * 2 * math.Pi

		for _, ch := range orbits {
			a := ch.theta0 + offset
			newX := cx + ch.radius*math.Cos(a)
			newOrbitY := cy + ch.radius*math.Sin(a)
			newBaseline := newOrbitY + halfMetricSpan
			ctx.DrawStringAnchored(ch.s, newX, newBaseline, 0.5, 0)
		}

		// Convert RGBA frame to paletted image for GIF encoding
		img := ctx.Image()
		paletted := image.NewPaletted(img.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(paletted, img.Bounds(), img, image.Point{})

		g.Image = append(g.Image, paletted)
		g.Delay = append(g.Delay, frameDelay)
	}

	return g, nil
}
