package render

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"math"

	"github.com/fogleman/gg"
)

const (
	numFrames  = 24 // frames per full animation cycle
	frameDelay = 4  // 40ms per frame → ~1 second per revolution
)

// compositeWithWrap draws src onto dst with seamless wrap-around scrolling.
// dx and dy are pre-computed pixel offsets (from FrameParams.ScrollX / ScrollY).
// Any content that exits one edge re-enters immediately from the opposite edge.
func compositeWithWrap(dst *gg.Context, src image.Image, dx, dy float64) {
	ix := int(math.Round(dx))
	iy := int(math.Round(dy))

	// Determine the wrap-copy offsets: the copy appears on the side opposite
	// to where the main image moved.
	hasX := ix != 0
	hasY := iy != 0
	wxOff := 0
	wyOff := 0
	if hasX {
		if ix < 0 {
			wxOff = canvasSize // content moved left → wrap copy enters from right
		} else {
			wxOff = -canvasSize // content moved right → wrap copy enters from left
		}
	}
	if hasY {
		if iy < 0 {
			wyOff = canvasSize
		} else {
			wyOff = -canvasSize
		}
	}

	dst.DrawImage(src, ix, iy)
	if hasX {
		dst.DrawImage(src, ix+wxOff, iy)
	}
	if hasY {
		dst.DrawImage(src, ix, iy+wyOff)
	}
	if hasX && hasY {
		dst.DrawImage(src, ix+wxOff, iy+wyOff)
	}
}

// composeGIF calls a per-frame render function for each frame and assembles the
// results into an animated GIF with the given frame delay.
// It builds an optimized color palette from all frames and applies it consistently
// across the animation, ensuring smooth color mapping without unnecessary dithering.
func composeGIF(
	renderFn func(frame, total int) (image.Image, error),
	numFrames int,
	delay int,
	bgColor color.Color,
) (*gif.GIF, error) {
	// Phase 1: render all frames
	frames := make([]image.Image, numFrames)
	for i := 0; i < numFrames; i++ {
		img, err := renderFn(i, numFrames)
		if err != nil {
			return nil, err
		}
		frames[i] = img
	}

	// Phase 2: build optimized color palette from all frames
	pal, isExact := buildPalette(frames, bgColor)

	// Phase 3: convert each frame to paletted image using the global palette
	g := &gif.GIF{LoopCount: 0}
	for _, img := range frames {
		paletted := image.NewPaletted(img.Bounds(), pal)

		// If palette contains all exact colors, use direct mapping without dithering.
		// Otherwise, use Floyd-Steinberg dithering to approximate colors.
		if isExact {
			draw.Draw(paletted, img.Bounds(), img, image.Point{}, draw.Src)
		} else {
			draw.FloydSteinberg.Draw(paletted, img.Bounds(), img, image.Point{})
		}

		g.Image = append(g.Image, paletted)
		g.Delay = append(g.Delay, delay)
		g.Disposal = append(g.Disposal, gif.DisposalBackground)
	}

	return g, nil
}
