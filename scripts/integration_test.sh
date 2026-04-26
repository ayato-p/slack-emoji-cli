#!/bin/bash
# Integration test script for slack-emoji-cli
# Usage:
#   bash scripts/integration_test.sh generate <output_dir>
#   bash scripts/integration_test.sh compare <before_dir> <after_dir> [expected_case...]
#   bash scripts/integration_test.sh list

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BINARY="/tmp/emo/emo"
IMGCMP="/tmp/emo/imgcmp"
TEST_TEXT="テスト"
MULTILINE_TEXT='東京都\港区'

# ──────────────────────────────────────────────
# build_binary: バイナリをビルドして /tmp/emo/emo に配置
# ──────────────────────────────────────────────
build_binary() {
  mkdir -p /tmp/emo
  echo "Building binary..."
  (cd "$PROJECT_ROOT" && go build -o "$BINARY" ./cmd/emo)
  echo "Build complete: $BINARY"
}

# ──────────────────────────────────────────────
# build_imgcmp: imgcmp ツールをビルドして /tmp/emo/imgcmp に配置
# ──────────────────────────────────────────────
build_imgcmp() {
  mkdir -p /tmp/emo
  (cd "$PROJECT_ROOT" && go build -o "$IMGCMP" ./cmd/imgcmp)
}

# ──────────────────────────────────────────────
# run_case: 1ケースを実行して output_dir に保存
#   $1 = ファイル名（拡張子なし）
#   $2 = 出力ディレクトリ
#   $3以降 = emo に渡すフラグ
# ──────────────────────────────────────────────
run_case() {
  local name="$1"
  local outdir="$2"
  shift 2
  local flags=("$@")

  # 拡張子の自動判定: アニメーションフラグまたは --font-color gaming があれば gif
  local ext="png"
  local prev=""
  for f in "${flags[@]}"; do
    case "$f" in
      --rotate|--rotate=*|--revolve|--revolve=*|\
      --scroll-x|--scroll-x=*|--scroll-y|--scroll-y=*|\
      --pulsing|--pulsing=*)
        ext="gif"
        ;;
    esac
    if [[ "$prev" == "--font-color" || "$prev" == "-c" ]] && [[ "$f" == "gaming" ]]; then
      ext="gif"
    fi
    prev="$f"
  done

  local outfile="$outdir/${name}.${ext}"
  "$BINARY" "${flags[@]}" -o "$outfile" "$TEST_TEXT" 2>/dev/null
}

# ──────────────────────────────────────────────
# run_case_t: カスタムテキストで1ケースを実行
#   $1 = ファイル名（拡張子なし）
#   $2 = 出力ディレクトリ
#   $3 = テキスト
#   $4以降 = emo に渡すフラグ
# ──────────────────────────────────────────────
run_case_t() {
  local name="$1"
  local outdir="$2"
  local text="$3"
  shift 3
  local flags=("$@")

  local ext="png"
  local prev=""
  for f in "${flags[@]}"; do
    case "$f" in
      --rotate|--rotate=*|--revolve|--revolve=*|\
      --scroll-x|--scroll-x=*|--scroll-y|--scroll-y=*|\
      --pulsing|--pulsing=*)
        ext="gif"
        ;;
    esac
    if [[ "$prev" == "--font-color" || "$prev" == "-c" ]] && [[ "$f" == "gaming" ]]; then
      ext="gif"
    fi
    prev="$f"
  done

  local outfile="$outdir/${name}.${ext}"
  "$BINARY" "${flags[@]}" -o "$outfile" "$text" 2>/dev/null
}

