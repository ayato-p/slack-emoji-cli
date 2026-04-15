# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## IMPORTANT: Before Any Implementation

**実装に入る前に必ず `advisor()` を呼び出してアドバイスを求めること。**

This applies to any code writing, editing, or architectural decision. Orientation tasks (reading files, searching, building context) may proceed first, but call `advisor()` before the first substantive change.

---

## Quick Start

| Command | Description |
|---|---|
| `go build -o emo .` | Build |
| `./emo [options] TEXT` | Run (e.g., `./emo 'hello'`) |
| `go install .` | Install to `$GOBIN` |

## Permissions

The following commands run automatically without confirmation:
- `./emo [options]` — run the emoji CLI tool (any flags/arguments)
- `go build`, `go test`, `go install` — build and test

---

## Project Overview

`slack-emoji-cli` generates Slack emoji images (128×128px PNG/GIF) from text. Supports multiple animation styles (rotate, revolve, scroll-x/y) and customizable backgrounds.

### Tech Stack

- **Language**: Go 1.25.1
- **CLI**: cobra (command structure) + viper (configuration parsing)
- **Graphics**: `gg` for 2D graphics, `image/gif` and `image/png` for encoding
- **Font**: System default or `--font` flag; discovery via `github.com/flopp/go-findfont`
- **Release**: goreleaser (multi-platform builds on git tags)

---

## Architecture

### Core Flow: Text → Image

```
CLI Args (cobra flags)
  ↓
viper.BindPFlags() + viper.Unmarshal()
  ↓
EmoConfig (config/emoconfig.go)
  ↓
render.Run(cfg)
  ↓
PNG (static) or GIF (animated)
```

### Package Organization

| Path | Role |
|---|---|
| `config/emoconfig.go` | Central config struct, `SetDefaults()`, `Validate()` |
| `render/run.go` | Orchestrator — dispatches to specific renderers |
| `render/render.go` | Static PNG rendering |
| `render/rendergif.go` | Animated GIF with transformers |
| `render/revolve.go` | Revolve animation (characters orbit canvas center) |
| `render/transformer.go` | Animation transformer interface + Rotate type |
| `render/font.go` | Font loading and text measurement |
| `main.go` | cobra.Command definition, flag registration, font resolution |

### Key Design Patterns

**EmoConfig as Central Representation**  
All input (CLI flags, future config files) unmarshals into `EmoConfig`. Decouples parsing from rendering.

**mapstructure Tags for Hyphenated Flags**  
CLI flags use hyphens (`--scroll-x`); Go fields are PascalCase. Tags bridge the gap:
```go
type EmoConfig struct {
    ScrollX string `mapstructure:"scroll-x"`
}
```

**Animation Strings**  
Animation flags hold three states: `""` (off), `"true"` (normal), `"reverse"` (reversed).  
`animToFlags()` in `run.go` converts to `*bool`: `nil` / `&false` / `&true`.

**NoOptDefVal for Boolean-Like Flags**  
`--rotate` → `"true"`, `--rotate=reverse` → `"reverse"`. Requires `NoOptDefVal = "true"` in `init()`.

---

## Adding New Features

### New Animation Option

1. **`config/emoconfig.go`** — add field:
   ```go
   MyAnim string `json:"my-anim,omitempty" mapstructure:"my-anim"`
   ```
2. **`config/emoconfig.go`** — add mutual exclusion in `Validate()` if needed
3. **`main.go`** — register flag with `NoOptDefVal`:
   ```go
   f.String("my-anim", "", "description")
   f.Lookup("my-anim").NoOptDefVal = "true"
   ```
4. **`render/run.go`** — dispatch in `Run()`:
   ```go
   myAnim := animToFlags(cfg.MyAnim)
   opts = append(opts, withMyAnim(myAnim))
   ```

### New Configuration Option (Non-Animation)

1. **`config/emoconfig.go`** — add field:
   ```go
   Blur float64 `json:"blur,omitempty" mapstructure:"blur"`
   ```
2. **`config/emoconfig.go`** — add validation in `Validate()` / defaults in `SetDefaults()`
3. **`main.go`** — register flag (no `NoOptDefVal`):
   ```go
   f.Float64("blur", 0, "blur radius (0–10)")
   ```
4. **`render/`** — use the new field in rendering functions

### Update GitHub Actions Samples

When adding any new CLI option, update both workflow files to include sample images:

**`.github/workflows/gh-pages.yml`**
- Add generation command in the appropriate section
- Add corresponding row(s) to the `index.html` table

**`.github/workflows/pr-preview.yml`**
- Add the same generation commands
- Add matching markdown rows to the GitHub script section

Guidelines: group related samples, include 3–4 representative examples, use `{feature}-{variant}.{png|gif}` filenames.

---

## Configuration Reference

**Output filename defaults** (`config.SetDefaults()`):
- Any animation flag set → `emoji.gif`
- Otherwise → `emoji.png`

**Speed** (`--speed`, range 0.5–2.0):  
Controls GIF frame count; validated in `config.Validate()`.

**Color parsing** (`render/run.go`):  
Accepts `#RGB`, `#RRGGBB`, `#RRGGBBAA`, or `"transparent"`. Returns `color.RGBA`.

---

## Integration Testing

コードを修正する際は以下のワークフローを必ず守ること：

1. **Before キャプチャ（自動）**: 最初の Edit/Write 前に PreToolUse フックが自動で `/tmp/emo/before/` にスナップショットを生成する（初回のみ・約20〜30秒かかる）
2. **実装完了後の検証（必須）**: `/emo-integration-test after <変更したケース名...>` を実行する
   - ケース名は `bash scripts/integration_test.sh list` で確認
   - 変更したオプションに関連するケースをすべて指定する（対応表は `.claude/commands/emo-integration-test.md`）
   - 例: `--rotate` 関連 → `/emo-integration-test after rotate rotate-reverse scroll-x-rotate pulsing-rotate fontcolor-gaming-rotate scroll-x-scroll-y-rotate`
3. **FAIL 時**: 予期しない変更があれば原因を調査・修正してから再検証する
4. **リセット**: スキルが自動で `/tmp/emo/before` と `/tmp/emo/after` を削除し次のセッションに備える

---

## References

- [ARCHITECTURE.md](ARCHITECTURE.md) — rendering orchestration, dependency graph, validation invariants
- [cobra/viper design memory](memory/design_cobra_viper.md) — EmoConfig strategy discussion
