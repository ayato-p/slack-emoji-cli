//go:build js && wasm

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	_ "embed"
	"strings"
	"syscall/js"

	"github.com/ayato-p/slack-emoji-cli/config"
	"github.com/ayato-p/slack-emoji-cli/render"
)

//go:embed fonts/DejaVuSans.ttf
var defaultFontData []byte

// generateEmoji は JS から呼び出される絵文字生成関数。
// 引数: args[0] = JSON文字列 (config.EmoConfig フィールドに対応)
// 戻り値: { data: "<base64>", mimeType: "image/png"|"image/gif" } または { error: "..." }
func generateEmoji(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("missing config argument")
	}

	var cfg config.EmoConfig
	if err := json.Unmarshal([]byte(args[0].String()), &cfg); err != nil {
		return errorResult(fmt.Sprintf("invalid config JSON: %v", err))
	}

	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return errorResult(err.Error())
	}

	// フォントが未指定の場合は埋め込みデフォルトフォントを使用
	if len(cfg.FontData) == 0 {
		cfg.FontData = defaultFontData
	}

	var buf bytes.Buffer
	if err := render.RunTo(&buf, cfg); err != nil {
		return errorResult(err.Error())
	}

	mimeType := "image/png"
	if strings.HasSuffix(cfg.Out, ".gif") {
		mimeType = "image/gif"
	}

	return map[string]any{
		"data":     base64.StdEncoding.EncodeToString(buf.Bytes()),
		"mimeType": mimeType,
	}
}

func errorResult(msg string) map[string]any {
	return map[string]any{"error": msg}
}

func main() {
	js.Global().Set("generateEmoji", js.FuncOf(generateEmoji))
	// WASM モジュールが GC されないようにブロック
	select {}
}
