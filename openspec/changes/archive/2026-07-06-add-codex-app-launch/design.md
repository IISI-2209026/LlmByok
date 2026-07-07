## Context

目前 `byok launch codex` 透過 `exec.LookPath("codex")` 解析 Codex CLI 可執行檔，以 `runner.LaunchCodex` 注入 `BYOK_CODEX_API_KEY` 環境變數與 `--config` 旗標覆寫啟動子程序。命令列順序為 `codex [--config ...] [<extraArgs...]`。

Codex 同時提供桌面版，透過 `codex app` 子命令啟動。使用者需要 `byok launch codex-app` 以相同的 BYOK 機制啟動桌面版。桌面版與 CLI 版使用同一個 `codex` 二進位檔，差異僅在命令列前方插入 `app` 子命令。

## Goals / Non-Goals

**Goals:**

- 新增 `byok launch codex-app` 指令，以 BYOK profile 啟動 Codex 桌面版
- 命令列順序為 `codex app [--config ...] [<extraArgs...]`
- 與 `byok launch codex` 共用相同的 profile 解析、provider 驗證、金鑰解析、環境隔離邏輯
- 支援 `--model`、`--profile`、`--config`、`--yolo`、`--` 透傳等所有既有旗標

**Non-Goals:**

- 不修改 `byok launch codex` 的既有行為
- 不為 Codex 桌面版引入新的環境變數或新的 `--config` 覆寫鍵
- 不偵測桌面版是否已安裝（由 `codex app` 自身處理，若未安裝會開啟安裝器）
- 不修改 `~/.codex/config.toml` 或任何使用者設定檔

## Decisions

### 以獨立 `LaunchCodexApp` 函式插入 `app` 子命令

在 `internal/runner/codex.go` 新增 `LaunchCodexApp` 函式，呼叫現有 `BuildCodexArgs` 取得環境變數與 `--config` 旗標切片，再將 `app` 插入為命令列第一個參數，接著是 `--config` 旗標，最後是透傳參數。命令列順序：`codex app [--config ...] [<extraArgs...]`。

**替代方案考量**：修改 `LaunchCodex` 加入 `subcommand` 參數。此方案會改變既有函式簽章，影響現有呼叫端與測試，增加不必要的改動範圍。採用獨立函式可保持 `LaunchCodex` 不變，降低回歸風險。

### 提取共用 profile 解析邏輯為 `resolveProfileForLaunch` 輔助函式

`runLaunchCodex` 的步驟 1–6（解析設定檔路徑、載入設定檔、選擇 profile、provider 驗證、LookPath、解析金鑰）與 `codex-app` 完全相同。將這段邏輯提取為 `cmd` 套件內的共用輔助函式 `resolveProfileForLaunch`，回傳解析後的 `*config.Profile` 與已解析的執行檔路徑。`runLaunchCodex` 與 `runLaunchCodexApp` 皆呼叫此輔助函式，避免程式碼重複。

**替代方案考量**：直接複製 `runLaunchCodex` 為 `runLaunchCodexApp`。此方案產生約 80 行幾乎相同的程式碼，違反 DRY 原則，未來維護時需同步修改兩處。採用提取共用邏輯可避免此問題。

### `codex-app` 使用同一個 `codex` 二進位檔

`byok launch codex-app` 以 `exec.LookPath("codex")` 解析同一個 `codex` 可執行檔。桌面版未安裝時由 `codex app` 自身開啟安裝器，byok 不額外偵測。

### dispatch 新增 `codex-app` 目標

在 `cmd/launch.go` 的 `switch target` 新增 `case "codex-app"`，呼叫 `runLaunchCodexApp`。usage 模板的 Targets 區段與錯誤訊息同步更新支援目標清單。

## Implementation Contract

**行為**：
- 使用者執行 `byok launch codex-app` 時，byok 以選定的 profile 解析金鑰、建置環境變數與 `--config` 覆寫，啟動 `codex app` 子程序
- 子程序命令列為 `codex app --config model="..." --config model_provider="byok" --config model_providers.byok.name="BYOK" --config model_providers.byok.base_url="..." --config model_providers.byok.env_key="BYOK_CODEX_API_KEY" [<extraArgs...]`
- 子程序環境包含 `BYOK_CODEX_API_KEY=<profile api_key>`，父程序環境不變
- `~/.codex/config.toml` 不被修改

**介面**：
- `internal/runner/codex.go` 新增 `LaunchCodexApp(profile *config.Profile, modelOverride, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer) error`
- `cmd/launch_codex_app.go` 新增 `runLaunchCodexApp(cfgPath, profileName, model string, extraArgs []string, stdout, stderr io.Writer) error`
- `cmd/launch.go` 內新增共用輔助函式 `resolveProfileForLaunch` 並在 dispatch 新增 `codex-app` case

**失敗模式**：
- 設定檔不存在 → 印出錯誤並 exit 1（與 codex 相同）
- profile 找不到 → 印出可用 profile 清單並 exit 1（與 codex 相同）
- 非 openai provider → 印出錯誤並 exit 1（與 codex 相同）
- `codex` 不在 PATH → 印出錯誤提示安裝 Codex 並 exit 1（與 codex 相同）
- 金鑰找不到 → 印出錯誤提示 `byok config set-key` 並 exit 1（與 codex 相同）
- 子程序非零結束碼 → 靜默傳遞 exit 1（與 codex 相同）

**驗收條件**：
- `go test ./... -race` 全數通過
- `byok launch codex-app --help` 顯示 `codex-app` 為支援目標
- `byok launch gemini` 錯誤訊息列出 `copilot`、`codex`、`codex-app`、`claude`
- `cmd/launch_codex_app_test.go` 驗證 `app` 子命令正確插入於 `--config` 旗標之前
- `cmd/launch_dispatch_test.go` 驗證 `codex-app` target 正確分派至 `runLaunchCodexApp`
- `internal/runner/codex_app_test.go` 驗證 `LaunchCodexApp` 建置的命令列順序為 `app` → `--config` → `extraArgs`

**範圍邊界**：
- In scope：新增 `codex-app` launch 指令、提取共用 profile 解析邏輯、更新 dispatch 與文件
- Out of scope：修改既有 `codex` launch 行為、新增 provider 類型、偵測桌面版安裝狀態

## Risks / Trade-offs

- [風險] `codex app` 子命令語法未來可能變更 → 緩解：`app` 為 Codex 文件記載的子命令，若未來變更僅需更新 `LaunchCodexApp` 一處
- [風險] 提取 `resolveProfileForLaunch` 可能影響既有 `runLaunchCodex` 行為 → 緩解：重構後以 `go test ./cmd/... -race` 確認既有 codex 測試全數通過
- [取捨] 使用獨立 `LaunchCodexApp` 而非參數化 `LaunchCodex` → 接受少量程式碼重複以保持既有函式簽章穩定
