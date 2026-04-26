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
func findFontAndMetrics(lines []string, f *opentype.Font, fitWidth bool) (font.Face, float64, float64, error) {
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
			if fitWidth {
				// fit-width: only the shortest line constrains width — wider lines
				// will be compressed to drawArea at draw time.
				minW := math.MaxFloat64
				for _, line := range lines {
					if line == "" {
						continue
					}
					w, _ := ctx.MeasureString(line)
					if w < minW {
						minW = w
					}
				}
				if minW != math.MaxFloat64 && minW > drawArea {
					fits = false
				}
			} else {
				// no-fit-width: all lines must fit within drawArea (original behavior).
				for _, line := range lines {
					if line == "" {
						continue
					}
					w, _ := ctx.MeasureString(line)
					if w > drawArea {
						fits = false
						break
					}
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

// EffectModel is a per-frame DTO that carries all information needed to render a single frame.
// It is produced by the effect function returned from buildEffect and consumed by renderFrame.
type EffectModel struct {
	BGColor     color.Color
	FontColor   color.Color // resolved for this frame (colorEffect already applied)
	FontFace    font.Face
	Params      FrameParams
	DrawContent func(ctx *gg.Context, params FrameParams)
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
	noFitWidth  bool  // disable per-line width equalization
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

func withNoFitWidth(v bool) rendererOption {
	if !v {
		return nil
	}
	return func(s *rendererSpec) { s.noFitWidth = true }
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

// buildEffect applies all options, computes layout, and returns a per-frame effect function.
// The returned function maps (frame, total) → EffectModel, a pure DTO with all render data.
// Callers pass the EffectModel to renderFrame to produce the actual image.
func buildEffect(f *opentype.Font, opts ...rendererOption) (func(frame, total int) EffectModel, error) {
	spec := &rendererSpec{}
	for _, o := range opts {
		if o != nil {
			o(spec)
		}
	}

	if spec.isRevolve {
		return buildRevolveEffect(f, spec)
	}
	return buildTextEffect(f, spec)
}

// buildTextEffect constructs the per-frame effect closure for the text rendering path.
func buildTextEffect(f *opentype.Font, spec *rendererSpec) (func(frame, total int) EffectModel, error) {
	fitWidth := !spec.noFitWidth
	face, ascent, descent, err := findFontAndMetrics(spec.lines, f, fitWidth)
	if err != nil {
		return nil, err
	}

	n := len(spec.lines)
	lineH := ascent + descent

	// Measure each line's natural width to determine per-line x-scaling.
	mctx := gg.NewContext(canvasSize, canvasSize)
	mctx.SetFontFace(face)
	lineWidths := make([]float64, len(spec.lines))
	for i, line := range spec.lines {
		if line == "" {
			continue
		}
		w, _ := mctx.MeasureString(line)
		lineWidths[i] = w
	}
	nonEmptyLines := 0
	for _, lw := range lineWidths {
		if lw > 0 {
			nonEmptyLines++
		}
	}

	// Compute scroll tile sizes from font metrics and append scroll effects.
	if spec.scrollX != nil {
		var tileW float64
		for _, w := range lineWidths {
			if w == 0 {
				continue
			}
			var rendered float64
			if (fitWidth && nonEmptyLines > 1) || w > drawArea {
				rendered = drawArea
			} else {
				rendered = w
			}
			if rendered > tileW {
				tileW = rendered
			}
		}
		if tileW <= 0 {
			tileW = drawArea
		}
		spec.effects = append(spec.effects, scrollXEffect(*spec.scrollX, tileW))
	}
	if spec.scrollY != nil {
		firstAscent, _ := glyphBounds(face, spec.lines[0])
		_, lastDescent := glyphBounds(face, spec.lines[n-1])
		tileH := float64(n-1)*lineH + firstAscent + lastDescent
		spec.effects = append(spec.effects, scrollYEffect(*spec.scrollY, tileH))
	}

	// Compute the y-baseline of the first line (centered vertically).
	firstAscent, _ := glyphBounds(face, spec.lines[0])
	_, lastDescent := glyphBounds(face, spec.lines[n-1])
	baseline0 := (canvasSize - float64(n-1)*lineH + firstAscent - lastDescent) / 2

	effect := composeEffects(spec.effects...)

	lines := spec.lines
	bgColor := spec.bgColor
	fontColor := spec.fontColor
	colorEffect := spec.colorEffect

	return func(frame, total int) EffectModel {
		params := effect(frame, total)
		fc := fontColor
		if colorEffect != nil {
			fc = colorEffect(frame, total)
		}
		return EffectModel{
			BGColor:   bgColor,
			FontColor: fc,
			FontFace:  face,
			Params:    params,
			DrawContent: func(ctx *gg.Context, p FrameParams) {
				ctx.Push()

				s := 1.0 + p.SizeScale
				if s < 0.001 {
					s = 0.001
				}

				if p.SizeScale != 0 || p.RotationAngle != 0 {
					ctx.Translate(canvasSize/2, canvasSize/2)
					if p.SizeScale != 0 {
						ctx.Scale(s, s)
					}
					if p.RotationAngle != 0 {
						ctx.Rotate(p.RotationAngle)
					}
					ctx.Translate(-canvasSize/2, -canvasSize/2)
				}

				hasScrollX := p.ScrollTileW > 0
				hasScrollY := p.ScrollTileH > 0

				if hasScrollX || hasScrollY {
					scrollXPre := 0.0
					if hasScrollX {
						scrollXPre = p.ScrollX / s
					}
					scrollYPre := 0.0
					if hasScrollY {
						scrollYPre = p.ScrollY / s
					}

					kMin, kMax := 0, 0
					if hasScrollX {
						nk := int(math.Ceil(canvasSize/(s*p.ScrollTileW))) + 2
						kMin, kMax = -nk, nk
					}
					lMin, lMax := 0, 0
					if hasScrollY {
						nl := int(math.Ceil(canvasSize/(s*p.ScrollTileH))) + 2
						lMin, lMax = -nl, nl
					}

					for k := kMin; k <= kMax; k++ {
						for l := lMin; l <= lMax; l++ {
							for i, line := range lines {
								x := canvasSize/2.0 + scrollXPre + float64(k)*p.ScrollTileW
								y := baseline0 + scrollYPre + float64(l)*p.ScrollTileH + float64(i)*lineH
								if lw := lineWidths[i]; lw > 0 && ((fitWidth && nonEmptyLines > 1) || lw > drawArea) {
									ctx.Push()
									ctx.Translate(x, y)
									ctx.Scale(drawArea/lw, 1)
									ctx.Translate(-x, -y)
									ctx.DrawStringAnchored(line, x, y, 0.5, 0)
									ctx.Pop()
								} else {
									ctx.DrawStringAnchored(line, x, y, 0.5, 0)
								}
							}
						}
					}
				} else {
					for i, line := range lines {
						baseline := baseline0 + float64(i)*lineH
						if lw := lineWidths[i]; lw > 0 && ((fitWidth && nonEmptyLines > 1) || lw > drawArea) {
							ctx.Push()
							ctx.Translate(canvasSize/2, baseline)
							ctx.Scale(drawArea/lw, 1)
							ctx.Translate(-canvasSize/2, -baseline)
							ctx.DrawStringAnchored(line, canvasSize/2, baseline, 0.5, 0)
							ctx.Pop()
						} else {
							ctx.DrawStringAnchored(line, canvasSize/2, baseline, 0.5, 0)
						}
					}
				}

				ctx.Pop()
			},
		}
	}, nil
}

// buildRevolveEffect constructs the per-frame effect closure for the revolve rendering path.
func buildRevolveEffect(f *opentype.Font, spec *rendererSpec) (func(frame, total int) EffectModel, error) {
	// Flatten all characters from all lines into one slice.
	var chars []string
	for _, line := range spec.lines {
		for _, r := range []rune(line) {
			chars = append(chars, string(r))
		}
	}
	N := len(chars)

	var face font.Face
	var orbitRadius, halfMetricSpan float64

	if N == 0 {
		// No characters: load a face at any size; the renderer will just draw background.
		var err error
		face, err = loadFace(f, 12)
		if err != nil {
			return nil, err
		}
	} else {
		// Find the largest font where every character fits within the orbit spacing limit.
		// This is a different algorithm from findFontAndMetrics: it constrains each glyph
		// to fit within the arc between adjacent orbit positions.
		sinPN := math.Sin(math.Pi / float64(N))
		charMaxLimit := float64(drawArea) * sinPN / (1 + sinPN)

		var ascent, descent float64
		tmpCtx := gg.NewContext(canvasSize, canvasSize)
		for fontSize := 120.0; fontSize >= 6; fontSize-- {
			faceAtSize, err := loadFace(f, fontSize)
			if err != nil {
				return nil, err
			}
			tmpCtx.SetFontFace(faceAtSize)

			var maxW, maxH float64
			var iterAscent, iterDescent float64
			for _, s := range chars {
				w, _ := tmpCtx.MeasureString(s)
				if w > maxW {
					maxW = w
				}
				a, d := glyphBounds(faceAtSize, s)
				h := a + d
				if h > maxH {
					maxH = h
				}
				if a > iterAscent {
					iterAscent = a
				}
				if d > iterDescent {
					iterDescent = d
				}
			}
			face = faceAtSize
			ascent = iterAscent
			descent = iterDescent
			if math.Max(maxW, maxH) <= charMaxLimit {
				break
			}
		}

		// Re-measure at the final selected font size.
		tmpCtx.SetFontFace(face)
		var maxCharW, maxCharH float64
		for _, s := range chars {
			w, _ := tmpCtx.MeasureString(s)
			if w > maxCharW {
				maxCharW = w
			}
			a, d := glyphBounds(face, s)
			h := a + d
			if h > maxCharH {
				maxCharH = h
			}
		}
		charMax := math.Max(maxCharW, maxCharH)
		orbitRadius = float64(drawArea)/2 - charMax/2
		halfMetricSpan = (ascent - descent) / 2
	}

	// For revolve, scroll uses offscreen+compositeWithWrap with canvasSize tile.
	if spec.scrollX != nil {
		spec.effects = append(spec.effects, scrollXEffect(*spec.scrollX, canvasSize))
	}
	if spec.scrollY != nil {
		spec.effects = append(spec.effects, scrollYEffect(*spec.scrollY, canvasSize))
	}
	effect := composeEffects(spec.effects...)

	bgColor := spec.bgColor
	fontColor := spec.fontColor
	colorEffect := spec.colorEffect

	return func(frame, total int) EffectModel {
		params := effect(frame, total)
		fc := fontColor
		if colorEffect != nil {
			fc = colorEffect(frame, total)
		}
		return EffectModel{
			BGColor:   bgColor,
			FontColor: fc,
			FontFace:  face,
			Params:    params,
			DrawContent: func(ctx *gg.Context, p FrameParams) {
				N := len(chars)
				if N == 0 {
					return
				}

				const startAngle = -3 * math.Pi / 4
				cx, cy := float64(canvasSize)/2, float64(canvasSize)/2

				drawChars := func(target *gg.Context) {
					target.SetFontFace(face)
					target.SetColor(fc)
					target.Push()
					if p.SizeScale != 0 {
						s := 1.0 + p.SizeScale
						target.Translate(canvasSize/2, canvasSize/2)
						target.Scale(s, s)
						target.Translate(-canvasSize/2, -canvasSize/2)
					}
					for k, ch := range chars {
						theta := startAngle - float64(k)*(2*math.Pi/float64(N)) + p.RevolveOffset
						x := cx + orbitRadius*math.Cos(theta)
						yOrbit := cy + orbitRadius*math.Sin(theta)
						baseline := yOrbit + halfMetricSpan
						target.DrawStringAnchored(ch, x, baseline, 0.5, 0)
					}
					target.Pop()
				}

				if p.ScrollX != 0 || p.ScrollY != 0 {
					offscreen := gg.NewContext(canvasSize, canvasSize)
					drawChars(offscreen)
					compositeWithWrap(ctx, offscreen.Image(), p.ScrollX, p.ScrollY, canvasSize, canvasSize)
				} else {
					drawChars(ctx)
				}
			},
		}
	}, nil
}

// renderFrame renders a single frame from the given EffectModel.
func renderFrame(m EffectModel) image.Image {
	ctx := gg.NewContext(canvasSize, canvasSize)
	r, g, b, a := m.BGColor.RGBA()
	ctx.SetRGBA(float64(r)/0xffff, float64(g)/0xffff, float64(b)/0xffff, float64(a)/0xffff)
	ctx.Clear()
	ctx.SetFontFace(m.FontFace)
	ctx.SetColor(m.FontColor)
	m.DrawContent(ctx, m.Params)
	return ctx.Image()
}
