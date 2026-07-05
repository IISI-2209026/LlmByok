## Why

byok CLI 目前沒有版本號管理機制，也沒有自動化打包與發布流程。每次開發完成後無法追蹤版本、無法產生可分发的執行檔，使用者只能自行從原始碼建置。需要導入 SemVer 版本規範與 GitHub Actions 自動發布流程，讓合併回 main 分支時自動打包多平台執行檔並發布至 GitHub Release。

## What Changes

- 新增版本號變數（`internal/version/version.go`），使用 Go 標準 ldflags 注入機制在建置時寫入版本號
- 新增 `byok version` 子指令顯示當前版本號
- 新增 GitHub Action workflow（`.github/workflows/release.yml`），於 main 分支 push 時自動建置 Windows/Linux/macOS 執行檔並建立 GitHub Release
- Makefile build target 加入 ldflags 注入版本號
- 每輪開發合併回 main 時遞增小版本號（patch version），並以 git tag 標記

## Non-Goals

- 不支援 pre-release / beta 版本號格式（alpha/beta/rc），MVP 僅用 `MAJOR.MINOR.PATCH`
- 不做套件管理器發布（Homebrew、Scoop、APT 等），僅 GitHub Release 附件
- 不做程式碼簽章（code signing），MVP 階段不處理
- 不自動計算版本號遞增邏輯，由開發者手動更新版本號字串

## Capabilities

### New Capabilities

- `byok-version`: 版本號管理與 `byok version` 指令顯示版本
- `byok-release`: GitHub Actions 自動打包多平台執行檔並發布至 GitHub Release

### Modified Capabilities

(none)

## Impact

- Affected specs: byok-version, byok-release
- Affected code:
  - New: internal/version/version.go, internal/version/version_test.go, cmd/version.go, cmd/version_test.go, .github/workflows/release.yml
  - Modified: main.go（註冊 version 子指令）, Makefile（ldflags 注入）, go.mod（不新增依賴）
