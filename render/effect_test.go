package render

import (
	"image"
	"image/color"
	"math"
	"testing"

	findfont "github.com/flopp/go-findfont"
	"golang.org/x/image/font/opentype"
)

// hasNonBgPixels reports whether img contains any pixel that differs from bg.
func hasNonBgPixels(img image.Image, bg color.Color) bool {
	bgR, bgG, bgB, _ := bg.RGBA()
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r != bgR || g != bgG || b != bgB {
				return true
			}
		}
	}
	return false
}

// imagesEqual reports whether two images have identical pixel values.
func imagesEqual(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	bounds := a.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ar, ag, ab, aa := a.At(x, y).RGBA()
			br, bg2, bb, ba := b.At(x, y).RGBA()
			if ar != br || ag != bg2 || ab != bb || aa != ba {
				return false
			}
		}
	}
	return true
}

// testFont loads a system font for use in tests. The test is skipped if no
// suitable font is found on the current machine.
func testFont(t *testing.T) *opentype.Font {
	t.Helper()
	candidates := []string{
		"DejaVuSans.ttf",
		"fonts-japanese-gothic.ttf",
		"ipag.ttf",
	}
	for _, name := range candidates {
		path, err := findfont.Find(name)
		if err != nil {
			continue
		}
		f, err := loadFont(path)
		if err != nil {
			continue
		}
		return f
	}
	t.Skip("no suitable test font found on this machine")
	return nil
}

// boolPtr returns a pointer to b, used to construct *bool animation flags.
func boolPtr(b bool) *bool { return &b }

// approxEq reports whether a and b are within eps of each other.
func approxEq(a, b, eps float64) bool { return math.Abs(a-b) < eps }

const eps = 1e-9

