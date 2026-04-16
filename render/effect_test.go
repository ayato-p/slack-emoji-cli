package render

import (
	"image/color"
	"math"
	"testing"

	findfont "github.com/flopp/go-findfont"
	"golang.org/x/image/font/opentype"
)

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
			name:  "text mode: IsRevolve false, Lines set",
			opts:  []rendererOption{withLines([]string{"hello"}), withBg(white), withFontColor(black)},
			frame: 0, total: 1,
			check: func(t *testing.T, m EffectModel) {
				if m.IsRevolve {
					t.Error("expected IsRevolve=false for text mode")
				}
				if len(m.Lines) != 1 || m.Lines[0] != "hello" {
					t.Errorf("expected Lines=[hello], got %v", m.Lines)
				}
				if m.FontFace == nil {
					t.Error("FontFace must not be nil")
				}
			},
		},
		{
			name:  "revolve mode: IsRevolve true, Chars split per rune",
			opts:  []rendererOption{withLines([]string{"AB"}), withBg(white), withFontColor(black), withRevolve(boolPtr(false))},
			frame: 0, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if !m.IsRevolve {
					t.Error("expected IsRevolve=true for revolve mode")
				}
				want := []string{"A", "B"}
				if len(m.Chars) != len(want) {
					t.Fatalf("expected Chars=%v, got %v", want, m.Chars)
				}
				for i, ch := range want {
					if m.Chars[i] != ch {
						t.Errorf("Chars[%d]: want %q, got %q", i, ch, m.Chars[i])
					}
				}
				if m.OrbitRadius <= 0 {
					t.Errorf("OrbitRadius must be > 0, got %f", m.OrbitRadius)
				}
				if m.FontFace == nil {
					t.Error("FontFace must not be nil")
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
			name: "revolve + scrollX: IsRevolve true and ScrollX non-zero at frame=1",
			opts: []rendererOption{
				withLines([]string{"AB"}), withBg(white), withFontColor(black),
				withRevolve(boolPtr(false)), withScrollX(boolPtr(false)),
			},
			frame: 1, total: 4,
			check: func(t *testing.T, m EffectModel) {
				if !m.IsRevolve {
					t.Error("expected IsRevolve=true")
				}
				if approxEq(m.Params.ScrollX, 0, eps) {
					t.Errorf("ScrollX should be non-zero in revolve+scrollX, got %f", m.Params.ScrollX)
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
