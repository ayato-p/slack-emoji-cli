# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Start

**Build**: `go build -o emo .`  
**Run**: `./emo [options] TEXT` (e.g., `./emo 'hello'`)  
**Install**: `go install .` (installs to $GOBIN)

## Permissions

The following commands can be executed automatically without confirmation:
- `./emo [options]` — Running the emoji CLI tool (with any flags and arguments)
- `go build`, `go test`, `go install` — Build and test commands

## Project Overview

`slack-emoji-cli` is a CLI tool that generates Slack emoji images (128×128px PNG/GIF) from text. It supports multiple animation styles (rotate, revolve, scroll-x/y) and customizable backgrounds.

### Tech Stack
- **Language**: Go 1.25.1
- **CLI Framework**: cobra (command structure) + viper (configuration parsing)
- **Graphics**: `gg` library for 2D graphics, `image/gif` and `image/png` for encoding
- **Font**: System default or user-specified via `--font` flag; uses `github.com/flopp/go-findfont` for font discovery
- **Release**: goreleaser (multi-platform builds triggered on git tags)

## Architecture

### Core Flow: Text → Image

```
CLI Args (cobra flags)
  ↓
viper.BindPFlags() + viper.Unmarshal()
  ↓
EmoConfig (config/emoconfig.go)
  ↓
render.Run(cfg) orchestrates rendering
  ↓
PNG (static) or GIF (animated)
```

### Package Organization

- **`config/emoconfig.go`**: Central configuration struct + validation
  - `EmoConfig`: Holds all settings (Text, animation flags, speed, colors, output path)
  - `SetDefaults()`: Fills missing fields with defaults (e.g., output filename based on animation type)
  - `Validate()`: Checks constraints (mutually exclusive animations, speed range [0.5–2.0], etc.)

- **`render/`**: Rendering pipeline
  - `run.go`: **Orchestrator** — calls specific render functions based on config
  - `render.go`: Static PNG rendering for single image
  - `rendergif.go`: Animated GIF with transformers (rotate, scroll)
  - `revolve.go`: Specialized revolve animation (characters orbit canvas center)
  - `transformer.go`: Animation transformers (Rotate type + interface)
  - `font.go`: Font loading from file paths and text measurement utilities

- **`main.go`**: CLI entry point
  - cobra.Command definition with flag registration
  - `init()`: Sets up flags with mapstructure tags for viper unmarshaling
  - **Key detail**: Animation flags use `NoOptDefVal = "true"` to allow `--rotate` (becomes `"true"`) and `--rotate=reverse` (becomes `"reverse"`)
  - **Font resolution** (`resolveFont()`): Empty `--font` → pick first system font; explicit path/name → look up via `findfont.Find()`

### Key Design Patterns

**EmoConfig as Central Representation**  
All input (CLI flags, future JSON config) unmarshals into `EmoConfig`. This decouples input parsing from rendering logic and enables future config file support.

**mapstructure Tags for Hyphenated Flags**  
CLI flags use hyphens (`--scroll-x`), but Go struct fields are PascalCase. mapstructure tags explicitly map `"scroll-x"` → `ScrollX`:
```go
type EmoConfig struct {
    ScrollX string `mapstructure:"scroll-x"`
}
```

**Animation Strings**  
Animation flags store three states as strings: `""` (off), `"true"` (normal), `"reverse"` (reversed). `animToFlags()` in `run.go` converts these to `*bool`: `nil` (off), `&false` (normal), `&true` (reversed).

## Adding New Features

### Example: Adding a New Animation Option

1. **config/emoconfig.go**: Add field with mapstructure tag
   ```go
   type EmoConfig struct {
       MyAnim string `json:"my-anim,omitempty" mapstructure:"my-anim"`
   }
   ```

2. **config/emoconfig.go**: Update `Validate()` if mutual exclusions apply
   ```go
   if c.MyAnim != "" && c.SomeOtherAnim != "" {
       return fmt.Errorf("--my-anim and --some-other-anim cannot be used together")
   }
   ```

3. **main.go**: Register the flag in `init()`
   ```go
   f.String("my-anim", "", "description of animation")
   f.Lookup("my-anim").NoOptDefVal = "true"
   ```

4. **render/run.go**: Handle in `Run()` dispatcher
   ```go
   myAnim := animToFlags(cfg.MyAnim)
   opts = append(opts, withMyAnim(myAnim))  // withMyAnim returns nil if myAnim is nil
   ```

### Example: Adding a New Configuration Option (Non-Animation)

For a simple option like `--blur`:

1. **config/emoconfig.go**: Add field with mapstructure tag
   ```go
   Blur float64 `json:"blur,omitempty" mapstructure:"blur"`
   ```

2. **config/emoconfig.go**: Update `SetDefaults()` if needed, add `Validate()` constraints
   ```go
   if c.Blur < 0 || c.Blur > 10 {
       return fmt.Errorf("--blur must be 0–10, got %v", c.Blur)
   }
   ```

3. **main.go**: Register without NoOptDefVal (it's not a boolean-like flag)
   ```go
   f.Float64("blur", 0, "blur radius (0–10)")
   ```

4. **render/**: Modify rendering functions to use the new config field

### After Adding Any New Option: Update GitHub Actions Samples

When adding any new CLI option (animation, color, or parameter), you **must** update the GitHub Actions workflows to generate sample images. This ensures both `gh-pages` and PR preview include the new feature.

**Always add samples to both files:**

1. **`.github/workflows/gh-pages.yml`** — Sample generation + index.html update
   - Add generation command in the appropriate section (e.g., after background color samples)
   - Add corresponding row(s) to the `index.html` table section
   - Example: For `--font-color`, add:
     ```bash
     ./emo -c '#FF0000' -o sample-images/font-red.png 'Red'
     ```
   - And add an HTML table row with the command and image reference

2. **`.github/workflows/pr-preview.yml`** — Same samples + PR comment update
   - Add the same generation commands to the "プレビュー画像生成" step
   - Add corresponding markdown table rows to the GitHub script section
   - Example table row:
     ```javascript
     `| 赤 (#FF0000) | \`emo -c '#FF0000' 'Red'\` | ![Red](${base}/font-red.png) |`,
     ```

**Pattern to follow:**
- Group related samples (e.g., all color variations together)
- Include at least 3–4 representative examples
- Use consistent image filenames: `{feature}-{variant}.{png|gif}`

## Configuration

**Output Filename Defaults** (set in `config.SetDefaults()`):
- If any animation flag is set → `emoji.gif`
- Otherwise → `emoji.png`

**Speed** (animation frame count control):
- Range: 0.5–2.0
- Validation in `config.Validate()`
- Used by `RenderGIF()` to scale frame count

**Color Parsing** (render/run.go):
- Supports: `#RGB`, `#RRGGBB`, `#RRGGBBAA`, or literal `"transparent"`
- Returns `color.RGBA`

## References

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed rendering orchestration, dependency graph, and validation invariants.

See [cobra/viper design memory](file:///home/ayato-p/.claude/projects/-home-ayato-p-sources-github-com-ayato-p-slack-emoji-cli/memory/design_cobra_viper.md) for discussion of EmoConfig strategy.
