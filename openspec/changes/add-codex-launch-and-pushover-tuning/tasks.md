## 1. 目標工具分派

- [x] 1.1 修改 `cmd/launch.go` 的 `newLaunchCmd`，將第一個位置參數正式化為目標工具選擇器：`copilot` 分派至既有 `runLaunchCopilot`，`codex` 分派至新 `runLaunchCodex`；省略目標時印出「必須指定目標工具」、非支援目標時印出支援清單，兩者皆 exit 1（對應設計 Decision: 目標工具分派置於 cmd 層，runner 保持工具特定建置函式）。驗證：更新 `cmd/launch_test.go` 新增「Target tool selection and dispatch」情境測試（省略、`copilot`、`codex`、不支援目標），`go test ./cmd/...` 通過。
- [x] 1.2 確認 `byok launch copilot` 既有測試全數通過且無行為退化。驗證：`go test ./cmd/... ./internal/runner/...` 綠燈，既有 copilot 測試未更動即通過。

## 2. Codex BYOK 注入核心

- [x] 2.1 新增 `internal/runner/codex.go`，實作 `BuildCodexArgs(profile *config.Profile, modelOverride string) (env []string, configArgs []string)`：`env` 以現有環境為起點過濾掉既存 `BYOK_CODEX_API_KEY` 後附加 `BYOK_CODEX_API_KEY=<profile.api_key>`；`configArgs` 回傳 `--config` 旗標切片，設定 `model`、`model_provider="byok"`、`model_providers.byok.base_url`、`model_providers.byok.env_key="BYOK_CODEX_API_KEY"`，`model` 在 `modelOverride` 非空時使用之否則用 `profile.DefaultModel`（對應設計 Decision: Codex BYOK 採環境變數承載 API key + `--config` 旗標覆寫連線設定）。驗證：新增 `internal/runner/codex_test.go` 表列斷言 `BYOK_CODEX_API_KEY` 值與各 `--config` 鍵值格式（含 TOML 引號），`go test ./internal/runner/...` 通過。
- [x] 2.2 在 `internal/runner/codex.go` 新增 `LaunchCodex(profile, modelOverride, exePath string, extraArgs []string, stdin, stdout, stderr) error`：以 `BuildCodexArgs` 組裝 `codex [<--config ...>] [<--yolo>] [<passthrough...>]` 並以子程序環境啟動；父程序環境不變。驗證：`codex_test.go` 新增 `TestLaunchCodex`，以 stub 驗證命令列順序（`--config` 在前、`--yolo` 在中、透傳在後）與環境變數隔離。

## 3. Codex 啟動指令流程

- [x] 3.1 新增 `cmd/launch_codex.go` 的 `runLaunchCodex`，與 `runLaunchCopilot` 對等：解析設定檔路徑、載入設定檔（不存在則印「找不到設定檔」並提示 `byok config add`、exit 1，涵蓋 Codex missing config file error）、選擇 profile（未指定用 `default_profile`，未設印提示、exit 1）、profile 不存在印出可用清單並 exit 1（Codex missing profile error）、provider 驗證僅接受 `openai`（空字串回退 `openai`，非 openai 印錯並 exit 1，涵蓋 Codex provider validation）。驗證：`cmd/launch_codex_test.go` 對各錯誤路徑斷言 stderr 訊息與 exit code。
- [x] 3.2 在 `runLaunchCodex` 中以 `exec.LookPath("codex")` 解析可執行檔，不存在時印出安裝 Codex CLI 提示並 exit 1（Codex executable presence check）；存在時呼叫 `runner.LaunchCodex`，非零結束靜默傳遞 exit code。驗證：`cmd/launch_codex_test.go` 注入假的 PATH 環境測試缺失與成功解析路徑。
- [x] 3.3 在 codex 路徑套用 `-y`/`--yolo` 與 `--` 透傳：`-y` 時附加 `--yolo`（Codex YOLO mode flag），`--` 之後參數原樣附加在後（Codex argument passthrough via double dash）；順序與 copilot 一致（yolo 在前、透傳在後）（對應設計 Decision: `-y`/`--yolo` 對 codex 透傳為 `--yolo` 旗標）。驗證：`cmd/launch_codex_test.go` 以 stub 斷言命令列含 `--yolo` 與透傳順序。

## 4. Codex 整合測試

