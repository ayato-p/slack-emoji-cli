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

const (
	numFrames  = 24 // frames per full animation cycle
	frameDelay = 4  // 40ms per frame → ~1 second per revolution
)

// ScrollConfig controls wrap-around scrolling in RenderGIF and RevolveGIF.
// X enables horizontal scrolling; Y enables vertical scrolling.
// Default direction: X scrolls right-to-left, Y scrolls bottom-to-top.
// ReverseX/ReverseY flip the respective scroll direction.
type ScrollConfig struct {
	X, ReverseX bool
	Y, ReverseY bool
}

// compositeWithWrap draws src onto dst with seamless wrap-around scrolling.
// What exits one edge of the canvas re-enters immediately from the opposite edge.
func compositeWithWrap(dst *gg.Context, src image.Image, scroll ScrollConfig, frame, total int) {
	progress := float64(frame) / float64(total)

	var dx, dy float64
	var wxOff, wyOff int

	if scroll.X {
		if scroll.ReverseX {
			dx = progress * canvasSize // left-to-right
			wxOff = -canvasSize        // wrap copy enters from left
		} else {
			dx = -progress * canvasSize // right-to-left
			wxOff = canvasSize          // wrap copy enters from right
		}
	}
	if scroll.Y {
		if scroll.ReverseY {
			dy = progress * canvasSize // top-to-bottom
			wyOff = -canvasSize        // wrap copy enters from top
		} else {
			dy = -progress * canvasSize // bottom-to-top
			wyOff = canvasSize          // wrap copy enters from bottom
		}
	}

	ix, iy := int(math.Round(dx)), int(math.Round(dy))

	// Main copy
	dst.DrawImage(src, ix, iy)
	// Horizontal wrap copy
	if scroll.X {
		dst.DrawImage(src, ix+wxOff, iy)
	}
	// Vertical wrap copy
	if scroll.Y {
		dst.DrawImage(src, ix, iy+wyOff)
	}
	// Corner wrap copy (needed when both X and Y scroll are active)
	if scroll.X && scroll.Y {
		dst.DrawImage(src, ix+wxOff, iy+wyOff)
	}
}

// RenderGIF generates an animated GIF with the given lines of text on a solid
// background color. Each frame has all transformers applied in order, enabling
// composable animations. When scroll is active, wrap-around scrolling is applied
// as a post-compositing step so the content seamlessly re-enters from the
// opposite edge.
func RenderGIF(lines []string, bgColor color.Color, transformers []Transformer, scroll ScrollConfig, speed float64) (*gif.GIF, error) {
	face, ascent, descent, err := findFontAndMetrics(lines)
	if err != nil {
		return nil, err
	}

	n := len(lines)
	lineH := ascent + descent
	firstAscent, _ := glyphBounds(face, lines[0])
	_, lastDescent := glyphBounds(face, lines[n-1])
	baseline0 := (canvasSize - float64(n-1)*lineH + firstAscent - lastDescent) / 2

	actualDelay := int(math.Round(float64(frameDelay) / speed))
	if actualDelay < 1 {
		actualDelay = 1
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

		if scroll.X || scroll.Y {
			// Two-pass rendering: draw content onto a transparent offscreen canvas,
			// then composite it onto the background at wrapped positions so that
			// what exits one edge re-enters immediately from the opposite edge.
			offscreen := gg.NewContext(canvasSize, canvasSize)
			offscreen.SetFontFace(face)
			offscreen.SetColor(color.White)
			offscreen.Push()
			for _, t := range transformers {
				t(offscreen, i, numFrames)
			}
			for j, line := range lines {
				baseline := baseline0 + float64(j)*lineH
				offscreen.DrawStringAnchored(line, canvasSize/2, baseline, 0.5, 0)
			}
			offscreen.Pop()
			compositeWithWrap(ctx, offscreen.Image(), scroll, i, numFrames)
		} else {
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
		}

		// Convert RGBA frame to paletted image for GIF encoding
		img := ctx.Image()
		paletted := image.NewPaletted(img.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(paletted, img.Bounds(), img, image.Point{})

		g.Image = append(g.Image, paletted)
		g.Delay = append(g.Delay, actualDelay)
	}

	return g, nil
}
