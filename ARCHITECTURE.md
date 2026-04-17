# Architecture

## Package Structure

Follows the Go standard CLI layout: binaries under `cmd/<bin>/`, internal
packages (not importable from outside the module) under `internal/`.

```
cmd/
├── emo/
│   └── main.go            # CLI entry: cobra/viper wiring, font resolution
└── imgcmp/
    └── main.go            # Image-compare helper used by integration tests
internal/
├── config/
│   └── emoconfig.go       # EmoConfig struct, validation, defaults
└── render/
    ├── run.go             # Run(EmoConfig) - orchestrates rendering
    ├── render.go          # Effect system, layout builders, unified renderFrame
    ├── rendergif.go       # GIF assembly + compositeWithWrap
    └── font.go            # Font utilities
```

## Dependency Graph

```
cmd/emo/main.go (CLI entry)
  ├─→ internal/config/ (EmoConfig, validation)
  └─→ internal/render/ (Run)
        ├─→ internal/config/ (EmoConfig)
        ├─→ buildEffect() → func(frame, total int) EffectModel
        └─→ renderFrame(EffectModel) → image.Image
              └─→ EffectModel.DrawContent(ctx, params)
```

**Key principle:** 
- render package depends on config, not the reverse
- Each frame effect computes its own animation parameters
- Effects compose via `composeEffects()` to build a unified `frameEffect`
- Rendering strategy (text vs revolve) is captured as `DrawContent` in `EffectModel`; `renderFrame` is a single unified function

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

`internal/config/emoconfig.go` defines:
- **EmoConfig struct**: Central configuration representation
  - `Text`: Required positional argument
  - `Font`: Font file path (resolved in cmd/emo/main.go; empty = system default)
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

### Frame Effect Composition (internal/render/render.go)

**FrameParams**: Encapsulates per-frame animation state
```go
type FrameParams struct {
    RotationAngle float64  // Canvas rotation in radians
    RevolveOffset float64  // Orbit offset angle in radians
    ScrollX       float64  // Horizontal pixel offset
    ScrollY       float64  // Vertical pixel offset
}
```

**frameEffect**: Function that computes FrameParams for a given frame
```go
type frameEffect func(frame, total int) FrameParams
```

**Effect Functions** (each independently computes one parameter):
- `scrollXEffect(reverse bool)` → FrameParams with ScrollX
- `scrollYEffect(reverse bool)` → FrameParams with ScrollY
- `rotateEffect(reverse bool)` → FrameParams with RotationAngle
- `revolveEffect(reverse bool)` → FrameParams with RevolveOffset

**composeEffects()**: Combines effects by summing their contributions
```go
composeEffects(effects...) → frameEffect that merges all parameters
```

### Effect Builders (internal/render/render.go)

**buildTextEffect(font, spec)** → `func(frame, total int) EffectModel`
- Computes text layout (font metrics, baseline, scroll tile sizes)
- Per-frame closure: resolves color, returns `EffectModel` with `DrawContent` capturing text drawing logic

**buildRevolveEffect(font, spec)** → `func(frame, total int) EffectModel`
- Computes orbit geometry (orbit radius, character positions) from font metrics
- Per-frame closure: resolves color, returns `EffectModel` with `DrawContent` capturing revolve drawing logic

**DrawContent** (`func(*gg.Context, FrameParams)` stored in EffectModel):
- Text mode: applies scale/rotate transforms, draws lines with scroll tiling
- Revolve mode: uses `drawChars` local helper, applies scroll via `compositeWithWrap` when needed

### GIF Assembly (internal/render/rendergif.go)

**composeGIF(renderFn, numFrames, delay)** → *gif.GIF
- Calls renderFn for each frame
- Converts to paletted image + Floyd-Steinberg dithering
- Assembles into GIF with uniform delay

### Run() Orchestration (internal/render/run.go)

