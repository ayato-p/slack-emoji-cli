package config

import "fmt"

// EmoConfig はすべての設定を保持する中間表現です。
// CLIフラグからもJSONファイルからも同じ構造体に変換できます。
type EmoConfig struct {
	Text       string  `json:"text"               mapstructure:"text"`
	ScrollX    string  `json:"scroll-x,omitempty" mapstructure:"scroll-x"`
	ScrollY    string  `json:"scroll-y,omitempty" mapstructure:"scroll-y"`
	Revolve    string  `json:"revolve,omitempty"  mapstructure:"revolve"`
	Rotate     string  `json:"rotate,omitempty"   mapstructure:"rotate"`
	Pulsing    string  `json:"pulsing,omitempty"  mapstructure:"pulsing"`
	Speed      float64 `json:"speed,omitempty"    mapstructure:"speed"`
	Bg         string  `json:"bg,omitempty"       mapstructure:"bg"`
	FontColor  string  `json:"font-color,omitempty" mapstructure:"font-color"`
	Out        string  `json:"out,omitempty"      mapstructure:"out"`
	Font       string  `json:"font,omitempty"     mapstructure:"font"`
	NoFitWidth bool    `json:"no-fit-width,omitempty" mapstructure:"no-fit-width"`
}

// SetDefaults はゼロ値フィールドにデフォルト値を設定します。
// CLIパスではflag側のデフォルト値が使われますが、JSON設定パス向けのフォールバックです。
func (c *EmoConfig) SetDefaults() {
	if c.Bg == "" {
		c.Bg = "#1D3557"
	}
	if c.FontColor == "" {
		c.FontColor = "#FFFFFF"
	}
	if c.Speed == 0 {
		c.Speed = 1.0
	}
	if c.Out == "" {
		if c.Rotate != "" || c.Revolve != "" || c.ScrollX != "" || c.ScrollY != "" || c.Pulsing != "" || c.FontColor == "gaming" {
			c.Out = "emoji.gif"
		} else {
			c.Out = "emoji.png"
		}
	}
}

// Validate は設定の整合性を検証します。
func (c *EmoConfig) Validate() error {
	if c.Text == "" {
		return fmt.Errorf("TEXT argument is required")
	}
	if c.Rotate != "" && c.Revolve != "" {
		return fmt.Errorf("--rotate and --revolve cannot be used together")
	}
	for name, val := range map[string]string{
		"rotate": c.Rotate, "revolve": c.Revolve,
		"scroll-x": c.ScrollX, "scroll-y": c.ScrollY,
		"pulsing": c.Pulsing,
	} {
		if err := validateAnimValue(name, val); err != nil {
			return err
		}
	}
	if c.Speed < 0.5 || c.Speed > 2.0 {
		return fmt.Errorf("--speed must be between 0.5 and 2.0")
	}
	return nil
}

func validateAnimValue(name, val string) error {
	switch val {
	case "", "true", "reverse":
		return nil
	default:
		return fmt.Errorf("--%s must be \"true\" or \"reverse\", got %q", name, val)
	}
}
