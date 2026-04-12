package render

import (
	"image"
	"image/color"
	"math"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	canvasSize = 128
	padding    = 2
	drawArea   = canvasSize - padding*2 // 124px
)

// hsvToRGB converts HSV (h in [0,360), s/v in [0,1]) to color.RGBA.
func hsvToRGB(h, s, v float64) color.RGBA {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{
		R: uint8((r+m)*255), G: uint8((g+m)*255), B: uint8((b+m)*255), A: 255,
	}
}

// findFontAndMetrics finds the largest font size where all lines fit within the
// draw area and returns the font face, ascent, and descent in points.
// Uses actual glyph bounds instead of font-declared metrics to handle decorative
// fonts (e.g., Dela Gothic) that declare larger metrics than their actual CJK glyphs.
func findFontAndMetrics(lines []string, f *opentype.Font) (font.Face, float64, float64, error) {
	ctx := gg.NewContext(canvasSize, canvasSize)
	fontSize := 120.0
	var face font.Face
	for fontSize >= 6 {
		var err error
		face, err = loadFace(f, fontSize)
		if err != nil {
			return nil, 0, 0, err
		}
		ctx.SetFontFace(face)

		// Use actual glyph bounds instead of font-declared metrics
		var maxAscent, maxDescent float64
		for _, line := range lines {
			a, d := glyphBounds(face, line)
			if a > maxAscent {
				maxAscent = a
			}
			if d > maxDescent {
				maxDescent = d
			}
		}
		lineH := maxAscent + maxDescent
		totalHeight := lineH * float64(len(lines))

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
	// Return max glyph bounds (actual ink extents, consistent with fitting check)
	var maxAscent, maxDescent float64
	for _, line := range lines {
		a, d := glyphBounds(face, line)
		if a > maxAscent {
			maxAscent = a
		}
		if d > maxDescent {
			maxDescent = d
		}
	}
	return face, maxAscent, maxDescent, nil
}

// glyphBounds returns the actual glyph ascent and descent for a string using
// font.Drawer.BoundString, which reflects the real rendered extent of the glyphs
// rather than the font-wide metrics that may include space not used by CJK characters.
func glyphBounds(face font.Face, s string) (glyphAscent, glyphDescent float64) {
	d := &font.Drawer{Face: face}
	bounds, _ := d.BoundString(s)
	return float64(-bounds.Min.Y) / 64, float64(bounds.Max.Y) / 64
}

// FrameParams encapsulates per-frame animation state contributed by all effects.
type FrameParams struct {
	RotationAngle float64 // canvas rotation in radians; 0 = no rotation
	RevolveOffset float64 // orbit offset angle in radians; 0 = starting position
	ScrollX       float64 // horizontal pixel offset for wrap-around; 0 = no scroll
	ScrollY       float64 // vertical pixel offset for wrap-around; 0 = no scroll
	SizeScale     float64 // scale delta from 1.0: 0=1x (identity), +1.0=2x, -0.5=0.5x
}

// frameEffect computes per-frame animation parameters.
type frameEffect func(frame, total int) FrameParams

// composeEffects combines multiple effects by summing their contributions.
func composeEffects(effects ...frameEffect) frameEffect {
	return func(frame, total int) FrameParams {
		var p FrameParams
		for _, e := range effects {
			ep := e(frame, total)
			p.RotationAngle += ep.RotationAngle
			p.RevolveOffset += ep.RevolveOffset
			p.ScrollX += ep.ScrollX
			p.ScrollY += ep.ScrollY
			p.SizeScale += ep.SizeScale
		}
		return p
	}
}

// scrollXEffect returns an effect function for horizontal scrolling.
func scrollXEffect(reverse bool) frameEffect {
	return func(frame, total int) FrameParams {
		progress := float64(frame) / float64(total)
		dx := -progress * canvasSize // default: right-to-left (negative)
		if reverse {
			dx = -dx // reverse: left-to-right (positive)
		}
		return FrameParams{ScrollX: dx}
	}
}

// scrollYEffect returns an effect function for vertical scrolling.
func scrollYEffect(reverse bool) frameEffect {
	return func(frame, total int) FrameParams {
		progress := float64(frame) / float64(total)
		dy := -progress * canvasSize // default: bottom-to-top (negative)
		if reverse {
			dy = -dy // reverse: top-to-bottom (positive)
		}
		return FrameParams{ScrollY: dy}
	}
}

// rotateEffect returns an effect function for text rotation.
func rotateEffect(reverse bool) frameEffect {
	return func(frame, total int) FrameParams {
		angle := float64(frame) / float64(total) * 2 * math.Pi
		if reverse {
			angle = -angle
		}
		return FrameParams{RotationAngle: angle}
	}
}

// revolveEffect returns an effect function for character revolution.
func revolveEffect(reverse bool) frameEffect {
	return func(frame, total int) FrameParams {
		angle := float64(frame) / float64(total) * 2 * math.Pi
		if reverse {
			angle = -angle
		}
		return FrameParams{RevolveOffset: angle}
	}
}

// pulseEffect returns an effect function for scale pulsing.
// Forward: first half expands 1.0→2.0→1.0, second half shrinks 1.0→0.5→1.0.
// Reverse: second half (shrink) runs first, then first half (expand).
func pulseEffect(reverse bool) frameEffect {
	return func(frame, total int) FrameParams {
		half := total / 2
		f := frame
		if reverse {
			f = (frame + half) % total
		}
		var delta float64
		if f < half {
			delta = math.Sin(math.Pi * float64(f) / float64(half)) // 0→+1.0→0
		} else {
			delta = -0.5 * math.Sin(math.Pi * float64(f-half) / float64(half)) // 0→-0.5→0
		}
		return FrameParams{SizeScale: delta}
	}
}

// rendererSpec accumulates rendering options before the renderer is built.
type rendererSpec struct {
	lines       []string
	bgColor     color.Color
	fontColor   color.Color
	colorEffect func(frame, total int) color.Color
	effects     []frameEffect
	isRevolve   bool
}

// rendererOption configures a rendererSpec.
type rendererOption func(*rendererSpec)

func withLines(lines []string) rendererOption {
	return func(s *rendererSpec) { s.lines = lines }
}

func withBg(c color.Color) rendererOption {
	return func(s *rendererSpec) { s.bgColor = c }
}

func withFontColor(c color.Color) rendererOption {
	return func(s *rendererSpec) { s.fontColor = c }
}

func withScrollX(reverse *bool) rendererOption {
	if reverse == nil {
		return nil
	}
	return func(s *rendererSpec) { s.effects = append(s.effects, scrollXEffect(*reverse)) }
}

func withScrollY(reverse *bool) rendererOption {
	if reverse == nil {
		return nil
	}
	return func(s *rendererSpec) { s.effects = append(s.effects, scrollYEffect(*reverse)) }
}

func withRotate(reverse *bool) rendererOption {
	if reverse == nil {
		return nil
	}
	return func(s *rendererSpec) { s.effects = append(s.effects, rotateEffect(*reverse)) }
}

func withRevolve(reverse *bool) rendererOption {
	if reverse == nil {
		return nil
	}
	return func(s *rendererSpec) {
		s.isRevolve = true
		s.effects = append(s.effects, revolveEffect(*reverse))
	}
}

func withPulse(reverse *bool) rendererOption {
	if reverse == nil {
		return nil
	}
	return func(s *rendererSpec) { s.effects = append(s.effects, pulseEffect(*reverse)) }
}

func withGaming() rendererOption {
	return func(s *rendererSpec) {
		s.colorEffect = func(frame, total int) color.Color {
			hue := float64(frame) / float64(total) * 360.0
			// 1-cycle brightness pulse per full rotation: smoother loop transition
			v := 0.875 + 0.125*math.Sin(2*math.Pi*float64(frame)/float64(total))
			return hsvToRGB(hue, 1.0, v)
		}
	}
}

// buildRenderer applies all options and returns a per-frame render closure.
// When called with frame=0, total=1 it produces a static image (no animation).
func buildRenderer(f *opentype.Font, opts ...rendererOption) (func(frame, total int) (image.Image, error), error) {
	spec := &rendererSpec{}
	for _, o := range opts {
		if o != nil {
			o(spec)
		}
	}
	effect := composeEffects(spec.effects...)
	if spec.isRevolve {
		return newRevolveRenderer(spec.lines, spec.bgColor, spec.fontColor, spec.colorEffect, effect, f)
	}
	return newTextRenderer(spec.lines, spec.bgColor, spec.fontColor, spec.colorEffect, effect, f)
}

// newTextRenderer returns a closure that renders text with effects (rotation, scrolling, etc.).
// The closure captures precomputed font metrics and computes per-frame transforms using the effect.
func newTextRenderer(
	lines []string,
	bgColor color.Color,
	fontColor color.Color,
	colorEffect func(frame, total int) color.Color,
	effect frameEffect,
	f *opentype.Font,
) (func(frame, total int) (image.Image, error), error) {
	face, ascent, descent, err := findFontAndMetrics(lines, f)
	if err != nil {
		return nil, err
	}

	n := len(lines)
	lineH := ascent + descent
	firstAscent, _ := glyphBounds(face, lines[0])
	_, lastDescent := glyphBounds(face, lines[n-1])
	baseline0 := (canvasSize - float64(n-1)*lineH + firstAscent - lastDescent) / 2

	return func(frame, total int) (image.Image, error) {
		params := effect(frame, total)

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

		// Determine frame-specific font color
		fc := fontColor
		if colorEffect != nil {
			fc = colorEffect(frame, total)
		}

		if params.ScrollX != 0 || params.ScrollY != 0 {
			// Two-pass: render content to offscreen, then composite with wrap.
			// This allows seamless wrapping where content that exits one edge
			// re-enters from the opposite edge.
			offscreen := gg.NewContext(canvasSize, canvasSize)
			offscreen.SetFontFace(face)
			offscreen.SetColor(fc)
			offscreen.Push()
			if params.RotationAngle != 0 || params.SizeScale != 0 {
				offscreen.Translate(canvasSize/2, canvasSize/2)
				if params.SizeScale != 0 {
					s := 1.0 + params.SizeScale
					offscreen.Scale(s, s)
				}
				if params.RotationAngle != 0 {
					offscreen.Rotate(params.RotationAngle)
				}
				offscreen.Translate(-canvasSize/2, -canvasSize/2)
			}
			for i, line := range lines {
				baseline := baseline0 + float64(i)*lineH
				offscreen.DrawStringAnchored(line, canvasSize/2, baseline, 0.5, 0)
			}
			offscreen.Pop()
			compositeWithWrap(ctx, offscreen.Image(), params.ScrollX, params.ScrollY)
		} else {
			ctx.SetFontFace(face)
			ctx.SetColor(fc)
			ctx.Push()
			if params.RotationAngle != 0 || params.SizeScale != 0 {
				ctx.Translate(canvasSize/2, canvasSize/2)
				if params.SizeScale != 0 {
					s := 1.0 + params.SizeScale
					ctx.Scale(s, s)
				}
				if params.RotationAngle != 0 {
					ctx.Rotate(params.RotationAngle)
				}
				ctx.Translate(-canvasSize/2, -canvasSize/2)
			}
			for i, line := range lines {
				baseline := baseline0 + float64(i)*lineH
				ctx.DrawStringAnchored(line, canvasSize/2, baseline, 0.5, 0)
			}
			ctx.Pop()
		}

		return ctx.Image(), nil
	}, nil
}