`internal/render/run.go` implements `Run(EmoConfig)`:
1. Parse text into lines (split by `\`)
2. Load font from `cfg.Font` (already resolved to absolute path in cmd/emo/main.go)
3. Convert hex color string → RGBA
4. Convert animation strings to `*bool` flags via `animToFlags()`
5. Build effect function via `buildEffect(font, opts...)` using Functional Option pattern:
   - `withLines()`, `withBg()`: Store base configuration
   - `withScrollX()`, `withScrollY()`, `withRotate()`, `withRevolve()`, `withPulse()`, `withGaming()`: Add effects
   - `buildEffect()` dispatches to `buildTextEffect()` or `buildRevolveEffect()`
6. Check if animation needed → call `composeGIF()` or render static frame
7. Encode output (GIF or PNG) to file

## CLI Integration

`cmd/emo/main.go` uses cobra + viper:
- **cobra.Command**: Defines command structure and positional argument validation
- **pflag strings** with `NoOptDefVal = "true"`: Allows `--rotate` to become `"true"` and `--rotate=reverse` to become `"reverse"`
- **viper.BindPFlags()**: Links pflag values into viper's config store
- **viper.Unmarshal()**: Converts viper store → EmoConfig via mapstructure tags

### Flag Definitions
| Flag | Type | Default | Values |
|---|---|---|---|
| `-o, --out` | string | "" | Output file path |
| `--bg` | string | "#1D3557" | Hex color or "transparent" |
| `--font` | string | "" | Font file path or system font name (empty = auto-select first available) |
| `--rotate` | string | "" | "" / "true" / "reverse" |
| `--revolve` | string | "" | "" / "true" / "reverse" |
| `--scroll-x` | string | "" | "" / "true" / "reverse" |
| `--scroll-y` | string | "" | "" / "true" / "reverse" |
| `--speed` | float64 | 1.0 | 0.5–2.0 |

## Adding New Options

### Adding a New Animation Effect

To add a new animation effect (e.g., `--skew`):

1. **internal/render/render.go**: Define an effect function
   ```go
   func skewEffect(reverse bool) frameEffect {
       return func(frame, total int) FrameParams {
           // Compute skew parameter based on frame progress
           // Return FrameParams with desired field set
           return FrameParams{/* ... */}
       }
   }
   ```

2. **internal/render/render.go**: Add a rendererOption helper
   ```go
   func withSkew(reverse bool) rendererOption {
       return func(s *rendererSpec) { 
           s.effects = append(s.effects, skewEffect(reverse)) 
       }
   }
   ```

3. **internal/render/render.go**: Update `FrameParams` if needed to carry new animation data
   ```go
   type FrameParams struct {
       // ... existing fields
       SkewAngle float64  // New field
   }
   ```

4. **internal/render/render.go**: Apply the new parameter inside the `DrawContent` closure in `buildTextEffect` / `buildRevolveEffect`
   - In `buildTextEffect` DrawContent: use `p.SkewAngle` to transform context
   - In `buildRevolveEffect` DrawContent: use `p.SkewAngle` to adjust character positioning

5. **internal/config/emoconfig.go**: Add config field with `json` and `mapstructure` tags
   ```go
   type EmoConfig struct {
       Skew string `json:"skew,omitempty" mapstructure:"skew"`  // "" / "true" / "reverse"
   }
   ```

6. **internal/config/emoconfig.go**: Update `Validate()` if constraints apply (mutual exclusions, etc.)

7. **cmd/emo/main.go**: Register flag in `init()`, set `NoOptDefVal` if boolean-like
   ```go
   f.String("skew", "", "skew animation")
   f.Lookup("skew").NoOptDefVal = "true"
   ```

8. **internal/render/run.go**: Add to option builder in `Run()`
   ```go
   skewSet, skewReverse := animToFlags(cfg.Skew)
   if skewSet {
       opts = append(opts, withSkew(skewReverse))
   }
   ```

### Adding a Non-Animation Configuration Option

Example (`--blur` parameter, not animating):

1. **FrameParams** does not need updating (blur is not frame-dependent)

2. **internal/render/render.go or internal/render/revolve.go**: Pass blur value as a captured variable in the renderer factory
   - Store in rendererSpec and pass to newTextRenderer/newRevolveRenderer
   - Apply during rendering based on the blur value

3. **internal/config/emoconfig.go**: Add field
   ```go
   type EmoConfig struct {
       Blur float64 `json:"blur,omitempty" mapstructure:"blur"`
   }
   ```

4. **cmd/emo/main.go**: Register flag
   ```go
   f.Float64("blur", 0, "blur radius (0–10)")
   ```

5. **internal/render/run.go**: Include in rendererSpec/buildRenderer if needed
   ```go
   opts = append(opts, withBlur(cfg.Blur))
   ```

## Validation Invariants

- `Text` must be non-empty (required positional)
- `Rotate` and `Revolve` are mutually exclusive (validated in internal/config/emoconfig.go)
- Animation string values must be `""`, `"true"`, or `"reverse"` (validated in internal/config/emoconfig.go)
- `Speed` must be in [0.5, 2.0] (validated in internal/config/emoconfig.go)
- Output file extension determines format (currently not validated, inferred from animation flags in internal/config/emoconfig.go)

## Design Principles

### Separation of Concerns

1. **Config layer** (internal/config/emoconfig.go): Validates user intent, converts flags to (set, reverse) pairs
2. **Effect layer** (internal/render/render.go): Each effect independently computes per-frame parameters
3. **Renderer layer** (internal/render/render.go, internal/render/revolve.go): Applies effects to graphics, handles rendering
4. **Assembly layer** (internal/render/rendergif.go): Composes frames into GIF

### Composability

Effects are independent and additive:
- `scrollXEffect` + `rotateEffect` → both parameters applied each frame
- `scrollXEffect` + `scrollYEffect` + `rotateEffect` → all three parameters applied
- `scrollXEffect` + `revolveEffect` → both scroll and revolve active in same frame
- No mutual exclusivity needed in effect layer (enforced at config layer)
