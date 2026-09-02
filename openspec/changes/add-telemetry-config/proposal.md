## Why

byok 使用者在企業環境中需要監控 AI coding agent 的使用量、成本與工具活動。目前 byok 只負責注入 BYOK provider 設定，但不支援 OpenTelemetry 遙測設定的統一注入。各 target（Copilot CLI、Codex、Claude、Pi）皆已各自支援 OTEL，但設定方式各異（環境變數、config 旗標、plugin），使用者需要一個統一的設定入口。

## What Changes

- 設定檔新增頂層 `telemetry` 區段，承載 enabled、service_name、headers、grpc endpoint、http endpoint + protocol
- `byok launch <target>` 時依據 telemetry 設定與各 target 原生支援方式，自動注入對應的 OTEL 環境變數或 `--config` 旗標
- 各 target 注入邏輯：
  - Copilot CLI：環境變數（`COPILOT_OTEL_ENABLED`、`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_EXPORTER_OTLP_PROTOCOL`、`OTEL_EXPORTER_OTLP_HEADERS`、`OTEL_SERVICE_NAME`）
  - Codex/Codex App：`--config otel.*` 旗標
  - Claude：環境變數（`CLAUDE_CODE_ENABLE_TELEMETRY`、`OTEL_METRICS_EXPORTER`、`OTEL_LOGS_EXPORTER`、`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_EXPORTER_OTLP_PROTOCOL`、`OTEL_EXPORTER_OTLP_HEADERS`、`OTEL_SERVICE_NAME`）
  - Pi：環境變數（`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_SERVICE_NAME`、`PI_OTEL_ENABLED`）
- Service name 規則：未設定不注入；設定時組合為 `<custom-name>-<agent-name>`
- Protocol 選擇：同時支援 gRPC 與 HTTP，各 target 依原生能力選用（支援 gRPC 的優先 gRPC，僅支援 HTTP 的用 HTTP）
- README 新增 OpenTelemetry 章節，列出各 target 官方文件來源與設定對應方式

## Non-Goals (optional)

- 不新增 `byok config` 子指令來管理 telemetry（首版直接編輯 config.yaml）
- 不支援 per-signal endpoint（如 traces/metrics 分開）
- 不支援 mTLS 證書路徑（進階需求，留待後續）
- 不支援 file exporter
- 不實作 byok 自身的 OTEL instrumentation

## Capabilities

### New Capabilities

- `byok-telemetry-config`: 設定檔 telemetry 區段的載入、驗證與各 target 注入邏輯

### Modified Capabilities

- `byok-launch`: launch 流程新增 telemetry 注入步驟
- `byok-launch-dry-run`: dry-run 輸出需包含 telemetry 相關環境變數或旗標

## Impact

- Affected specs: `byok-telemetry-config`（新增）、`byok-launch`（修改）、`byok-launch-dry-run`（修改）
- Affected code:
  - New: `internal/config/telemetry.go`
  - Modified: `internal/config/config.go`、`internal/runner/runner.go`、`internal/runner/codex.go`、`internal/runner/claude.go`、`internal/runner/pi.go`、`cmd/launch.go`、`README.md`
  - Removed: 無
