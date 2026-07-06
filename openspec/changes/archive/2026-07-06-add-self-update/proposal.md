## Why

`byok` 目前只能從原始碼建置安裝，沒有「下載預建二進位」的說明，也沒有任何更新機制。使用者無從得知自己執行的版本是否過舊，每次升級都得手動重裝。隨著 develop 預發布與 main 穩定發布流程已透過 Release workflow 產生多平台二進位資產，缺一個能對照 GitHub Releases 自動檢查並自我更新的功能，讓 dev 版只升 dev、正式版只升正式版。

## What Changes

- 新增 `byok update` 子指令：依當前版本所屬 channel（含 `-dev.` 為 dev channel，否則 stable channel）查詢 GitHub Releases，下載對應平台資產並自我替換執行檔；提供 `--channel prerelease|release` 旗標覆寫自動 channel 判定，讓使用者明確選擇要更新到預發布或正式版本。
- 新增啟動版本檢查：執行非 `launch`／非 `update` 指令後，查詢 GitHub Releases 並於有新版本時在 stderr 印出提示與更新指令；`launch copilot|codex` 與 `byok update` 跳過此檢查。每次執行皆檢查（every-run），網路失敗靜默不影響指令結果。
- 新增 README.md「安裝」段落，提供兩種安裝方式：`go install github.com/IISI-2209026/LlmByok@latest` 與自 GitHub Releases 下載對應平台預建二進位；並補 `byok update`（含 `--channel` 與 `--check`）使用說明。
- 新增 `internal/updater` 套件：實作 channel 判定、release 查詢、資產選擇、下載與跨平台執行檔自我替換（原子替換：Windows 以 MoveFileEx、Unix 以 rename）。
- 修改 Release workflow：於建立 GitHub Release 前，自上一個 release tag 至當前 HEAD 的 commit 範圍產生分類 changelog（新增功能 `feat:`、優化功能 `refactor:`/`perf:`、修復功能 `fix:`），作為 release body 取代 GitHub 自動產生的 release notes。

## Non-Goals

- 不實作背景常駐或排程自動更新（僅使用者主動執行 `byok update` 或執行其他指令時的啟動檢查）。
- 不下載/安裝 codex 或 copilot 目標工具本身，僅更新 `byok` 自身。
- 不切換 channel（預設）：`byok update` 預設依當前版本自動判定 channel、不跨 channel；使用者須以 `--channel prerelease|release` 明確覆寫才會更新到另一 channel。
- 不處理套件管理器（Homebrew/winget/scoop）安裝路徑。
- 啟動檢查不快取查詢結果（every-run），但以短 timeout 與失敗靜默確保不阻斷。

## Capabilities

### New Capabilities

- `byok-self-update`: `byok` 自我更新能力——依當前版本 channel（或 `--channel` 覆寫）查詢 GitHub Releases、下載對應平台資產、原子替換執行檔，以及非 launch/update 指令後的啟動版本檢查提示。

### Modified Capabilities

- `byok-setup`: README 安裝說明新增「下載 GitHub Releases 預建二進位」安裝路徑，與既有 `go install` 並列。
- `byok-release`: Release workflow 新增分類 changelog 產生步驟，以 commit 範圍自上一個 release tag 至 HEAD 的新增/優化/修復功能分類撰寫 release body。

## Impact

- Affected specs: `byok-self-update`（new）、`byok-setup`（modified）、`byok-release`（modified）
- Affected code:
  - New: `internal/updater/updater.go`, `internal/updater/updater_test.go`, `cmd/update.go`, `cmd/update_test.go`
  - Modified: `cmd/root.go`（註冊 update 子指令、加入 PostRun 版本檢查）、`internal/version/version.go`（新增 channel 判定輔助）、`README.md`（安裝段落與 update 說明）、`.github/workflows/release.yml`（changelog 產生步驟取代 `generate_release_notes`）
- Relies on existing release asset naming from `byok-release` spec: `byok-<version>-<os>-<arch>.<ext>`（zip for Windows, tar.gz for Linux/macOS）。
- New runtime dependency：GitHub REST API（`GET /repos/IISI-2209026/LlmByok/releases`）與 release 資產下載；無新 Go 第三方相依（以 `net/http` + `archive/zip` + `compress/gzip` + `archive/tar` 標準庫實作）。
