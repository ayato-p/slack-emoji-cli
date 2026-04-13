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
// tileW and tileH define the tile size for wrapping (0 disables tiling for that axis).
// Any content that exits one edge re-enters immediately from the opposite edge.
func compositeWithWrap(dst *gg.Context, src image.Image, dx, dy float64, tileW, tileH float64) {
	ix := int(math.Round(dx))
	iy := int(math.Round(dy))

	// Calculate X-axis tile positions
	var xOffsets []int
	if tileW > 0 {
		// Compute range of tile indices needed to cover the canvas [0, canvasSize)
		// A tile at position p covers [p, p+canvasSize)
		// We need tiles where p < canvasSize and p+canvasSize > 0
		// Which gives: -canvasSize < p < canvasSize
		// Position p = ix + k*tileW, so:
		// -canvasSize < ix + k*tileW < canvasSize
		// (-canvasSize - ix) / tileW < k < (canvasSize - ix) / tileW
		kMin := int(math.Ceil(float64(-canvasSize-ix) / tileW))
		kMax := int(math.Floor(float64(canvasSize-ix) / tileW))
		for k := kMin; k <= kMax; k++ {
			xOffsets = append(xOffsets, ix+int(math.Round(float64(k)*tileW)))
		}
	} else {
		xOffsets = []int{ix}
	}

	// Calculate Y-axis tile positions
	var yOffsets []int
	if tileH > 0 {
		// Same logic as X-axis
		lMin := int(math.Ceil(float64(-canvasSize-iy) / tileH))
		lMax := int(math.Floor(float64(canvasSize-iy) / tileH))
		for l := lMin; l <= lMax; l++ {
			yOffsets = append(yOffsets, iy+int(math.Round(float64(l)*tileH)))
		}
	} else {
		yOffsets = []int{iy}
	}

	// Draw all tile combinations
	for _, x := range xOffsets {
		for _, y := range yOffsets {
			dst.DrawImage(src, x, y)
		}
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
