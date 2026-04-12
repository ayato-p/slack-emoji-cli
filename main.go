package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ayato-p/slack-emoji-cli/config"
	"github.com/ayato-p/slack-emoji-cli/render"
	"github.com/flopp/go-findfont"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "emo [options] TEXT",
	Short: "Generate Slack emoji images from text",
	Long: `Generate Slack emoji images from text.

TEXT: text to render; use \ to separate lines (e.g. '猫に\小判')`,
	Args:         cobra.ExactArgs(1),
	RunE:         runEmo,
	SilenceUsage: true,
}

func init() {
	f := rootCmd.Flags()
	f.StringP("out", "o", "", "output file path (default: emoji.png, or emoji.gif with animation flags)")
	f.String("bg", "#1D3557", `background color (hex e.g. #1D3557, or "transparent")`)
	f.StringP("font-color", "c", "", `font color (hex e.g. #FFFFFF, or "transparent")`)
	f.String("font", "", "font file path or font file name to search in system fonts (e.g. NotoSansJP-Regular.ttf)")
	f.String("rotate", "", "add rotation animation (outputs GIF); use =reverse to reverse direction")
	f.String("revolve", "", "add revolve animation: characters orbit the canvas center (outputs GIF); use =reverse to reverse direction")
	f.String("scroll-x", "", "add horizontal scroll animation (outputs GIF); use =reverse to reverse direction")
	f.String("scroll-y", "", "add vertical scroll animation (outputs GIF); use =reverse to reverse direction")
	f.String("pulsing", "", "add scale pulse animation (outputs GIF); use =reverse to reverse direction")
	f.Float64("speed", 1.0, "animation speed multiplier (0.5–2.0, GIF only)")

	for _, name := range []string{"rotate", "revolve", "scroll-x", "scroll-y", "pulsing"} {
		f.Lookup(name).NoOptDefVal = "true"
	}

	if err := viper.BindPFlags(f); err != nil {
		fmt.Fprintf(os.Stderr, "flag bind error: %v\n", err)
		os.Exit(1)
	}
}

func runEmo(cmd *cobra.Command, args []string) error {
	var cfg config.EmoConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("config error: %w", err)
	}
	cfg.Text = args[0]
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}
	// Resolve font: explicit --font or find default system font
	resolved, err := resolveFont(cfg.Font)
	if err != nil {
		return fmt.Errorf("font error: %w", err)
	}
	cfg.Font = resolved
	return render.Run(cfg)
}

func resolveFont(spec string) (string, error) {
	// Empty spec: try CJK-aware font candidates (Noto, IPA, etc.)
	// Note: Look for TTF/OTF files, not TTC (font collections) since opentype.Parse doesn't support TTC
	if spec == "" {
		// Try candidates in priority order (prefer TTF over TTC)
		candidates := []string{
			"fonts-japanese-gothic.ttf",
			"fonts-japanese-mincho.ttf",
			"ipag.ttf",
			"ipam.ttf",
			"IPAGothic", // Fallback without extension
			"IPAMincho",
			"DejaVuSans.ttf", // Fallback: works on most systems
		}

		for _, candidate := range candidates {
			path, err := findfont.Find(candidate)
			if err == nil {
				// Skip TTC files (font collections) - opentype.Parse doesn't support them
				if strings.HasSuffix(strings.ToLower(path), ".ttc") {
					continue
				}
				return path, nil
			}
		}

		// No suitable default font found: require explicit --font
		return "", fmt.Errorf("no suitable default font found on system; please use --font to specify a font file (e.g. --font /path/to/font.ttf)")
	}

	if strings.ContainsRune(spec, '/') {
		// 明示的なパス（/ を含む）は直接チェック
		if _, err := os.Stat(spec); err != nil {
			return "", fmt.Errorf("cannot open font file %q: %w", spec, err)
		}
		abs, err := filepath.Abs(spec)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	// findfont.Find は既存ファイルもフォント名も両方対応
	return findfont.Find(spec)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
