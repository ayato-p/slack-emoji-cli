# Architecture

## Package Structure

```
main.go
├── config/
│   └── emoconfig.go       # EmoConfig struct, validation, defaults
├── render/
│   ├── run.go             # Run(EmoConfig) - orchestrates rendering
│   ├── render.go          # Static PNG rendering
│   ├── gif.go             # Animated GIF rendering
│   ├── revolve.go         # Revolving animation
│   ├── transformer.go     # Animation transformers
│   └── font.go            # Font utilities
```

## Dependency Graph

```
main.go (CLI entry)
  ├─→ config/ (EmoConfig, validation)
  └─→ render/ (Run)
        ├─→ config/ (EmoConfig)
        ├─→ render.Render()
        ├─→ render.RenderGIF()
        └─→ render.RevolveGIF()
```

**Key principle:** render package depends on config, not the reverse.

## Configuration Flow

### CLI Path
```
CLI flags
  ↓
cobra command
  ↓
viper.Unmarshal() → EmoConfig
  ↓
EmoConfig.SetDefaults()
  ↓
EmoConfig.Validate()
  ↓
render.Run(EmoConfig)
```

### Future JSON Path (ready to implement)
```
JSON file
  ↓
viper.ReadInConfig() → viper store
  ↓
viper.Unmarshal() → EmoConfig
  ↓ (rest same as CLI)
```

## EmoConfig Design

`config/emoconfig.go` defines:
- **EmoConfig struct**: Central configuration representation
  - `Text`: Required positional argument
  - Animation flags: `Rotate`, `Revolve`, `ScrollX`, `ScrollY` (values: `""` / `"true"` / `"reverse"`)
  - `Speed`: Animation speed multiplier (0.5–2.0)
  - `Bg`: Background color (hex or `"transparent"`)
  - `Out`: Output file path

- **SetDefaults()**: Fills zero-value fields with defaults
- **Validate()**: Checks constraints (mutually exclusive animations, speed range, etc.)

### Tag Strategy
- `json`: For future JSON config file support
- `mapstructure`: For viper.Unmarshal() — **must be explicit for hyphenated keys** ("scroll-x" → ScrollX)

## Rendering Orchestration

`render/run.go` implements `Run(EmoConfig)`:
1. Parses text into lines (split by `\`)
2. Converts hex color string → RGBA
3. Converts animation strings to (set, reverse) flags via `animToFlags()`
4. Builds `ScrollConfig` struct
5. Dispatches to `RevolveGIF()`, `RenderGIF()`, or `Render()` based on flags
6. Encodes output (GIF or PNG) to file

## CLI Integration

`main.go` uses cobra + viper:
- **cobra.Command**: Defines command structure and positional argument validation
- **pflag strings** with `NoOptDefVal = "true"`: Allows `--rotate` to become `"true"` and `--rotate=reverse` to become `"reverse"`
- **viper.BindPFlags()**: Links pflag values into viper's config store
- **viper.Unmarshal()**: Converts viper store → EmoConfig via mapstructure tags

### Flag Definitions
| Flag | Type | Default | Values |
|---|---|---|---|
| `-o, --out` | string | "" | Output file path |
| `--bg` | string | "#1D3557" | Hex color or "transparent" |
| `--rotate` | string | "" | "" / "true" / "reverse" |
| `--revolve` | string | "" | "" / "true" / "reverse" |
| `--scroll-x` | string | "" | "" / "true" / "reverse" |
| `--scroll-y` | string | "" | "" / "true" / "reverse" |
| `--speed` | float64 | 1.0 | 0.5–2.0 |

## Adding New Options

To add a new rendering option:

1. **config/emoconfig.go**: Add field with `json` and `mapstructure` tags
2. **config/emoconfig.go**: Update `Validate()` if constraints apply
3. **main.go**: Register flag in `init()`, set `NoOptDefVal` if boolean-like
4. **render/run.go**: Handle the new config field in `Run()`

Example (hypothetical `--blur` option):
```go
// config/emoconfig.go
type EmoConfig struct {
    Blur float64 `json:"blur,omitempty" mapstructure:"blur"`
}

// main.go init()
f.Float64("blur", 0, "blur radius (0–10)")

// render/run.go Run()
if cfg.Blur > 0 {
    // apply blur transformation
}
```

## Validation Invariants

- `Text` must be non-empty (required positional)
- `Rotate` and `Revolve` are mutually exclusive
- Animation string values must be `""`, `"true"`, or `"reverse"` (validated in config, not render)
- `Speed` must be in [0.5, 2.0]
- Output file extension determines format (currently not validated, inferred from animation flags)
