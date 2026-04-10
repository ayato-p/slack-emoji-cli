package render

import (
	"image"
	"image/color"
	"math"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

const (
	canvasSize = 128
	padding    = 2
	drawArea   = canvasSize - padding*2 // 124px
)

// findFontAndMetrics finds the largest font size where all lines fit within the
// draw area and returns the font face, ascent, and descent in points.
func findFontAndMetrics(lines []string) (font.Face, float64, float64, error) {
	ctx := gg.NewContext(canvasSize, canvasSize)
	fontSize := 120.0
	var face font.Face
	for fontSize >= 6 {
		var err error
		face, err = loadFace(fontSize)
		if err != nil {
			return nil, 0, 0, err
		}
		ctx.SetFontFace(face)

		m := face.Metrics()
		ascent := float64(m.Ascent) / 64
		descent := float64(m.Descent) / 64
		totalHeight := (ascent + descent) * float64(len(lines))

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
	m := face.Metrics()
	return face, float64(m.Ascent) / 64, float64(m.Descent) / 64, nil
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

// rendererSpec accumulates rendering options before the renderer is built.
type rendererSpec struct {
	lines     []string
	bgColor   color.Color
	effects   []frameEffect
	isRevolve bool
}

// rendererOption configures a rendererSpec.
type rendererOption func(*rendererSpec)

func withLines(lines []string) rendererOption {
	return func(s *rendererSpec) { s.lines = lines }
}

func withBg(c color.Color) rendererOption {
	return func(s *rendererSpec) { s.bgColor = c }
}

func withScrollX(reverse bool) rendererOption {
	return func(s *rendererSpec) { s.effects = append(s.effects, scrollXEffect(reverse)) }
}

func withScrollY(reverse bool) rendererOption {
	return func(s *rendererSpec) { s.effects = append(s.effects, scrollYEffect(reverse)) }
}

func withRotate(reverse bool) rendererOption {
	return func(s *rendererSpec) { s.effects = append(s.effects, rotateEffect(reverse)) }
}

func withRevolve(reverse bool) rendererOption {
	return func(s *rendererSpec) {
		s.isRevolve = true
		s.effects = append(s.effects, revolveEffect(reverse))
	}
}

// buildRenderer applies all options and returns a per-frame render closure.
// When called with frame=0, total=1 it produces a static image (no animation).
func buildRenderer(opts ...rendererOption) (func(frame, total int) (image.Image, error), error) {
	spec := &rendererSpec{}
	for _, o := range opts {
		o(spec)
	}
	effect := composeEffects(spec.effects...)
	if spec.isRevolve {
		return newRevolveRenderer(spec.lines, spec.bgColor, effect)
	}
	return newTextRenderer(spec.lines, spec.bgColor, effect)
}

// newTextRenderer returns a closure that renders text with effects (rotation, scrolling, etc.).
// The closure captures precomputed font metrics and computes per-frame transforms using the effect.
func newTextRenderer(
	lines []string,
	bgColor color.Color,
	effect frameEffect,
) (func(frame, total int) (image.Image, error), error) {
	face, ascent, descent, err := findFontAndMetrics(lines)
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

		if params.ScrollX != 0 || params.ScrollY != 0 {
			// Two-pass: render content to offscreen, then composite with wrap.
			// This allows seamless wrapping where content that exits one edge
			// re-enters from the opposite edge.
			offscreen := gg.NewContext(canvasSize, canvasSize)
			offscreen.SetFontFace(face)
			offscreen.SetColor(color.White)
			offscreen.Push()
			if params.RotationAngle != 0 {
				offscreen.RotateAbout(params.RotationAngle, canvasSize/2, canvasSize/2)
			}
			for i, line := range lines {
				baseline := baseline0 + float64(i)*lineH
				offscreen.DrawStringAnchored(line, canvasSize/2, baseline, 0.5, 0)
			}
			offscreen.Pop()
			compositeWithWrap(ctx, offscreen.Image(), params.ScrollX, params.ScrollY)
		} else {
			ctx.SetFontFace(face)
			ctx.SetColor(color.White)
			ctx.Push()
			if params.RotationAngle != 0 {
				ctx.RotateAbout(params.RotationAngle, canvasSize/2, canvasSize/2)
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
