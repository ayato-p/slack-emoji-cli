// imgcmp compares two image files (PNG or GIF) at the pixel level and reports
// whether they are visually identical within a configurable tolerance.
//
// Usage:
//
//	imgcmp [-threshold N] file1 file2
//
// Exit codes:
//
//	0  visually same (mean absolute diff per channel <= threshold)
//	1  visually different
//	2  usage error or file I/O failure
package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	_ "image/png"
	"math"
	"os"
)

func main() {
	threshold := flag.Float64("threshold", 1.0, "max allowed mean absolute difference per channel (0–255)")
	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: imgcmp [-threshold N] file1 file2")
		os.Exit(2)
	}

	frames1, err := loadFrames(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading %s: %v\n", flag.Arg(0), err)
		os.Exit(2)
	}
	frames2, err := loadFrames(flag.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading %s: %v\n", flag.Arg(1), err)
		os.Exit(2)
	}

	if len(frames1) != len(frames2) {
		fmt.Printf("frames=%d vs %d\nDIFF (frame count mismatch)\n", len(frames1), len(frames2))
		os.Exit(1)
	}

	b1 := frames1[0].Bounds()
	b2 := frames2[0].Bounds()
	if b1 != b2 {
		fmt.Printf("size=%dx%d vs %dx%d\nDIFF (size mismatch)\n", b1.Dx(), b1.Dy(), b2.Dx(), b2.Dy())
		os.Exit(1)
	}

	meanDiff, maxDiff := compareFrames(frames1, frames2)

	fmt.Printf("frames=%d size=%dx%d mean_diff=%.2f max_diff=%d\n",
		len(frames1), b1.Dx(), b1.Dy(), meanDiff, maxDiff)

	if meanDiff <= *threshold {
		fmt.Printf("SAME (mean %.2f <= threshold %.2f)\n", meanDiff, *threshold)
		os.Exit(0)
	}
	fmt.Printf("DIFF (mean %.2f > threshold %.2f)\n", meanDiff, *threshold)
	os.Exit(1)
}

// loadFrames decodes a PNG or GIF file into a slice of RGBA images.
// PNG produces one frame; GIF produces one frame per animation frame,
// each composited onto a full-canvas RGBA image.
func loadFrames(path string) ([]*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Peek at format without consuming the reader.
	// gif.DecodeAll and image.Decode both accept an io.Reader.
	// Re-open to avoid seek issues.
	f.Close()

	f2, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f2.Close()

	// Try GIF first by peeking the magic bytes, then fall back to image.Decode.
	buf := make([]byte, 6)
	n, _ := f2.Read(buf)
	f2.Close()

	if n >= 6 && (string(buf[:6]) == "GIF89a" || string(buf[:6]) == "GIF87a") {
		return loadGIFFrames(path)
	}
	return loadSingleFrame(path)
}

func loadSingleFrame(path string) ([]*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return []*image.RGBA{toRGBA(img, img.Bounds())}, nil
}

func loadGIFFrames(path string) ([]*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		return nil, err
	}

	canvas := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	frames := make([]*image.RGBA, len(g.Image))
	for i, frame := range g.Image {
		rgba := image.NewRGBA(canvas)
		draw.Draw(rgba, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
		frames[i] = rgba
	}
	return frames, nil
}

func toRGBA(src image.Image, bounds image.Rectangle) *image.RGBA {
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}

// compareFrames computes the mean and max absolute per-channel (RGBA) difference
// across all pixels in all frames.
func compareFrames(a, b []*image.RGBA) (mean float64, maxDiff int) {
	var totalDiff int64
	var totalPixels int64
	maxDiff = 0

	for i := range a {
		fa, fb := a[i], b[i]
		bounds := fa.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				ar, ag, ab, aa := fa.At(x, y).RGBA()
				br, bg, bb, ba := fb.At(x, y).RGBA()
				// RGBA() returns 16-bit values; shift to 8-bit.
				diffs := [4]int{
					absDiff(int(ar>>8), int(br>>8)),
					absDiff(int(ag>>8), int(bg>>8)),
					absDiff(int(ab>>8), int(bb>>8)),
					absDiff(int(aa>>8), int(ba>>8)),
				}
				for _, d := range diffs {
					totalDiff += int64(d)
					if d > maxDiff {
						maxDiff = d
					}
				}
				totalPixels += 4
			}
		}
	}

	if totalPixels == 0 {
		return 0, 0
	}
	mean = math.Round(float64(totalDiff)/float64(totalPixels)*100) / 100
	return mean, maxDiff
}

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}
