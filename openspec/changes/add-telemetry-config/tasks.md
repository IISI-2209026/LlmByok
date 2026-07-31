## 1. 設定檔結構

- [ ] [P] 1.1 實作 Telemetry configuration structure（設計決策：設定檔結構：頂層 telemetry 區段，分離 gRPC 與 HTTP；Telemetry 資料結構設計）：建立 `internal/config/telemetry.go`，定義 `Telemetry`、`TelemetryGRPC`、`TelemetryHTTP` struct，包含 YAML tag。`http.protocol` 驗證僅接受 `protobuf` 或 `json`，其他值回傳描述性錯誤。驗證方式：`internal/config/telemetry_test.go` 中單元測試涵蓋 valid config、invalid protocol、nil telemetry 三種情境。
- [ ] [P] 1.2 整合 Telemetry 至 Config struct：修改 `internal/config/config.go`，在 `Config` struct 新增 `Telemetry *Telemetry` 欄位（yaml tag `telemetry,omitempty`）。Load 函式成功解析後呼叫 telemetry 驗證。驗證方式：`internal/config/config_test.go` 中測試含 telemetry 與不含 telemetry 的 YAML 均可正確載入。

## 2. Service Name 組合

- [ ] 2.1 實作 Service name composition（設計決策：Service name 組合規則）：在 `internal/config/telemetry.go` 新增 `ComposeServiceName(serviceName, target string) string` 函式，target 接受 `copilot`、`codex`、`codex-app`、`claude`、`pi`，回傳組合後的名稱（如 `my-team-github-copilot`）。空 serviceName 回傳空字串。驗證方式：`internal/config/telemetry_test.go` 中 table-driven 測試涵蓋所有 target 與空值情境。

## 3. Runner 注入

- [ ] [P] 3.1 實作 Copilot CLI telemetry injection（設計決策：注入層級：runner 層負責轉譯；Protocol 映射表）：修改 `internal/runner/runner.go` 的 `BuildEnv` 函式，新增 `telemetry *config.Telemetry` 參數。當 telemetry enabled 且 HTTP endpoint 存在時，注入 `COPILOT_OTEL_ENABLED`、`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_EXPORTER_OTLP_PROTOCOL`、`OTEL_EXPORTER_OTLP_HEADERS`、`OTEL_SERVICE_NAME`。驗證方式：`internal/runner/runner_test.go` 測試含 telemetry 與 nil telemetry 的 env 輸出。
- [ ] [P] 3.2 實作 Codex telemetry injection（設計決策：Target 選用優先順序：支援 gRPC 者優先 gRPC；Protocol 映射表；注入層級：runner 層負責轉譯）：修改 `internal/runner/codex.go` 的 `BuildCodexArgs` 函式，新增 `telemetry *config.Telemetry` 參數。依 Target endpoint selection 規則（gRPC 優先），注入 `--config otel.*` 旗標與 `OTEL_SERVICE_NAME` 環境變數。驗證方式：`internal/runner/codex_test.go` 測試 gRPC endpoint、HTTP fallback、nil telemetry 三種情境。
- [ ] [P] 3.3 實作 Claude telemetry injection（設計決策：Target 選用優先順序：支援 gRPC 者優先 gRPC；Protocol 映射表；注入層級：runner 層負責轉譯）：修改 `internal/runner/claude.go` 的 `BuildClaudeEnv` 函式，新增 `telemetry *config.Telemetry` 參數。注入 `CLAUDE_CODE_ENABLE_TELEMETRY`、`OTEL_METRICS_EXPORTER`、`OTEL_LOGS_EXPORTER`、`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_EXPORTER_OTLP_PROTOCOL`、`OTEL_EXPORTER_OTLP_HEADERS`、`OTEL_SERVICE_NAME`。驗證方式：`internal/runner/claude_test.go` 測試 gRPC 與 HTTP 路徑。
- [ ] [P] 3.4 實作 Pi telemetry injection（設計決策：Target 選用優先順序：支援 gRPC 者優先 gRPC；注入層級：runner 層負責轉譯）：修改 `internal/runner/pi.go` 的 `BuildPiEnv` 函式，新增 `telemetry *config.Telemetry` 參數。注入 `PI_OTEL_ENABLED`、`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_SERVICE_NAME`（不注入 headers）。驗證方式：`internal/runner/pi_test.go` 測試 HTTP endpoint 注入與僅 gRPC 時跳過。

## 4. 呼叫端整合

- [ ] 4.1 整合 Launch 呼叫端傳遞 telemetry：修改 `cmd/launch.go` 及各 target launch 檔案（`cmd/launch_*.go`），從載入的 Config 取出 Telemetry 並傳入對應 runner 函式。Target tool selection and dispatch 流程新增 telemetry 讀取步驟。驗證方式：`go build ./...` 編譯通過，`go test ./cmd/...` 通過。
- [ ] 4.2 整合 Render platform-specific equivalent commands（dry-run）：修改 `cmd/launch_dry_run.go`，當 telemetry 啟用時在 dry-run 輸出中加入對應的環境變數或旗標，headers 值以 `***` mask。Telemetry disabled means no injection 時 dry-run 不輸出 OTEL 項目。驗證方式：`cmd/launch_dry_run_test.go` 新增測試案例驗證含 telemetry 與不含 telemetry 的 dry-run 輸出。

## 5. 文件

- [ ] [P] 5.1 README 新增 OpenTelemetry 章節：在 `README.md` 新增 `## OpenTelemetry` 段落，說明設定格式、各 target 對應方式、官方文件來源連結（Copilot: `copilot help monitoring`、Codex: https://github.com/openai/codex codex-rs/otel/README.md、Claude: https://docs.anthropic.com/en/docs/claude-code/monitoring-usage、Pi: https://github.com/mprokopov/pi-otel-telemetry）。驗證方式：人工 review README 內容完整性與連結正確性。
- [ ] [P] 5.2 更新 AGENTS.md：在設定檔格式段落新增 telemetry 區段說明，記錄 telemetry 欄位與注入行為。驗證方式：人工 review AGENTS.md 與實際行為一致。

## 6. 整合驗證

- [ ] 6.1 全套件測試通過：執行 `go test ./...` 確認所有測試通過。驗證方式：exit code 0，無 FAIL 輸出。
