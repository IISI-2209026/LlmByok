## 1. 底層函式

- [x] [P] 1.1 以獨立 `LaunchCodexApp` 函式插入 `app` 子命令：在 `internal/runner/codex.go` 新增 `LaunchCodexApp(profile *config.Profile, modelOverride, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer) error`，呼叫現有 `BuildCodexArgs` 取得環境變數與 `--config` 旗標，再將 `app` 插入為命令列第一個參數（`codex app [--config ...] [<extraArgs...]`）。驗證：新增 `internal/runner/codex_app_test.go`，確認 `LaunchCodexApp` 建置的命令列順序為 `app` → `--config` 覆寫 → `extraArgs`，環境變數包含 `BYOK_CODEX_API_KEY`。
- [x] [P] 1.2 提取共用 profile 解析邏輯為 `resolveProfileForLaunch` 輔助函式：從 `runLaunchCodex` 的步驟 1–6（解析設定檔路徑、載入設定檔、選擇 profile、provider 驗證、`exec.LookPath`、解析金鑰）提取至 `cmd/launch.go` 的共用函式，回傳 `*config.Profile`、已解析的執行檔路徑與錯誤。`runLaunchCodex` 改為呼叫此輔助函式。驗證：`go test ./cmd/... -race` 既有 codex 與 copilot 測試全數通過，行為不變。

## 2. cmd 層 launch codex-app 實作與 dispatch

- [x] 2.1 新增 `runLaunchCodexApp` 並實作 Launch Codex desktop app with BYOK profile：在 `cmd/launch_codex_app.go` 新增 `runLaunchCodexApp`，呼叫 `resolveProfileForLaunch`（`codex-app` 使用同一個 `codex` 二進位檔，以 `exec.LookPath("codex")` 解析），再以 `runner.LaunchCodexApp` 啟動子程序。涵蓋所有錯誤路徑：Codex-app missing config file error（設定檔不存在印提示並 exit 1）、Codex-app missing profile error（profile 找不到列出可用名稱並 exit 1）、Codex-app provider validation（非 openai 拒絕並 exit 1）、Codex executable presence check for codex-app（codex 不在 PATH 印安裝提示並 exit 1）、金鑰找不到印 `byok config set-key` 提示並 exit 1。Parent process environment unchanged for codex-app（僅子程序環境注入 `BYOK_CODEX_API_KEY`）。驗證：新增 `cmd/launch_codex_app_test.go`，涵蓋預設 profile、`--model` 覆寫、`--profile` 選擇、設定檔不存在、profile 不存在、非 openai provider、codex 不在 PATH、金鑰找不到等情境。
- [x] 2.2 dispatch 新增 `codex-app` 目標並支援 Codex-app YOLO mode flag 與 Codex-app argument passthrough via double dash：在 `cmd/launch.go` 的 `switch target` 新增 `case "codex-app"` 呼叫 `runLaunchCodexApp`；更新 usage 模板 Targets 區段加入 `codex-app` 說明；更新 Target tool selection and dispatch 的錯誤訊息列出 `copilot`、`codex`、`codex-app`、`claude`。確認 `buildExtraArgs` 對 `codex-app` target 正確附加 `--yolo`（Codex-app YOLO mode flag）與透傳參數（Codex-app argument passthrough via double dash）。更新 `Example` 區段加入 `byok launch codex-app` 範例。驗證：更新 `cmd/launch_dispatch_test.go`，確認 `codex-app` target 分派至 `runLaunchCodexApp`；不支援的 target 錯誤訊息列出四個支援目標。

## 3. 文件更新

- [x] [P] 3.1 更新 `AGENTS.md`：在 CLI 介面段落與套件職責表加入 `launch codex-app` 指令說明（啟動 Codex 桌面版，程式邏輯與 `launch codex` 相同但插入 `app` 子命令）。驗證：`AGENTS.md` 文件提及 `codex-app` 目標與 `LaunchCodexApp` 函式。
- [x] [P] 3.2 更新 `README.md`：在 usage 範例與支援目標清單加入 `byok launch codex-app`。驗證：`README.md` 包含 `byok launch codex-app` 指令範例。

## 4. 全套測試驗證

- [x] 4.1 執行 `go test ./... -race` 確認全數通過：所有新增測試（`internal/runner/codex_app_test.go`、`cmd/launch_codex_app_test.go`、更新後的 `cmd/launch_dispatch_test.go`）與既有測試皆無失敗且無資料競爭。驗證：命令輸出零失敗、零 race 告警。
