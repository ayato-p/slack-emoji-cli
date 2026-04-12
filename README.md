# slack-emoji-cli

テキストからSlack絵文字用の画像（128×128px PNG）を生成するCLIツール。

**サンプル**: https://ayato-p.github.io/slack-emoji-cli/

## インストール

```bash
go install github.com/ayato-p/slack-emoji-cli@latest
```

またはソースからビルド：

```bash
git clone https://github.com/ayato-p/slack-emoji-cli.git
cd slack-emoji-cli
go build -o emo .
```

## フォント

このツールはシステムのフォントを使用します。日本語を含むテキストを生成する場合、以下のいずれかが必要です：

**オプション 1: Noto Sans CJK JP をインストール（推奨）**

```bash
# Ubuntu/Debian
sudo apt install fonts-noto-cjk

# macOS (Homebrew)
brew install font-noto-nerd-font

# または手動でダウンロード
# https://github.com/google/fonts/tree/main/ofl/notosanscjk
```

**オプション 2: `--font` で明示的に指定**

```bash
emo --font /path/to/font.ttf 'テキスト'
# またはシステムフォント名で検索
emo --font 'NotoSansCJKJP-Regular' 'テキスト'
```

## 使い方

```bash
emo [options] TEXT
```

`\` でテキストを区切ると複数行になります。

```bash
# 2行テキスト → emoji.png に保存
emo '猫に\小判'

# 出力先を指定
emo -o /tmp/my-emoji.png '猫に\小判'

# 背景色を変える
emo --bg '#E63946' 'お気持ち'

# フォント色を変える
emo -c '#FF0000' 'お気持ち'

# 背景色とフォント色を変える
emo --bg '#000000' -c '#FFFFFF' 'コントラスト'

# 1行
emo '猫'
```

> **シェルでの注意**: `\` を正しく渡すにはシングルクォート推奨。ダブルクォートの場合は `\\` とエスケープしてください。

## オプション

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-o`   | `emoji.png` | 出力ファイルパス（アニメ時は `emoji.gif`） |
| `--bg` | `#1D3557` | 背景色（`#RGB` / `#RRGGBB` / `#RRGGBBAA` または `"transparent"`） |
| `-c`, `--font-color` | `#FFFFFF` | フォント色（`#RGB` / `#RRGGBB` / `#RRGGBBAA` または `"transparent"`） |
| `--font` | システムのデフォルト | フォントファイルのパス、またはシステムフォント名（例：`DejaVuSans.ttf`） |
| `--rotate` | — | 回転アニメGIFを生成。`=reverse` で逆方向 |
| `--revolve` | — | 文字が中心を公転するアニメGIFを生成。`=reverse` で逆方向 |
| `--scroll-x` | — | 水平スクロールアニメGIFを生成。`=reverse` で逆方向 |
| `--scroll-y` | — | 垂直スクロールアニメGIFを生成。`=reverse` で逆方向 |
| `--speed` | `1.0` | アニメーション速度倍率（0.5–2.0、GIFのみ） |

## 出力仕様

- フォーマット: PNG
- サイズ: 128×128px（Slack推奨サイズ）
- フォント: システムのデフォルトフォント（`--font` で変更可能）
- フォントサイズ: テキストが収まるよう自動調整

## ライセンス

MIT License — 詳細は [LICENSE](LICENSE) を参照。

サードパーティコンポーネントのライセンスは [NOTICE](NOTICE) および [LICENSES/](LICENSES/) を参照。
