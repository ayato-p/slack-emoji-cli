package render

import (
	"image"
	"image/color"
	"slices"
)

// buildPalette derives an optimal ≤256-entry color.Palette from a set of
// pre-rendered RGBA images and a mandatory background color that must be
// preserved exactly in the palette.
// Returns (palette, isExact) where isExact is true when all unique colors fit
// in the palette (no quantization needed).
func buildPalette(images []image.Image, bgColor color.Color) (color.Palette, bool) {
	// Convert bgColor to RGBA for comparison
	bgRGBA := color.RGBAModel.Convert(bgColor).(color.RGBA)

	// Collect unique colors and their frequencies from all frames
	colorFreq := make(map[color.RGBA]int)
	for _, img := range images {
		rgba := img.(*image.RGBA)
		for y := rgba.Bounds().Min.Y; y < rgba.Bounds().Max.Y; y++ {
			for x := rgba.Bounds().Min.X; x < rgba.Bounds().Max.X; x++ {
				c := rgba.RGBAAt(x, y)
				colorFreq[c]++
			}
		}
	}

	// If ≤256 unique colors, use them directly (no quantization)
	if len(colorFreq) <= 256 {
		return buildExactPalette(colorFreq, bgRGBA), true
	}

	// Otherwise, use median cut quantization
	return buildQuantizedPalette(colorFreq, bgRGBA), false
}

// buildExactPalette constructs a palette from all unique colors,
// ensuring bgColor is at index 0.
func buildExactPalette(colorFreq map[color.RGBA]int, bgColor color.RGBA) color.Palette {
	colors := make([]color.RGBA, 0, len(colorFreq))
	for c := range colorFreq {
		colors = append(colors, c)
	}

	// Sort for determinism: background first, then by R, G, B
	slices.SortFunc(colors, func(a, b color.RGBA) int {
		// Background color comes first
		if a == bgColor {
			return -1
		}
		if b == bgColor {
			return 1
		}
		// Otherwise sort by R, G, B lexicographically
		if a.R != b.R {
			return int(a.R) - int(b.R)
		}
		if a.G != b.G {
			return int(a.G) - int(b.G)
		}
		return int(a.B) - int(b.B)
	})

	pal := make(color.Palette, len(colors))
	for i, c := range colors {
		pal[i] = c
	}
	return pal
}

// buildQuantizedPalette uses median cut to reduce colors to 255,
// then prepends bgColor as index 0.
func buildQuantizedPalette(colorFreq map[color.RGBA]int, bgColor color.RGBA) color.Palette {
	// Exclude background color from quantization to guarantee it survives
	colorsCopy := make(map[color.RGBA]int)
	for c, freq := range colorFreq {
		if c != bgColor {
			colorsCopy[c] = freq
		}
	}

	// Run median cut to produce 255 representative colors
	representatives := medianCut(colorsCopy, 255)

	// Build final palette: bgColor at index 0, then representatives
	pal := make(color.Palette, 0, 256)
	pal = append(pal, bgColor)
	for _, c := range representatives {
		pal = append(pal, c)
	}
	return pal
}

// colorBox represents a bounding box of colors in RGB space.
type colorBox struct {
	colors map[color.RGBA]int // color -> frequency count
	rMin, rMax uint8
	gMin, gMax uint8
	bMin, bMax uint8
}

// medianCut recursively splits color boxes to produce up to maxBoxes
// representative colors, each as the weighted average of its box.
func medianCut(colorFreq map[color.RGBA]int, maxBoxes int) []color.RGBA {
	if len(colorFreq) == 0 {
		return nil
	}
	if len(colorFreq) <= maxBoxes {
		// All colors fit; return weighted average of the entire set
		colors := make([]color.RGBA, 0, len(colorFreq))
		for c := range colorFreq {
			colors = append(colors, c)
		}
		return colors
	}

	// Initialize with a single box containing all colors
	boxes := []*colorBox{buildBox(colorFreq)}

	// Split boxes until we have maxBoxes
	for len(boxes) < maxBoxes {
		// Find the box with the largest range
		var largest *colorBox
		var largestIdx int
		for i, box := range boxes {
			if largest == nil || boxRange(box) > boxRange(largest) {
				largest = box
				largestIdx = i
			}
		}

		if largest == nil {
			break
		}

		// Split the largest box along its longest axis
		box1, box2 := splitBox(largest)
		if box2 == nil {
			// Cannot split further
			break
		}

		// Replace largest with the two new boxes
		boxes[largestIdx] = box1
		boxes = append(boxes, box2)
	}

	// Extract representative color from each box
	representatives := make([]color.RGBA, len(boxes))
	for i, box := range boxes {
		representatives[i] = boxRepresentative(box)
	}
	return representatives
}

