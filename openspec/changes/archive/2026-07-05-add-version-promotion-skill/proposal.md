## Why

`byok` 的版本號目前以 `internal/version/version.go` 的 `Version` 字面值為唯一來源，由人手編輯，沒有自動晉升機制。Release workflow 對 develop 與 main 使用同一個 base 版號，develop 預發布 tag 固定為 `v<base>-dev`，重複推送 develop 會撞 tag 導致 release 失敗。此外沒有「發布後跳下一版」的標準流程，使得 develop 下一輪預發布與 main 穩定版號缺乏一致的晉升節奏。需要一個可重複執行的版號晉升機制，並以 skill 形式提供，讓 AI agent 與貢獻者能以單一指令完成 bump + push 到 develop。

## What Changes

- 正式化版號模型：`internal/version/version.go` 的 `Version` 字面值為 canonical base 版號（semver，無 prefix），起始值設為 `0.1.0`。
- develop 分支預發布：tag 與二進位版號改為 `<base>-dev.<run_number>`（以 GitHub Actions `run_number` 確保每次推送唯一，不再撞 tag）；main 分支穩定發布維持 `<base>`。
- 新增 Copilot CLI skill `.github/skills/byok-bump-version/SKILL.md`：讀取現有 base 版號、依指定等級（預設 patch，可選 minor/major）計算下一版、編輯 `internal/version/version.go`、commit `chore: bump version to <next>` 並 push 到 `origin develop`。
- 在 `AGENTS.md` 新增「版本號機制」區塊，說明 canonical 來源、develop/main 版號格式、晉升流程（merge develop→main 觸發穩定發布 → 於 develop 執行 bump skill 跳下一版），並指向 bump skill。
- 在 `README.md` 新增「版本號與發布流程」段落，說明 develop 預發布（`<base>-dev.<run_number>`）、main 穩定發布（`<base>`）、以及如何使用 `byok-bump-version` skill 晉升版號。

## Non-Goals (optional)

本變更不會：

- 引入獨立的版本檔（如 VERSION 檔或 Git tag 作為唯一來源）；版號唯一來源維持 `internal/version/version.go`。
- 變更 Release workflow 的平台矩陣、產物命名規則或 Pushover 通知邏輯（僅修改版號/ tag 推導）。
- 自動化 develop→main 的合併/PR 流程（合併仍由人或 agent 以 git 完成；skill 僅負責 bump + push 到 develop）。
- 變更 `internal/version` 套件的 ldflags 注入機制。

## Capabilities

### New Capabilities

- `byok-version-bump`: 以 Copilot CLI skill 形式提供版號晉升機制——讀取 `internal/version/version.go` canonical base、依等級計算下一版、編輯檔案、commit 並 push 到 develop；涵蓋 patch/minor/major 等級、未來於 main 分支執行的防呆、錯誤處理。

### Modified Capabilities

- `byok-version`: 正式化 `internal/version/version.go` `Version` 字面值為 canonical base 版號（semver、無 prefix、起始 `0.1.0`），並定義 develop 與 main 的二進位版號格式。
- `byok-release`: develop 預發布 tag 與版號改用 `<base>-dev.<run_number>` 以確保每次推送唯一；main 維持 `v<base>` 穩定發布。

## Impact

- Affected specs:
  - New: `byok-version-bump`
  - Modified: `byok-version`, `byok-release`
- Affected code:
  - New:
    - `.github/skills/byok-bump-version/SKILL.md`（版號晉升 skill）
  - Modified:
    - `internal/version/version.go`（`Version` 字面值由 `dev` 改為 `0.1.0` 作為起始 base）
    - `.github/workflows/release.yml`（develop 版號/tag 改為 `<base>-dev.<run_number>`，傳入 `github.run_number`）
    - `AGENTS.md`（新增「版本號機制」區塊）
    - `README.md`（新增「版本號與發布流程」段落）
- Affected systems: Release workflow 的 tag/版號推導邏輯變更；不影響外部系統。