# ──────────────────────────────────────────────
# generate_all: 全28ケースを output_dir に生成
# ──────────────────────────────────────────────
generate_all() {
  local outdir="$1"
  mkdir -p "$outdir"
  build_binary

  echo "Generating test cases to $outdir ..."

  # ── シングルオプション（16件）──
  run_case "default"              "$outdir"
  run_case "bg-red"               "$outdir" --bg '#E63946'
  run_case "fontcolor-red"        "$outdir" -c '#FF0000'
  run_case "rotate"               "$outdir" --rotate
  run_case "rotate-reverse"       "$outdir" --rotate=reverse
  run_case "revolve"              "$outdir" --revolve
  run_case "revolve-reverse"      "$outdir" --revolve=reverse
  run_case "scroll-x"             "$outdir" --scroll-x
  run_case "scroll-x-reverse"     "$outdir" --scroll-x=reverse
  run_case "scroll-y"             "$outdir" --scroll-y
  run_case "scroll-y-reverse"     "$outdir" --scroll-y=reverse
  run_case "pulsing"              "$outdir" --pulsing
  run_case "pulsing-reverse"      "$outdir" --pulsing=reverse
  run_case "fontcolor-gaming"     "$outdir" --font-color gaming
  run_case "speed-slow-rotate"    "$outdir" --speed 0.5 --rotate
  run_case "speed-fast-rotate"    "$outdir" --speed 2.0 --rotate

  # ── 組み合わせ（12件）──
  run_case "scroll-x-scroll-y"          "$outdir" --scroll-x --scroll-y
  run_case "scroll-x-rotate"            "$outdir" --scroll-x --rotate
  run_case "pulsing-rotate"             "$outdir" --pulsing --rotate
  run_case "pulsing-scroll-x"           "$outdir" --pulsing --scroll-x
  run_case "pulsing-scroll-y"           "$outdir" --pulsing --scroll-y
  run_case "revolve-scroll-x"           "$outdir" --revolve --scroll-x
  run_case "fontcolor-gaming-rotate"    "$outdir" --font-color gaming --rotate
  run_case "fontcolor-gaming-revolve"   "$outdir" --font-color gaming --revolve
  run_case "pulsing-fontcolor-gaming"   "$outdir" --pulsing --font-color gaming
  run_case "bg-black-fontcolor-white"   "$outdir" --bg '#000000' -c '#FFFFFF'
  run_case "scroll-x-scroll-y-rotate"   "$outdir" --scroll-x --scroll-y --rotate
  run_case "pulsing-scroll-x-scroll-y"  "$outdir" --pulsing --scroll-x --scroll-y

  # ── 複数行幅調整（3件）──
  run_case_t "multiline-width-diff"     "$outdir" "$MULTILINE_TEXT"
  run_case_t "multiline-width-no-fit"   "$outdir" "$MULTILINE_TEXT" --no-fit-width
  run_case_t "multiline-width-scroll-x" "$outdir" "$MULTILINE_TEXT" --scroll-x

  local count
  count=$(ls "$outdir" | wc -l)
  echo "Done: $count files generated in $outdir"
}

