#!/bin/bash
# GitHub Actions ワークフロー更新チェッカー
# 新しいオプション追加時に gh-pages.yml と pr-preview.yml のサンプル更新が必要か確認

OPTION_NAME="${1:?オプション名を指定してください (例: font-color, blur)}"
OPTION_SHORT="${2:?ショートネームを指定してください (例: c, b) または - を指定}"

echo "==============================================="
echo "GitHub Actions ワークフロー更新チェックリスト"
echo "==============================================="
echo ""
echo "新しいオプション: --${OPTION_NAME}${OPTION_SHORT:+ / -${OPTION_SHORT}}"
echo ""
echo "以下のファイルにサンプル画像生成コマンドを追加してください："
echo ""
echo "1. .github/workflows/gh-pages.yml"
echo "   - 「サンプル画像生成」セクションに ./emo コマンド 3-4 個追加"
echo "   - 「index.html 生成」の該当テーブルセクションに <tr> 行を追加"
echo ""
echo "2. .github/workflows/pr-preview.yml"
echo "   - 「プレビュー画像生成」セクションに ./emo コマンド 3-4 個追加"
echo "   - GitHub script の body 配列に markdown テーブル行を追加"
echo ""
echo "CLAUDE.md の「After Adding Any New Option」セクションを参照してください。"
echo ""
echo "==============================================="
