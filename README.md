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

# 1行
emo '猫'
```

> **シェルでの注意**: `\` を正しく渡すにはシングルクォート推奨。ダブルクォートの場合は `\\` とエスケープしてください。

## オプション

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-o`   | `emoji.png` | 出力ファイルパス（アニメ時は `emoji.gif`） |
| `--bg` | `#1D3557` | 背景色（`#RGB` / `#RRGGBB` / `#RRGGBBAA`） |
| `--rotate` | — | 回転アニメGIFを生成。`=reverse` で逆方向 |
| `--revolve` | — | 文字が中心を公転するアニメGIFを生成。`=reverse` で逆方向 |

## 出力仕様

- フォーマット: PNG
- サイズ: 128×128px（Slack推奨サイズ）
- フォント: Noto Sans JP（バイナリに埋め込み済み、実行環境へのフォントインストール不要）
- フォントサイズ: テキストが収まるよう自動調整

## ライセンス

MIT License — 詳細は [LICENSE](LICENSE) を参照。

サードパーティコンポーネントのライセンスは [NOTICE](NOTICE) および [LICENSES/](LICENSES/) を参照。
