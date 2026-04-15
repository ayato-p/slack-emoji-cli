#!/bin/bash
# PreToolUse hook: Edit/Write の直前に before スナップショットをキャプチャ
# /tmp/emo/before が既に存在する場合はスキップ（1セッション1回のみ実行）

if [ ! -d "/tmp/emo/before" ]; then
  PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
  echo "📸 Before スナップショット生成中... (初回のみ・しばらくお待ちください)"
  bash "$PROJECT_ROOT/scripts/integration_test.sh" generate /tmp/emo/before
  echo "✅ Before スナップショット完了: /tmp/emo/before"
fi
