package render

import (
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"math"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

// RevolveGIF generates an animated GIF where each character revolves around
// the canvas center. Characters remain upright throughout the animation.
//
// All characters from all input lines are flattened into a single sequence
// and placed at uniformly-spaced angles on a circle. Reading order maps to
// screen-clockwise positions starting from the top-left:
//
//	char 0 → top-left, char 1 → bottom-left, char 2 → bottom-right, char 3 → top-right, …
//
// For example, "ab\cd" (chars: a,b,c,d) gives initial layout:
//
//	a d
//	b c
//
// One full 360° revolution equals one GIF cycle.
// When reverse is true, the revolution direction is reversed.
func RevolveGIF(lines []string, bgColor color.Color, reverse bool, speed float64, transformers []Transformer) (*gif.GIF, error) {
	// Flatten all characters from all lines into one slice (reading order)
	var chars []string
	for _, line := range lines {
		for _, r := range []rune(line) {
			chars = append(chars, string(r))
		}
	}
	N := len(chars)
	if N == 0 {
		return nil, nil
	}

	// For N chars uniformly spaced on a circle of radius r, adjacent char
	// centers are 2r·sin(π/N) apart. To avoid overlap and stay within the
	// draw area the maximum allowed char size is:
	//   charMax ≤ drawArea · sin(π/N) / (1 + sin(π/N))
	// The orbit radius is then:
	//   r = drawArea/2 − charMax/2
	sinPN := math.Sin(math.Pi / float64(N))
	charMaxLimit := float64(drawArea) * sinPN / (1 + sinPN)

	// Find the largest font where every character fits within charMaxLimit
	var face font.Face
	var ascent, descent float64
	tmpCtx := gg.NewContext(canvasSize, canvasSize)
	for fontSize := 120.0; fontSize >= 6; fontSize-- {
		f, err := loadFace(fontSize)
		if err != nil {
			return nil, err
		}
		tmpCtx.SetFontFace(f)
		m := f.Metrics()
		a := float64(m.Ascent) / 64
		d := float64(m.Descent) / 64

		var maxW float64
		for _, s := range chars {
			w, _ := tmpCtx.MeasureString(s)
			if w > maxW {
				maxW = w
			}
		}

		face = f
		ascent = a
		descent = d
		if math.Max(maxW, a+d) <= charMaxLimit {
			break
		}
	}

	// Re-measure maxCharW at the final font size to compute orbit radius
	tmpCtx.SetFontFace(face)
	var maxCharW float64
	for _, s := range chars {
		w, _ := tmpCtx.MeasureString(s)
		if w > maxCharW {
			maxCharW = w
		}
	}
	charMax := math.Max(maxCharW, ascent+descent)
	r := float64(drawArea)/2 - charMax/2

	cx, cy := float64(canvasSize)/2, float64(canvasSize)/2

	actualDelay := int(math.Round(float64(frameDelay) / speed))
	if actualDelay < 1 {
		actualDelay = 1
	}

	// halfMetricSpan converts orbit-center Y back to baseline Y:
	//   baseline = yOrbit + halfMetricSpan
	halfMetricSpan := (ascent - descent) / 2

	// char k is placed at angle: startAngle − k·(2π/N)
	// Subtracting each step puts successive chars screen-clockwise
	// (TL→BL→BR→TR for N=4), so reading order wraps around the circle.
	// The animation adds +offset each frame, which moves chars in the
	// TL→TR→BR→BL direction (screen-counter-clockwise in atan2 terms,
	// but visually clockwise around the rectangle for N=4).
	const startAngle = -3 * math.Pi / 4

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

		// Rotate all positions by one full revolution over numFrames.
		// Reverse flag flips the direction.
		offset := float64(i) / float64(numFrames) * 2 * math.Pi
		if reverse {
			offset = -offset
		}

		ctx.Push()
		for _, t := range transformers {
			t(ctx, i, numFrames)
		}
		for k, s := range chars {
			theta := startAngle - float64(k)*(2*math.Pi/float64(N)) + offset
			x := cx + r*math.Cos(theta)
			yOrbit := cy + r*math.Sin(theta)
			baseline := yOrbit + halfMetricSpan
			ctx.DrawStringAnchored(s, x, baseline, 0.5, 0)
		}
		ctx.Pop()

		// Convert RGBA frame to paletted image for GIF encoding
		img := ctx.Image()
		paletted := image.NewPaletted(img.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(paletted, img.Bounds(), img, image.Point{})
		g.Image = append(g.Image, paletted)
		g.Delay = append(g.Delay, actualDelay)
	}

	return g, nil
}