# ──────────────────────────────────────────────
# compare: before と after を比較して結果を表示
#   $1 = before_dir
#   $2 = after_dir
#   $3以降 = 期待する変更ケース名（拡張子なし）
# ──────────────────────────────────────────────
compare() {
  local before_dir="$1"
  local after_dir="$2"
  shift 2
  local expected=("$@")

  build_imgcmp

  echo ""
  echo "=== Integration Test Comparison ==="
  if [ ${#expected[@]} -gt 0 ]; then
    echo "Expected changes: ${expected[*]}"
  else
    echo "Expected changes: (none specified — showing all diffs)"
  fi
  echo ""

  # before のファイル一覧を取得
  local files
  files=$(ls "$before_dir" | sort)

  local total=0
  local pass=0
  local fail=0
  local missing=0

  while IFS= read -r filename; do
    [ -z "$filename" ] && continue
    total=$((total + 1))

    local before_file="$before_dir/$filename"
    local after_file="$after_dir/$filename"

    # after にファイルが存在しない
    if [ ! -f "$after_file" ]; then
      printf "  [MISS]  %-40s missing from after/\n" "$filename"
      missing=$((missing + 1))
      continue
    fi

    # ファイル名から拡張子なしのケース名を取得
    local casename="${filename%.*}"

    # ハッシュ比較
    local hash_before hash_after
    hash_before=$(sha256sum "$before_file" | awk '{print $1}')
    hash_after=$(sha256sum "$after_file" | awk '{print $1}')

    local is_changed=false
    local visual_note=""
    if [ "$hash_before" = "$hash_after" ]; then
      is_changed=false
    else
      case "$filename" in
        *.gif)
          # GIF はパレット非決定性があるためピクセルレベルで比較
          if "$IMGCMP" "$before_file" "$after_file" > /dev/null 2>&1; then
            is_changed=false
            visual_note=" (hash differs, visually same)"
          else
            is_changed=true
          fi
          ;;
        *)
          is_changed=true
          ;;
      esac
    fi

    # expected に含まれるか確認
    local is_expected=false
    if [ ${#expected[@]} -gt 0 ]; then
      for e in "${expected[@]}"; do
        if [ "$e" = "$casename" ]; then
          is_expected=true
          break
        fi
      done
    fi

    # 判定
    if [ ${#expected[@]} -eq 0 ]; then
      # expected 未指定: 中立表示
      if $is_changed; then
        printf "  [DIFF]  %-40s CHANGED\n" "$filename"
      else
        printf "  [SAME]  %-40s unchanged%s\n" "$filename" "$visual_note"
      fi
      pass=$((pass + 1))
    elif ! $is_changed && ! $is_expected; then
      printf "  [PASS]  %-40s unchanged%s\n" "$filename" "$visual_note"
      pass=$((pass + 1))
    elif $is_changed && $is_expected; then
      printf "  [PASS]  %-40s CHANGED  (expected)\n" "$filename"
      pass=$((pass + 1))
    elif $is_changed && ! $is_expected; then
      printf "  [FAIL]  %-40s CHANGED  (unexpected!)\n" "$filename"
      fail=$((fail + 1))
    elif ! $is_changed && $is_expected; then
      printf "  [FAIL]  %-40s expected change not detected\n" "$filename"
      fail=$((fail + 1))
    fi
  done <<< "$files"

  echo ""
  echo "=== Summary ==="
  if [ ${#expected[@]} -eq 0 ]; then
    echo "Total: $total | SAME: $pass | DIFF: $((total - pass - missing)) | MISSING: $missing"
    echo "Result: (no expected specified — review diffs manually)"
  else
    echo "Total: $total | PASS: $pass | FAIL: $fail | MISSING: $missing"
    if [ "$fail" -eq 0 ] && [ "$missing" -eq 0 ]; then
      echo "Result: PASSED"
    else
      echo "Result: FAILED"
    fi
  fi
  echo ""

  # FAIL があれば終了コード 1 を返す
  if [ "$fail" -gt 0 ] || [ "$missing" -gt 0 ]; then
    return 1
  fi
  return 0
}

# ──────────────────────────────────────────────
# list_cases: 全ケース名（拡張子なし）を一覧表示
# ──────────────────────────────────────────────
list_cases() {
  cat <<'EOF'
# Single-option cases (16)
default
bg-red
fontcolor-red
rotate
rotate-reverse
revolve
revolve-reverse
scroll-x
scroll-x-reverse
scroll-y
scroll-y-reverse
pulsing
pulsing-reverse
fontcolor-gaming
speed-slow-rotate
speed-fast-rotate

# Combination cases (12)
scroll-x-scroll-y
scroll-x-rotate
pulsing-rotate
pulsing-scroll-x
pulsing-scroll-y
revolve-scroll-x
fontcolor-gaming-rotate
fontcolor-gaming-revolve
pulsing-fontcolor-gaming
bg-black-fontcolor-white
scroll-x-scroll-y-rotate
pulsing-scroll-x-scroll-y

# Multi-line width equalization cases (3)
multiline-width-diff
multiline-width-no-fit
multiline-width-scroll-x
EOF
}

# ──────────────────────────────────────────────
# main
# ──────────────────────────────────────────────
main() {
  local cmd="${1:-}"
  case "$cmd" in
    generate)
      local outdir="${2:?Usage: $0 generate <output_dir>}"
      generate_all "$outdir"
      ;;
    compare)
      local before="${2:?Usage: $0 compare <before_dir> <after_dir> [expected...]}"
      local after="${3:?Usage: $0 compare <before_dir> <after_dir> [expected...]}"
      shift 3 || true
      compare "$before" "$after" "$@"
      ;;
    list)
      list_cases
      ;;
    *)
      echo "Usage: $0 <command> [args...]"
      echo ""
      echo "Commands:"
      echo "  generate <output_dir>              Generate all test case images"
      echo "  compare <before> <after> [cases...]  Compare before/after directories"
      echo "  list                               List all test case names"
      exit 1
      ;;
  esac
}

main "$@"
