## Why

目前 `byok launch codex` 啟動的是 Codex CLI，但 Codex 同時提供桌面版應用程式（透過 `codex app` 子命令啟動）。使用者需要一個獨立的 `byok launch codex-app` 指令，以 BYOK profile 啟動 Codex 桌面版，享有與 CLI 版相同的 BYOK 金鑰注入與設定覆寫機制，而不需修改父程序環境或使用者設定檔。

## What Changes

- 新增 `byok launch codex-app` 指令，作為 `byok launch` 的新目標工具 `codex-app`
- `codex-app` 目標以 `exec.LookPath("codex")` 解析同一個 `codex` 可執行檔，但在命令列參數最前方插入 `app` 子命令，再接著 `--config` 覆寫與透傳參數
- 命令列順序為：`codex app [--config ...] [<extraArgs...>]`
- 其餘行為（profile 選擇、provider 驗證、金鑰解析、環境隔離、`--yolo`、`--` 透傳）與 `byok launch codex` 完全相同
- 更新 `byok launch` 的 dispatch 邏輯，接受 `codex-app` 作為有效目標
- 更新 `byok launch` 的 usage 模板與錯誤訊息，將 `codex-app` 列入支援目標
- 更新 AGENTS.md 與 README.md 的指令文件

## Capabilities

### New Capabilities

- `byok-codex-app-launch`: 以 BYOK profile 啟動 Codex 桌面版（`codex app` 子命令），注入相同的 `--config` 覆寫與 `BYOK_CODEX_API_KEY` 環境變數

### Modified Capabilities

- `byok-launch`: dispatch 邏輯新增 `codex-app` 目標；usage 模板與錯誤訊息更新支援目標清單

## Impact

- Affected specs: `byok-codex-app-launch`（新增）、`byok-launch`（修改 dispatch 需求）
- Affected code:
  - New: `cmd/launch_codex_app.go`, `cmd/launch_codex_app_test.go`, `internal/runner/codex_app_test.go`
  - Modified: `cmd/launch.go`, `internal/runner/codex.go`, `cmd/launch_dispatch_test.go`, `AGENTS.md`, `README.md`