// TestBuildEffect runs table-driven tests that call buildEffect with various
// option combinations, execute the returned effect function, and assert the
// resulting EffectModel fields.
func TestBuildEffect(t *testing.T) {
	f := testFont(t)

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	tests := []struct {
		name  string
		opts  []rendererOption
		frame int
		total int
		check func(t *testing.T, m EffectModel)
	}{
		// ── Group A: mode and content ──────────────────────────────────────────
		{
			name:  "text mode: renders text onto background",
			opts:  []rendererOption{withLines([]string{"hello"}), withBg(white), withFontColor(black)},
			frame: 0, total: 1,
			check: func(t *testing.T, m EffectModel) {
				img := renderFrame(m)
				want := image.Rect(0, 0, canvasSize, canvasSize)
				if img.Bounds() != want {
					t.Errorf("image bounds: want %v, got %v", want, img.Bounds())
				}
				if !hasNonBgPixels(img, white) {
					t.Error("rendered image is all background; expected text to be drawn")
				}
			},
		},
		{
			name:  "revolve mode: renders characters in orbit, animation advances per frame",
			opts:  []rendererOption{withLines([]string{"AB"}), withBg(white), withFontColor(black), withRevolve(boolPtr(false))},
			frame: 0, total: 8,
			check: func(t *testing.T, m EffectModel) {
				img0 := renderFrame(m)
				if !hasNonBgPixels(img0, white) {
					t.Error("frame 0 is all background; expected characters to be drawn")
				}
				// Quarter orbit: characters should have moved to different positions.
				effect2, err := buildEffect(f,
					withLines([]string{"AB"}), withBg(white), withFontColor(black), withRevolve(boolPtr(false)),
				)
				if err != nil {
					t.Fatalf("buildEffect error: %v", err)
				}
				img2 := renderFrame(effect2(2, 8))
				if imagesEqual(img0, img2) {
					t.Error("frame 0 and frame 2 should differ: revolve animation should advance characters")
				}
			},
		},

		// ── Group B: per-frame parameters (single effects) ────────────────────
		{
			name:  "rotate forward frame=0: RotationAngle == 0",
			opts:  []rendererOption{withLines([]string{"A"}), withBg(white), withFontColor(black), withRotate(boolPtr(false))},
			frame: 0, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if !approxEq(m.Params.RotationAngle, 0, eps) {
					t.Errorf("RotationAngle: want 0, got %f", m.Params.RotationAngle)
				}
			},
		},
		{
			name:  "rotate forward frame=1/4: RotationAngle ≈ π/2",
			opts:  []rendererOption{withLines([]string{"A"}), withBg(white), withFontColor(black), withRotate(boolPtr(false))},
			frame: 1, total: 4,
			check: func(t *testing.T, m EffectModel) {
				want := math.Pi / 2
				if !approxEq(m.Params.RotationAngle, want, eps) {
					t.Errorf("RotationAngle: want %f, got %f", want, m.Params.RotationAngle)
				}
			},
		},
		{
			name:  "rotate reverse frame=1/4: RotationAngle ≈ -π/2",
			opts:  []rendererOption{withLines([]string{"A"}), withBg(white), withFontColor(black), withRotate(boolPtr(true))},
			frame: 1, total: 4,
			check: func(t *testing.T, m EffectModel) {
				want := -math.Pi / 2
				if !approxEq(m.Params.RotationAngle, want, eps) {
					t.Errorf("RotationAngle: want %f, got %f", want, m.Params.RotationAngle)
				}
			},
		},
		{
			name:  "scrollX forward frame=0: ScrollTileW > 0, ScrollX == 0",
			opts:  []rendererOption{withLines([]string{"A"}), withBg(white), withFontColor(black), withScrollX(boolPtr(false))},
			frame: 0, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if m.Params.ScrollTileW <= 0 {
					t.Errorf("ScrollTileW must be > 0, got %f", m.Params.ScrollTileW)
				}
				if !approxEq(m.Params.ScrollX, 0, eps) {
					t.Errorf("ScrollX: want 0 at frame=0, got %f", m.Params.ScrollX)
				}
			},
		},
		{
			name:  "scrollX forward frame=1/4: ScrollX < 0 (right-to-left)",
			opts:  []rendererOption{withLines([]string{"A"}), withBg(white), withFontColor(black), withScrollX(boolPtr(false))},
			frame: 1, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if m.Params.ScrollX >= 0 {
					t.Errorf("ScrollX must be < 0 for forward scroll at frame=1, got %f", m.Params.ScrollX)
				}
			},
		},
		{
			name:  "scrollX reverse frame=1/4: ScrollX > 0 (left-to-right)",
			opts:  []rendererOption{withLines([]string{"A"}), withBg(white), withFontColor(black), withScrollX(boolPtr(true))},
			frame: 1, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if m.Params.ScrollX <= 0 {
					t.Errorf("ScrollX must be > 0 for reverse scroll at frame=1, got %f", m.Params.ScrollX)
				}
			},
		},
		{
			name:  "scrollY forward frame=1/4: ScrollY < 0 (bottom-to-top)",
			opts:  []rendererOption{withLines([]string{"A"}), withBg(white), withFontColor(black), withScrollY(boolPtr(false))},
			frame: 1, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if m.Params.ScrollY >= 0 {
					t.Errorf("ScrollY must be < 0 for forward scroll at frame=1, got %f", m.Params.ScrollY)
				}
				if m.Params.ScrollTileH <= 0 {
					t.Errorf("ScrollTileH must be > 0, got %f", m.Params.ScrollTileH)
				}
			},
		},
		{
			name:  "revolve forward frame=1/4: RevolveOffset ≈ π/2",
			opts:  []rendererOption{withLines([]string{"AB"}), withBg(white), withFontColor(black), withRevolve(boolPtr(false))},
			frame: 1, total: 4,
			check: func(t *testing.T, m EffectModel) {
				want := math.Pi / 2
				if !approxEq(m.Params.RevolveOffset, want, eps) {
					t.Errorf("RevolveOffset: want %f, got %f", want, m.Params.RevolveOffset)
				}
			},
		},
		{
			name:  "pulse frame=0: SizeScale == 0",
			opts:  []rendererOption{withLines([]string{"A"}), withBg(white), withFontColor(black), withPulse(boolPtr(false))},
			frame: 0, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if !approxEq(m.Params.SizeScale, 0, eps) {
					t.Errorf("SizeScale: want 0 at frame=0, got %f", m.Params.SizeScale)
				}
			},
		},
		{
			name:  "pulse frame=total/4: SizeScale > 0 (expansion phase)",
			opts:  []rendererOption{withLines([]string{"A"}), withBg(white), withFontColor(black), withPulse(boolPtr(false))},
			frame: 1, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if m.Params.SizeScale <= 0 {
					t.Errorf("SizeScale must be > 0 in expansion phase, got %f", m.Params.SizeScale)
				}
			},
		},

		// ── Group C: composed effects ──────────────────────────────────────────
		{
			name: "rotate + scrollX: both params non-zero at frame=1",
			opts: []rendererOption{
				withLines([]string{"A"}), withBg(white), withFontColor(black),
				withRotate(boolPtr(false)), withScrollX(boolPtr(false)),
			},
			frame: 1, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if approxEq(m.Params.RotationAngle, 0, eps) {
					t.Errorf("RotationAngle should be non-zero in composed effect, got %f", m.Params.RotationAngle)
				}
				if approxEq(m.Params.ScrollX, 0, eps) {
					t.Errorf("ScrollX should be non-zero in composed effect, got %f", m.Params.ScrollX)
				}
			},
		},
		{
			name: "scrollX + scrollY: both offsets non-zero at frame=1",
			opts: []rendererOption{
				withLines([]string{"A"}), withBg(white), withFontColor(black),
				withScrollX(boolPtr(false)), withScrollY(boolPtr(false)),
			},
			frame: 1, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if approxEq(m.Params.ScrollX, 0, eps) {
					t.Errorf("ScrollX should be non-zero, got %f", m.Params.ScrollX)
				}
				if approxEq(m.Params.ScrollY, 0, eps) {
					t.Errorf("ScrollY should be non-zero, got %f", m.Params.ScrollY)
				}
			},
		},
		{
			name: "revolve + scrollX: ScrollX shifts rendered output vs revolve-only",
			opts: []rendererOption{
				withLines([]string{"AB"}), withBg(white), withFontColor(black),
				withRevolve(boolPtr(false)), withScrollX(boolPtr(false)),
			},
			frame: 1, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if approxEq(m.Params.ScrollX, 0, eps) {
					t.Errorf("ScrollX should be non-zero in revolve+scrollX, got %f", m.Params.ScrollX)
				}
				// Scroll should shift the rendered output relative to revolve-only at the same frame.
				imgScrolled := renderFrame(m)
				effectNoScroll, err := buildEffect(f,
					withLines([]string{"AB"}), withBg(white), withFontColor(black), withRevolve(boolPtr(false)),
				)
				if err != nil {
					t.Fatalf("buildEffect error: %v", err)
				}
				imgNoScroll := renderFrame(effectNoScroll(1, 4))
				if imagesEqual(imgScrolled, imgNoScroll) {
					t.Error("scrolled and non-scrolled revolve frames should differ at frame=1")
				}
			},
		},

		// ── Group D: color resolution ──────────────────────────────────────────
		{
			name:  "static font color: FontColor matches specified color",
			opts:  []rendererOption{withLines([]string{"A"}), withBg(white), withFontColor(red)},
			frame: 0, total: 1,
			check: func(t *testing.T, m EffectModel) {
				r1, g1, b1, a1 := m.FontColor.RGBA()
				r2, g2, b2, a2 := red.RGBA()
				if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
					t.Errorf("FontColor: want %v, got %v", red, m.FontColor)
				}
			},
		},
		{
			name:  "gaming frame=0 vs frame=6: FontColor differs (hue rotates)",
			opts:  []rendererOption{withLines([]string{"A"}), withBg(white), withGaming()},
			frame: 0, total: 12,
			check: func(t *testing.T, m EffectModel) {
				// Compute frame=6 separately to compare
				f2 := testFont(t)
				effect2, err := buildEffect(f2,
					withLines([]string{"A"}), withBg(white), withGaming(),
				)
				if err != nil {
					t.Fatalf("buildEffect error: %v", err)
				}
				m6 := effect2(6, 12)

				r0, g0, b0, _ := m.FontColor.RGBA()
				r6, g6, b6, _ := m6.FontColor.RGBA()
				if r0 == r6 && g0 == g6 && b0 == b6 {
					t.Errorf("gaming FontColor should differ between frame=0 and frame=6; both got r=%d g=%d b=%d", r0, g0, b0)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			effect, err := buildEffect(f, tc.opts...)
			if err != nil {
				t.Fatalf("buildEffect returned error: %v", err)
			}
			if effect == nil {
				t.Fatal("buildEffect returned nil effect function")
			}
			m := effect(tc.frame, tc.total)
			tc.check(t, m)
		})
	}
}
