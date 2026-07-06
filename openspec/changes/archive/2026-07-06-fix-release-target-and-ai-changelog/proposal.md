## Why

Release workflow 目前的 GitHub Release tag 一律落在 main 分支 commit 上，即使由 develop push 觸發亦然，導致 develop 預發布的 tag 指向錯誤的 commit。此外，目前 changelog 僅以 `git log` commit subject 加 `grep` 機械式分類，缺乏對程式碼與規格文件變更的語意理解；使用者希望改用 GitHub Models 免費額度，讀取 PR 變更的程式碼與規格文件後，針對 byok 程式的變更撰寫格式化 changelog。

## What Changes

- 修正 release tag commitish：在 `softprops/action-gh-release` 步驟明確設定 `target_commitish: ${{ github.sha }}`，使 tag 落在觸發該次 workflow 的分支 commit 上（develop push → develop commit，main push → main commit），不再預設指向 main。
- 改用 AI 產生 changelog：在 release job 新增一步，透過 GitHub Models（`@github/models` 或 GitHub Models API 端點，使用內建免費額度）讀取本次 release 範圍內變更的 byok 程式碼與規格文件（`cmd/`、`internal/`、`openspec/specs/` 等差異），產生分類為「新增功能」「優化功能」「修復功能」的 Markdown changelog，作為 release body。
- 保留 fallback：當 AI 模型呼叫失敗、回傳空白或未設定模型存取設定時，回退至現有以 commit history 分類的 changelog，確保 release 永不因 changelog 失敗而中斷。
- 分支適用：tag commitish 修正與 AI changelog 同時適用於 stable（main）與 prerelease（develop）workflow。

## Non-Goals

- 不改變版號機制（base 來源、develop `-dev.<run_number>`、main `<base>` 的晉升流程維持不變）。
- 不改變 build matrix、artifact 命名、Pushover 通知 job 的行為。
- AI changelog 僅讀取本次 release 的程式碼與規格差異，不對 release 以外的歷史 commit 重新生成過往 changelog。
- 不引入需要付費的外部 LLM 服務；僅使用 GitHub Models 免費額度。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `byok-release`: tag commitish 需求（tag 須落在觸發分支的 commit）與 changelog 產生需求（由 commit-history grep 改為 GitHub Models AI 生成，並保留 fallback）變更。

## Impact

- Affected specs:
  - Modified: `openspec/specs/byok-release/spec.md`（tag commitish 與 changelog 兩項需求）
- Affected code:
  - Modified: `.github/workflows/release.yml`
