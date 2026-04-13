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
	ScrollTileW   float64 // base tile width for x-scroll; 0 = no x-scroll
	ScrollTileH   float64 // base tile height for y-scroll; 0 = no y-scroll
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
			p.ScrollTileW += ep.ScrollTileW
			p.ScrollTileH += ep.ScrollTileH
		}
		return p
	}
}

// scrollXEffect returns an effect function for horizontal scrolling.
// tileWidth specifies the tile size for seamless wrapping (constant each frame).
func scrollXEffect(reverse bool, tileWidth float64) frameEffect {
	return func(frame, total int) FrameParams {
		progress := float64(frame) / float64(total)
		dx := -progress * tileWidth // default: right-to-left (negative)
		if reverse {
			dx = -dx // reverse: left-to-right (positive)
		}
		return FrameParams{ScrollX: dx, ScrollTileW: tileWidth}
	}
}

// scrollYEffect returns an effect function for vertical scrolling.
// tileHeight specifies the tile size for seamless wrapping (constant each frame).
func scrollYEffect(reverse bool, tileHeight float64) frameEffect {
	return func(frame, total int) FrameParams {
		progress := float64(frame) / float64(total)
		dy := -progress * tileHeight // default: bottom-to-top (negative)
		if reverse {
			dy = -dy // reverse: top-to-bottom (positive)
		}
		return FrameParams{ScrollY: dy, ScrollTileH: tileHeight}
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
	scrollX     *bool // nil=disabled, &false=forward, &true=reverse
	scrollY     *bool // nil=disabled, &false=forward, &true=reverse
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
	return func(s *rendererSpec) { s.scrollX = reverse }
}

func withScrollY(reverse *bool) rendererOption {
	if reverse == nil {
		return nil
	}
	return func(s *rendererSpec) { s.scrollY = reverse }
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

	if spec.isRevolve {
		// For revolve, scroll uses offscreen+compositeWithWrap with canvasSize tile.
		if spec.scrollX != nil {
			spec.effects = append(spec.effects, scrollXEffect(*spec.scrollX, canvasSize))
		}
		if spec.scrollY != nil {
			spec.effects = append(spec.effects, scrollYEffect(*spec.scrollY, canvasSize))
		}
		effect := composeEffects(spec.effects...)
		return newRevolveRenderer(spec.lines, spec.bgColor, spec.fontColor, spec.colorEffect, effect, f)
	}

	// For text renderer, compute tile sizes from font metrics and create scroll effects.
	if spec.scrollX != nil || spec.scrollY != nil {
		face, ascent, descent, err := findFontAndMetrics(spec.lines, f)
		if err != nil {
			return nil, err
		}
		n := len(spec.lines)
		lineH := ascent + descent

		if spec.scrollX != nil {
			var tileW float64
			mctx := gg.NewContext(canvasSize, canvasSize)
			mctx.SetFontFace(face)
			for _, line := range spec.lines {
				w, _ := mctx.MeasureString(line)
				if w > tileW {
					tileW = w
				}
			}
			spec.effects = append(spec.effects, scrollXEffect(*spec.scrollX, tileW))
		}

		if spec.scrollY != nil {
			firstAscent, _ := glyphBounds(face, spec.lines[0])
			_, lastDescent := glyphBounds(face, spec.lines[n-1])
			tileH := float64(n-1)*lineH + firstAscent + lastDescent
			spec.effects = append(spec.effects, scrollYEffect(*spec.scrollY, tileH))
		}
	}

	effect := composeEffects(spec.effects...)
	return newTextRenderer(spec.lines, spec.bgColor, spec.fontColor, spec.colorEffect, effect, f)
}

// newTextRenderer returns a closure that renders text with effects (rotation, scrolling, etc.).
// The closure captures precomputed font metrics and computes per-frame transforms using the effect.
// Scroll configuration is entirely determined by FrameParams (ScrollX, ScrollY, ScrollTileW, ScrollTileH).
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

	// Scroll configuration is in the effect (FrameParams.ScrollTileW/H).
	// No separate tile size computation here.

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

		ctx.SetFontFace(face)
		ctx.SetColor(fc)
		ctx.Push()

		s := 1.0 + params.SizeScale
		if s < 0.001 {
			s = 0.001
		}

		if params.SizeScale != 0 || params.RotationAngle != 0 {
			ctx.Translate(canvasSize/2, canvasSize/2)
			if params.SizeScale != 0 {
				ctx.Scale(s, s)
			}
			if params.RotationAngle != 0 {
				ctx.Rotate(params.RotationAngle)
			}
			ctx.Translate(-canvasSize/2, -canvasSize/2)
		}

		// Detect scroll from FrameParams
		hasScrollX := params.ScrollTileW > 0
		hasScrollY := params.ScrollTileH > 0

		if hasScrollX || hasScrollY {
			// Draw multiple tiled copies directly on the scaled canvas.
			//
			// The scale transform is already applied to ctx (about canvas center).
			// Scroll offsets are in canvas space; divide by s to get pre-transform coords.
			// Tile intervals in pre-transform space equal ScrollTileW/H (no s factor),
			// because the scale is baked into the canvas transform — the copies therefore
			// appear at intervals of s*ScrollTileW/H in the final image.
			scrollXPre := 0.0
			if hasScrollX {
				scrollXPre = params.ScrollX / s
			}
			scrollYPre := 0.0
			if hasScrollY {
				scrollYPre = params.ScrollY / s
			}

			// Number of copies needed to fill the canvas in each axis.
			// s*ScrollTileW is the visual tile size; ceil(canvas/tile)+2 guarantees coverage.
			kMin, kMax := 0, 0
			if hasScrollX {
				nk := int(math.Ceil(canvasSize/(s*params.ScrollTileW))) + 2
				kMin, kMax = -nk, nk
			}
			lMin, lMax := 0, 0
			if hasScrollY {
				nl := int(math.Ceil(canvasSize/(s*params.ScrollTileH))) + 2
				lMin, lMax = -nl, nl
			}

			for k := kMin; k <= kMax; k++ {
				for l := lMin; l <= lMax; l++ {
					for i, line := range lines {
						x := canvasSize/2.0 + scrollXPre + float64(k)*params.ScrollTileW
						y := baseline0 + scrollYPre + float64(l)*params.ScrollTileH + float64(i)*lineH
						ctx.DrawStringAnchored(line, x, y, 0.5, 0)
					}
				}
			}
		} else {
			for i, line := range lines {
				baseline := baseline0 + float64(i)*lineH
				ctx.DrawStringAnchored(line, canvasSize/2, baseline, 0.5, 0)
			}
		}

		ctx.Pop()
		return ctx.Image(), nil
	}, nil
}