// buildBox creates a colorBox from a color frequency map.
func buildBox(colorFreq map[color.RGBA]int) *colorBox {
	box := &colorBox{
		colors: colorFreq,
		rMin:   255, rMax: 0,
		gMin:   255, gMax: 0,
		bMin:   255, bMax: 0,
	}
	for c := range colorFreq {
		if c.R < box.rMin {
			box.rMin = c.R
		}
		if c.R > box.rMax {
			box.rMax = c.R
		}
		if c.G < box.gMin {
			box.gMin = c.G
		}
		if c.G > box.gMax {
			box.gMax = c.G
		}
		if c.B < box.bMin {
			box.bMin = c.B
		}
		if c.B > box.bMax {
			box.bMax = c.B
		}
	}
	return box
}

// boxRange returns the length of the longest axis in RGB space.
func boxRange(box *colorBox) int {
	rRange := int(box.rMax) - int(box.rMin)
	gRange := int(box.gMax) - int(box.gMin)
	bRange := int(box.bMax) - int(box.bMin)
	if rRange > gRange && rRange > bRange {
		return rRange
	}
	if gRange > bRange {
		return gRange
	}
	return bRange
}

// splitBox splits a box along its longest axis at the median.
func splitBox(box *colorBox) (*colorBox, *colorBox) {
	rRange := int(box.rMax) - int(box.rMin)
	gRange := int(box.gMax) - int(box.gMin)
	bRange := int(box.bMax) - int(box.bMin)

	// Determine split axis and sort colors
	var colors []color.RGBA
	if rRange > gRange && rRange > bRange {
		colors = make([]color.RGBA, 0, len(box.colors))
		for c := range box.colors {
			colors = append(colors, c)
		}
		slices.SortFunc(colors, func(a, b color.RGBA) int {
			return int(a.R) - int(b.R)
		})
	} else if gRange > bRange {
		colors = make([]color.RGBA, 0, len(box.colors))
		for c := range box.colors {
			colors = append(colors, c)
		}
		slices.SortFunc(colors, func(a, b color.RGBA) int {
			return int(a.G) - int(b.G)
		})
	} else {
		colors = make([]color.RGBA, 0, len(box.colors))
		for c := range box.colors {
			colors = append(colors, c)
		}
		slices.SortFunc(colors, func(a, b color.RGBA) int {
			return int(a.B) - int(b.B)
		})
	}

	if len(colors) < 2 {
		return nil, nil
	}

	// Split at the median
	mid := len(colors) / 2
	box1Colors := make(map[color.RGBA]int)
	box2Colors := make(map[color.RGBA]int)
	for i := 0; i < mid; i++ {
		box1Colors[colors[i]] = box.colors[colors[i]]
	}
	for i := mid; i < len(colors); i++ {
		box2Colors[colors[i]] = box.colors[colors[i]]
	}

	return buildBox(box1Colors), buildBox(box2Colors)
}

// boxRepresentative returns the weighted average color of a box.
func boxRepresentative(box *colorBox) color.RGBA {
	var sumR, sumG, sumB, totalFreq int
	for c, freq := range box.colors {
		sumR += int(c.R) * freq
		sumG += int(c.G) * freq
		sumB += int(c.B) * freq
		totalFreq += freq
	}
	if totalFreq == 0 {
		return color.RGBA{0, 0, 0, 255}
	}
	return color.RGBA{
		R: uint8(sumR / totalFreq),
		G: uint8(sumG / totalFreq),
		B: uint8(sumB / totalFreq),
		A: 255,
	}
}
