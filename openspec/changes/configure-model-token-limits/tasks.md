## 1. 設定資料模型與相容遷移

- [x] 1.1 依「模型項目採用相容的 scalar-or-mapping YAML」先在 `internal/config/config_test.go` 新增失敗測試，再實作 Model/ModelLimits、`Config file location` 的 scalar/mapping/legacy decode 與 canonical mapping encode，以及 `Model token configuration validation` 的空名、重名、零值和負值錯誤；以 `go test ./internal/config` 驗證所有格式與邊界。
- [x] 1.2 調整 `cmd/set_models.go`、`cmd/config.go` 與 `internal/config/models.go`，使 `Set candidate models for a profile` 以名稱保留同名模型限制，並符合 `Model name views omit token metadata`；先擴充 `cmd/set_models_test.go`、`cmd/config_test.go`、`internal/config/models_test.go`，再以 `go test ./cmd ./internal/config` 驗證順序、保留、刪除、list 與互動選單。

## 2. 共用 token 解析與相容行為

- [x] 2.1 依「Token 限制逐欄位解析並保留來源」在 `cmd/launch.go` 建立 normal launch 與 dry-run 共用的 resolved model limits，實作 `Resolve model token limits per field` 與 `Validate launch token flags`，包含陌生 `--model`、每欄位 fallback、正整數和 API key/executable 前置失敗；先新增 table-driven launch tests，再以 `go test ./cmd -run 'Token|Model'` 驗證。
- [x] 2.2 實作 `Unsupported token limit is warning-only` 的 stderr helper 與來源標示，確保 ignored value 不進 runner、unset 不提示且 exit status 不因 warning 改變；以 normal/dry-run 的 buffer 測試驗證 target、參數、值、來源及 stdout/stderr 分流。

## 3. Target runner 映射

- [x] [P] 3.1 依「Target adapter 只注入官方支援的參數」在 `internal/runner/runner.go` 移除固定 Copilot token 值並實作 `Copilot token limit mapping`，包含直接 prompt 映射、output 映射、unset omission、環境去重與不做算術；先更新 `internal/runner/runner_test.go`，再以 `go test ./internal/runner -run Copilot` 驗證。
- [x] [P] 3.2 在 `internal/runner/codex.go` 與 Codex launch dispatch 實作 `Codex model token limit mapping`、`Codex App model token limit mapping`，保持 Codex App 的 `app` 首參數和既有參數順序，並由共用 warning contract 忽略 output；以 `internal/runner/codex_test.go`、`internal/runner/codex_app_test.go` 與 cmd dispatch tests 驗證 context、unset、warning 及 argument order。
- [x] [P] 3.3 在 `internal/runner/claude.go` 實作 `Launch Claude with BYOK profile` 的模型字串原樣傳遞及 `Claude model token limit mapping`，移除自動 `[1m]` 並注入兩個可選環境變數；先更新 runner/cmd Claude tests，再以 `go test ./internal/runner ./cmd -run Claude` 驗證普通名稱、顯式 suffix、部分設定、unset 與父環境隔離。
- [x] [P] 3.4 依「Pi 以暫存 model override 並從輸出量衍生 headroom」在 `internal/runner/pi.go` 實作 `Pi model token limit overrides` 與 `Pi output limit reserves response headroom`，只在有效時產生 modelOverrides/settings、使用 0600、保留 provider-only unset 行為並清理整個暫存目錄；以 JSON 結構、檔案權限、成功/失敗清理的 Pi tests 驗證。

## 4. Dry-run parity

- [x] 4.1 依「Normal launch 與 dry-run 共用有效限制契約」擴充 `cmd/launch_dry_run.go` 的 `Dry-run renders effective token limits`，使 Copilot、Codex、Codex App、Claude 的 Windows/POSIX 輸出與 normal mapping 一致，unsupported warning 僅在 stderr，secret 仍為 `***`；以 `go test ./cmd -run DryRun` 的 parity 與 secret-negative assertions 驗證。
- [x] 4.2 實作 `Pi dry-run renders token configuration lifecycle`，輸出 masked modelOverrides、可選 settings headroom、Pi invocation 與平台原生 cleanup，context-only 時不產生 settings；以 Windows/POSIX dry-run tests 解析 stdout/stderr 並確認沒有真實 API key 或暫存副作用。

## 5. 使用者與維護文件

- [x] [P] 5.1 依「Context 容量與壓縮策略保持分離」更新 `README.md`，完成 `Documentation explains model token limits` 的完整 YAML、legacy/降版說明、旗標、逐欄位優先序、五 target 支援矩陣、Copilot prompt 語意、Codex warning、Pi headroom、容量限制與非壓縮門檻聲明；以逐項內容檢查及 README 命令/YAML 範例人工驗證。
- [x] [P] 5.2 更新 `AGENTS.md` 非 Spectra 管理區塊以完成 `AGENTS.md records token-limit contracts`，記錄 config/CLI/runner 注入與 warning 行為且不改動 Spectra 區塊；以 `git diff -- AGENTS.md` 確認管理區塊位元組不變並人工核對維護規則要求。

## 6. 整合驗證

- [x] 6.1 對完整 Implementation Contract 執行 `gofmt`、`go test ./...`、`go test -cover ./cmd ./internal/config ./internal/runner` 與 `go vet ./...`，確認舊 scalar config、五 target normal/dry-run、unset omission、warning channel、secret masking、Pi 清理及既有 telemetry/effort/yolo/passthrough regression 全部通過，並記錄 coverage 結果供審查。
