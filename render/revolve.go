package render

import (
	"image"
	"image/color"
	"math"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// newRevolveRenderer returns a closure that renders revolve animation frames.
// Characters are arranged in a circle around the canvas center and revolve.
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
// One full 360° revolution equals one animation cycle.
func newRevolveRenderer(
	lines []string,
	bgColor color.Color,
	fontColor color.Color,
	effect frameEffect,
	f *opentype.Font,
) (func(frame, total int) (image.Image, error), error) {
	// Flatten all characters from all lines into one slice (reading order)
	var chars []string
	for _, line := range lines {
		for _, r := range []rune(line) {
			chars = append(chars, string(r))
		}
	}
	N := len(chars)
	if N == 0 {
		return func(frame, total int) (image.Image, error) {
			ctx := gg.NewContext(canvasSize, canvasSize)
			r, g, b, a := bgColor.RGBA()
			ctx.SetRGBA(float64(r)/0xffff, float64(g)/0xffff, float64(b)/0xffff, float64(a)/0xffff)
			ctx.Clear()
			return ctx.Image(), nil
		}, nil
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
		faceAtSize, err := loadFace(f, fontSize)
		if err != nil {
			return nil, err
		}
		tmpCtx.SetFontFace(faceAtSize)
		m := faceAtSize.Metrics()
		a := float64(m.Ascent) / 64
		d := float64(m.Descent) / 64

		var maxW float64
		for _, s := range chars {
			w, _ := tmpCtx.MeasureString(s)
			if w > maxW {
				maxW = w
			}
		}

		face = faceAtSize
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

	return func(frame, total int) (image.Image, error) {
		params := effect(frame, total)

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
		ctx.SetColor(fontColor)

		// Compute orbit offset from animation parameters.
		offset := params.RevolveOffset

		if params.ScrollX != 0 || params.ScrollY != 0 {
			// Two-pass: render revolve frame to transparent offscreen, then
			// composite with wrap-around so content re-enters from the opposite edge.
			offscreen := gg.NewContext(canvasSize, canvasSize)
			offscreen.SetFontFace(face)
			offscreen.SetColor(fontColor)
			for k, s := range chars {
				theta := startAngle - float64(k)*(2*math.Pi/float64(N)) + offset
				x := cx + r*math.Cos(theta)
				yOrbit := cy + r*math.Sin(theta)
				baseline := yOrbit + halfMetricSpan
				offscreen.DrawStringAnchored(s, x, baseline, 0.5, 0)
			}
			compositeWithWrap(ctx, offscreen.Image(), params.ScrollX, params.ScrollY)
		} else {
			for k, s := range chars {
				theta := startAngle - float64(k)*(2*math.Pi/float64(N)) + offset
				x := cx + r*math.Cos(theta)
				yOrbit := cy + r*math.Sin(theta)
				baseline := yOrbit + halfMetricSpan
				ctx.DrawStringAnchored(s, x, baseline, 0.5, 0)
			}
		}

		return ctx.Image(), nil
	}, nil
}

