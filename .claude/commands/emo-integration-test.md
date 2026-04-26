# emo-integration-test

slack-emoji-cli のインテグレーションテストを実行するスキル。
コード修正の before/after で画像を生成し、意図した変更のみが変化していることを確認する。

## 使い方

```
/emo-integration-test after [ケース名...]
/emo-integration-test before
/emo-integration-test list
```

## 実行手順

引数の第1トークンを確認してフェーズを判断する。

### `after [ケース名...]` フェーズ（実装完了後の検証）

1. **after 生成**: 以下を実行する
   ```
   bash scripts/integration_test.sh generate /tmp/emo/after
   ```

2. **比較**: ケース名が指定されていれば expected として渡す
   ```
   bash scripts/integration_test.sh compare /tmp/emo/before /tmp/emo/after [ケース名...]
   ```

3. **結果の解釈**:
   - `Result: PASSED` → 変更は意図通り。「インテグレーションテスト PASSED」と報告する
   - `Result: FAILED` → 予期しない変更または期待した変更が検出されなかった。原因を調査して修正する
   - expected を省略した場合は DIFF 一覧を確認し、変更が妥当かを報告する

4. **リセット**: テスト完了後は必ず以下を実行して次のセッションに備える
   ```
   rm -rf /tmp/emo/before /tmp/emo/after
   ```

### `before` フェーズ（手動リセット）

before スナップショットを削除して再生成する（セッションをリセットしたいときに使う）：
```
rm -rf /tmp/emo/before
bash scripts/integration_test.sh generate /tmp/emo/before
```

### `list` フェーズ（ケース名の確認）

```
bash scripts/integration_test.sh list
```

全31ケース名を表示する。`after` に渡すケース名が不明なときに使う。

## ケース名とオプションの対応

変更したオプションから渡すべきケース名を選ぶ目安：

| 変更したオプション | 渡すべきケース名 |
|---|---|
| `--rotate` 関連 | `rotate`, `rotate-reverse`, `scroll-x-rotate`, `pulsing-rotate`, `fontcolor-gaming-rotate`, `scroll-x-scroll-y-rotate` |
| `--revolve` 関連 | `revolve`, `revolve-reverse`, `revolve-scroll-x`, `fontcolor-gaming-revolve` |
| `--scroll-x` 関連 | `scroll-x`, `scroll-x-reverse`, `scroll-x-scroll-y`, `scroll-x-rotate`, `pulsing-scroll-x`, `revolve-scroll-x`, `scroll-x-scroll-y-rotate`, `pulsing-scroll-x-scroll-y` |
| `--scroll-y` 関連 | `scroll-y`, `scroll-y-reverse`, `scroll-x-scroll-y`, `pulsing-scroll-y`, `scroll-x-scroll-y-rotate`, `pulsing-scroll-x-scroll-y` |
| `--pulsing` 関連 | `pulsing`, `pulsing-reverse`, `pulsing-rotate`, `pulsing-scroll-x`, `pulsing-scroll-y`, `pulsing-fontcolor-gaming`, `pulsing-scroll-x-scroll-y` |
| `--font-color gaming` 関連 | `fontcolor-gaming`, `fontcolor-gaming-rotate`, `fontcolor-gaming-revolve`, `pulsing-fontcolor-gaming` |
| `--speed` 関連 | `speed-slow-rotate`, `speed-fast-rotate` |
| `--bg` / `--font-color` 関連 | `bg-red`, `fontcolor-red`, `bg-black-fontcolor-white` |
| `--no-fit-width` 関連 | `multiline-width-diff`, `multiline-width-no-fit`, `multiline-width-scroll-x` |
| 共通処理（全体に影響） | 全ケース名を指定する（または expected なしで実行する） |
