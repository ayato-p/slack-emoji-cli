package render

import (
	"math"

	"github.com/fogleman/gg"
)

// Transformer applies a per-frame transformation to a drawing context.
// frame is the current frame index (0-based), totalFrames is the total number of frames.
type Transformer func(ctx *gg.Context, frame, totalFrames int)

// Rotate returns a Transformer that rotates the context around the canvas center
// by one full revolution over the course of all frames.
// When reverse is true, the rotation direction is reversed.
func Rotate(reverse bool) Transformer {
	return func(ctx *gg.Context, frame, total int) {
		angle := float64(frame) / float64(total) * 2 * math.Pi
		if reverse {
			angle = -angle
		}
		ctx.RotateAbout(angle, canvasSize/2, canvasSize/2)
	}
}

