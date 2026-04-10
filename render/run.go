package render

import (
	"fmt"
	"image/color"
	"image/gif"
	"image/png"
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

	f, err := os.Create(cfg.Out)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", cfg.Out, err)
	}
	defer f.Close()

	scrollXSet, scrollXReverse := animToFlags(cfg.ScrollX)
	scrollYSet, scrollYReverse := animToFlags(cfg.ScrollY)
	scroll := ScrollConfig{
		X: scrollXSet, ReverseX: scrollXReverse,
		Y: scrollYSet, ReverseY: scrollYReverse,
	}

	revolveSet, revolveReverse := animToFlags(cfg.Revolve)
	rotateSet, rotateReverse := animToFlags(cfg.Rotate)

	if revolveSet {
		anim, err := RevolveGIF(lines, bgColor, revolveReverse, cfg.Speed, scroll)
		if err != nil {
			return fmt.Errorf("render error: %w", err)
		}
		if err := gif.EncodeAll(f, anim); err != nil {
			return fmt.Errorf("gif encode error: %w", err)
		}
	} else {
		var transformers []Transformer
		if rotateSet {
			transformers = append(transformers, Rotate(rotateReverse))
		}

		if len(transformers) > 0 || scroll.X || scroll.Y {
			anim, err := RenderGIF(lines, bgColor, transformers, scroll, cfg.Speed)
			if err != nil {
				return fmt.Errorf("render error: %w", err)
			}
			if err := gif.EncodeAll(f, anim); err != nil {
				return fmt.Errorf("gif encode error: %w", err)
			}
		} else {
			img, err := Render(lines, bgColor)
			if err != nil {
				return fmt.Errorf("render error: %w", err)
			}
			if err := png.Encode(f, img); err != nil {
				return fmt.Errorf("png encode error: %w", err)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "saved: %s\n", cfg.Out)
	return nil
}

// animToFlags はEmoConfigのアニメーション文字列値をset/reverseのペアに変換します。
func animToFlags(val string) (set, reverse bool) {
	switch val {
	case "true":
		return true, false
	case "reverse":
		return true, true
	}
	return false, false
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
