package main

import (
	"flag"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"strings"

	"github.com/ayato-p/slack-emoji-cli/render"
)

func main() {
	var output string
	var bgHex string

	flag.StringVar(&output, "o", "emoji.png", "output file path")
	flag.StringVar(&bgHex, "bg", "#1D3557", "background color (hex, e.g. #1D3557)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: emo [options] TEXT")
		fmt.Fprintln(os.Stderr, "  TEXT: text to render; use \\ to separate lines (e.g. '猫に\\小判')")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	text := flag.Arg(0)
	lines := strings.Split(text, `\`)

	bgColor, err := parseHexColor(bgHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid bg color %q: %v\n", bgHex, err)
		os.Exit(1)
	}

	img, err := render.Render(lines, bgColor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "render error: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot create %s: %v\n", output, err)
		os.Exit(1)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		fmt.Fprintf(os.Stderr, "png encode error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "saved: %s\n", output)
}

// parseHexColor parses a CSS-style hex color string (#RGB, #RRGGBB, #RRGGBBAA).
func parseHexColor(s string) (color.RGBA, error) {
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
