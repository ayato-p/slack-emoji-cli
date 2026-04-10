package main

import (
	"fmt"
	"os"

	"github.com/ayato-p/slack-emoji-cli/config"
	"github.com/ayato-p/slack-emoji-cli/render"
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
	f.String("rotate", "", "add rotation animation (outputs GIF); use =reverse to reverse direction")
	f.String("revolve", "", "add revolve animation: characters orbit the canvas center (outputs GIF); use =reverse to reverse direction")
	f.String("scroll-x", "", "add horizontal scroll animation (outputs GIF); use =reverse to reverse direction")
	f.String("scroll-y", "", "add vertical scroll animation (outputs GIF); use =reverse to reverse direction")
	f.Float64("speed", 1.0, "animation speed multiplier (0.5–2.0, GIF only)")

	for _, name := range []string{"rotate", "revolve", "scroll-x", "scroll-y"} {
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
	return render.Run(cfg)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
