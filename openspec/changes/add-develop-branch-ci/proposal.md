## Why

目前 release workflow 只在推送至 main 時觸發穩定版發布。需要在 develop 分支上也能執行 CI 並產出帶有 `-dev` 後綴的測試版本，讓開發者能在 develop 分支驗證整合結果，同時不污染穩定版 Release。

## What Changes

- 修改 `.github/workflows/release.yml`，讓 push 至 develop 分支也觸發 workflow
- 在 workflow 中依分支判斷版本號後綴：develop 分支附加 `-dev`（例如 `0.1.0-dev`），main 分支維持原樣
- develop 分支產出的 GitHub Release 標記為 prerelease
- Release tag 格式：main 為 `v<version>`，develop 為 `v<version>-dev`
- feature 分支不觸發 release workflow

## Non-Goals

- 不新增獨立的 dev-build workflow 檔案（採單一 workflow + branch 條件判斷）
- 不修改 `internal/version/version.go` 的版本號來源（版本後綴由 workflow 動態附加）
- 不處理 feature 分支的 CI build（feature 分支透過 PR 驗證即可）
- 不實作 GitFlow 完整流程（如 release 分支、hotfix 分支）

## Capabilities

### New Capabilities

（無）

### Modified Capabilities

- `byok-release`: release workflow 觸發條件由僅 main 改為 main 與 develop，並依分支動態調整版本號後綴與 prerelease 標記

> 注意：`byok-release` 規格目前由未封存的 `add-version-and-release` 變更引入。本變更必須在 `add-version-and-release` 封存後才能封存，以確保主規格已存在對應的 requirement。

## Impact

- Affected specs: byok-release (modified)
- Affected code:
  - Modified: `.github/workflows/release.yml`
  - New: 無
  - Removed: 無