- [x] 4.1 在 `internal/runner` 新增 codex 整合測試（仿照 `launch_integration_test.go` 的 stub 架構），驗證「Launch Codex with BYOK profile」情境：以真實 profile 透過 `LaunchCodex` 啟動 stub，stub 印出其環境與 argv，測試斷言 `BYOK_CODEX_API_KEY` 與 `--config` 覆寫內容正確，且父程序環境與 `~/.codex/config.toml` 不被修改（Parent process environment unchanged for codex）。驗證：`go test ./internal/runner/... -run Codex` 通過。

## 5. Pushover 通知調整

- [x] 5.1 修改 `.github/workflows/release.yml` 的 `notify` job：以 `needs.build.result` 與 `needs.release.result` 推導整體狀態（兩者皆 `success` 才為 `success`，否則 `failure`，涵蓋 Notification status derivation），並以 ternary 依狀態選取 `priority`（成功 `'0'`、失敗 `'1'`）與 `sound`（成功中性/正向音、失敗警示音，涵蓋 Pushover notification priority and sound by result）（對應設計 Decision: Pushover 通知以 job/needs 結果決定 priority 與 sound）。驗證：YAML 靜態檢視確認 `priority`/`sound` 隨狀態分支；以 `act` 或 workflow run 歷史確認成功與失敗各發出對應優先權。
- [x] 5.2 修改 `.github/workflows/pr-test.yml` 的 Pushover 步驟：以 `job.status` 推導狀態，成功用 `priority: '0'` 與中性/正向音、失敗用 `priority: '1'` 與警示音。驗證：YAML 靜態檢視確認 `priority`/`sound` 隨 `job.status` 分支。

## 6. README 與官方文件

- [ ] 6.1 更新 `README.md`：新增 `byok launch codex` 操作說明（旗標表格與範例，對齊 copilot 段落）、新增「Codex BYOK 運作原理」段落說明 `BYOK_CODEX_API_KEY` + `--config` 覆寫且不寫入 `~/.codex/config.toml`。驗證：README 內容審查確認段落完整、範例可複製執行。
- [ ] 6.2 在 `README.md` 補上官方文件連結區段：Copilot CLI BYOK 官方文件（https://docs.github.com/zh/copilot/how-tos/copilot-cli/customize-copilot/use-byok-models ）與 Codex CLI BYOK 官方文件（https://developers.openai.com/codex/config-advanced#custom-model-providers 與 https://developers.openai.com/codex/auth#alternative-model-providers ）。驗證：連結可點擊且指向正確章節。

## 7. AGENTS.md 架構與規範文件

- [ ] 7.1 在 `AGENTS.md` 既有 Spectra 區塊（`<!-- SPECTRA:END -->`）之後新增「專案架構」區塊，內容取自現有原始碼事實：Go 1.26+ cobra CLI、模組路徑 `github.com/IISI-2209026/LlmByok`、入口 `main.go`、`cmd`（cobra 指令與目標分派）/`internal/config`（YAML profile 載入儲存）/`internal/runner`（BYOK 環境建置與子程序啟動）/`internal/version`（版本嵌入）套件職責、設定檔位置 `~/.byok/config.yaml`（實作 AGENTS.md documents project architecture）（對應設計 Decision: AGENTS.md 作為架構與規範的權威文件並隨變更同步）。驗證：`AGENTS.md` 含此區塊且所述套件/路徑與原始碼一致。
- [ ] 7.2 在「專案架構」之後新增「開發規範」區塊：BYOK 注入僅作用於子程序（父程序與 shell 環境不變）、不寫入使用者設定檔（`~/.byok/config.yaml`、`~/.codex/config.toml`）、profile 解析錯誤印訊息並 exit 1、預設 provider 為 `openai`、測試以 `go test ./... -race` 執行（實作 AGENTS.md documents development conventions）。驗證：`AGENTS.md` 含此區塊且各條與既有行為一致。
- [ ] 7.3 在「開發規範」之後新增明確維護規則：任何改變套件結構、BYOK 注入機制、設定檔格式、CLI 介面或已記錄開發規範的變更，須在相同變更內更新 `AGENTS.md` 對應段落；Spectra 區塊由 CLI 管理不得手動編輯（實作 AGENTS.md maintenance rule）。驗證：`AGENTS.md` 含維護規則文字且明確標示 Spectra 區塊不得手改。

## 8. 整體驗證

- [ ] 8.1 執行 `go vet ./...` 與 `go test ./... -race -coverprofile=coverage.out`，確認全部通過且無退化。驗證：指令結束碼為 0，coverage 產出。
