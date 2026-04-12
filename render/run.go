package render

import (
	"fmt"
	"image/color"
	"image/gif"
	"image/png"
	"math"
	"os"
	"strings"

	"github.com/ayato-p/slack-emoji-cli/config"
)

// Run はEmoConfigに基づいて絵文字を生成・保存します。
func Run(cfg config.EmoConfig) error {
	lines := strings.Split(cfg.Text, `\`)

	bgColor, err := parseHexColor(cfg.Bg)
	if err != nil {
		return fmt.Errorf("invalid bg color %q: %w", cfg.Bg, err)
	}

	// Load the font (path is already resolved in main.go)
	font, err := loadFont(cfg.Font)
	if err != nil {
		return fmt.Errorf("font error: %w", err)
	}

	f, err := os.Create(cfg.Out)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", cfg.Out, err)
	}
	defer f.Close()

	// Compute frame timing based on speed
	actualDelay := int(math.Round(float64(frameDelay) / cfg.Speed))
	if actualDelay < 1 {
		actualDelay = 1
	}

	// Parse animation flags
	scrollX := animToFlags(cfg.ScrollX)
	scrollY := animToFlags(cfg.ScrollY)
	revolve := animToFlags(cfg.Revolve)
	rotate := animToFlags(cfg.Rotate)

	// Build the per-frame renderer by composing effect options
	opts := []rendererOption{
		withLines(lines), withBg(bgColor),
		withScrollX(scrollX), withScrollY(scrollY),
		withRevolve(revolve), withRotate(rotate),
	}

	renderFn, err := buildRenderer(font, opts...)
	if err != nil {
		return fmt.Errorf("render error: %w", err)
	}

	// Check if animation is needed
	isAnimated := scrollX != nil || scrollY != nil || revolve != nil || rotate != nil
	if isAnimated {
		anim, err := composeGIF(renderFn, numFrames, actualDelay, bgColor)
		if err != nil {
			return fmt.Errorf("render error: %w", err)
		}
		if err := gif.EncodeAll(f, anim); err != nil {
			return fmt.Errorf("gif encode error: %w", err)
		}
	} else {
		img, err := renderFn(0, 1)
		if err != nil {
			return fmt.Errorf("render error: %w", err)
		}
		if err := png.Encode(f, img); err != nil {
			return fmt.Errorf("png encode error: %w", err)
		}
	}

	fmt.Fprintf(os.Stderr, "saved: %s\n", cfg.Out)
	return nil
}

// animToFlags はEmoConfigのアニメーション文字列値を*boolに変換します。
// nil は未設定、&false は正方向、&true は逆方向を示します。
func animToFlags(val string) *bool {
	switch val {
	case "true":
		b := false
		return &b
	case "reverse":
		b := true
		return &b
	}
	return nil
}

// parseHexColor はCSS形式の16進カラー文字列 (#RGB, #RRGGBB, #RRGGBBAA) を解析します。
func parseHexColor(s string) (color.RGBA, error) {
	if s == "transparent" {
		return color.RGBA{0, 0, 0, 0}, nil
	}
	s = strings.TrimPrefix(s, "#")
	var r, g, b, a uint8
	a = 0xff
	switch len(s) {
	case 3:
		_, err := fmt.Sscanf(s, "%1x%1x%1x", &r, &g, &b)
		if err != nil {
			return color.RGBA{}, err
		}
		r, g, b = r*17, g*17, b*17
	case 6:
		_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
		if err != nil {
			return color.RGBA{}, err
		}
	case 8:
		_, err := fmt.Sscanf(s, "%02x%02x%02x%02x", &r, &g, &b, &a)
		if err != nil {
			return color.RGBA{}, err
		}
	default:
		return color.RGBA{}, fmt.Errorf("invalid hex color length")
	}
	return color.RGBA{R: r, G: g, B: b, A: a}, nil
}
