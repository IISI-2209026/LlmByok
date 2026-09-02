## Context

byok 是以 Go 建構的 CLI 工具，透過 BYOK（Bring Your Own Key）profile 暫時啟動 AI coding agent 子程序（Copilot CLI、Codex、Claude、Pi），僅注入環境變數或 config 旗標至子程序，不修改父程序環境。

目前 `internal/runner` 已實作四種 target 的 provider 注入：
- Copilot：環境變數（`BuildEnv`）
- Codex/Codex App：`--config` TOML 旗標（`BuildCodexArgs`）
- Claude：環境變數（`BuildClaudeEnv`）
- Pi：環境變數 + 臨時 `models.json`（`BuildPiEnv` + `LaunchPi`）

設定檔 `~/.byok/config.yaml` 目前僅有 `profiles` 與 `default_profile` 兩個頂層欄位。

各 target 原生 OTEL 支援現況：
- **Copilot CLI**：環境變數驅動，僅支援 OTLP HTTP（`http/json`、`http/protobuf`），不支援 gRPC
- **Codex**：`config.toml` 的 `[otel]` section，支援 `otlp-http`（binary/json）與 `otlp-grpc`
- **Claude**：環境變數驅動，支援 `grpc`、`http/json`、`http/protobuf`
- **Pi**：需安裝 `pi-otel-telemetry` extension，環境變數驅動，僅支援 OTLP HTTP

## Goals / Non-Goals

**Goals:**

- 設定檔新增 `telemetry` 區段，同時承載 gRPC 與 HTTP endpoint
- 各 runner 的 Build* 函式依據 telemetry 設定自動注入對應格式
- service_name 未設定時不注入；設定時組合為 `<custom-name>-<agent-name>`
- dry-run 輸出包含 telemetry 注入內容
- README 新增 OpenTelemetry 說明章節

**Non-Goals:**

- 不新增 CLI 子指令管理 telemetry（首版直接編輯 YAML）
- 不支援 per-signal endpoint（traces/metrics/logs 分開送不同 endpoint）
- 不支援 mTLS 證書路徑
- 不支援 file exporter
- 不實作 byok 自身的 OTEL instrumentation
- 不處理 Pi extension 的自動安裝

## Decisions

### 設定檔結構：頂層 telemetry 區段，分離 gRPC 與 HTTP

OTEL 設定為團隊/組織級別共用，不因 provider profile 而異，因此放在頂層而非 profile 內。

同時提供 `grpc` 與 `http` 兩組 endpoint，避免 protocol 不相容時的降級邏輯。各 target 依原生能力直接選用：

```yaml
telemetry:
  enabled: true
  service_name: "my-team"
  headers:
    Authorization: "Bearer xxx"
  grpc:
    endpoint: "http://localhost:4317"
  http:
    endpoint: "http://localhost:4318"
    protocol: "protobuf"   # protobuf | json（預設 protobuf）
```

替代方案：單一 endpoint + protocol 欄位 → 需要降級邏輯（Copilot 不支援 gRPC），增加複雜度。

### Target 選用優先順序：支援 gRPC 者優先 gRPC

| Target | 選用 | 原因 |
|--------|------|------|
| Copilot CLI | `http` only | 原生僅支援 HTTP |
| Codex/Codex App | `grpc` > `http` | 兩者皆支援，gRPC 效能較佳 |
| Claude | `grpc` > `http` | 兩者皆支援 |
| Pi | `http` only | Plugin 僅支援 HTTP |

若使用者只設 `http`，全部 target 均可運作。若只設 `grpc`，僅 Copilot 與 Pi 無法注入 telemetry（靜默跳過，不報錯）。

### Service name 組合規則

- 未設定 `service_name` → 不注入 `OTEL_SERVICE_NAME`，各 target 使用原生預設
- 設定 `service_name: "x"` → 注入 `x-github-copilot` / `x-codex-cli` / `x-claude-code` / `x-pi-coding-agent`

替代方案：直接注入使用者設定的值不加後綴 → 在 Grafana 中無法區分不同 target。

### Protocol 映射表

```
byok http.protocol → Copilot env var       → Codex --config               → Claude env var
protobuf           → http/protobuf          → otlp-http + binary           → http/protobuf
json               → http/json              → otlp-http + json             → http/json
(grpc endpoint)    → (不適用)               → otlp-grpc                    → grpc
```

### Telemetry 資料結構設計

新增 `internal/config/telemetry.go`，定義 `Telemetry` struct：

```go
type TelemetryGRPC struct {
    Endpoint string `yaml:"endpoint"`
}

type TelemetryHTTP struct {
    Endpoint string `yaml:"endpoint"`
    Protocol string `yaml:"protocol"` // "protobuf" | "json"
}

type Telemetry struct {
    Enabled     bool              `yaml:"enabled"`
    ServiceName string            `yaml:"service_name,omitempty"`
    Headers     map[string]string `yaml:"headers,omitempty"`
    GRPC        *TelemetryGRPC    `yaml:"grpc,omitempty"`
    HTTP        *TelemetryHTTP    `yaml:"http,omitempty"`
}
```

`Config` struct 新增 `Telemetry *Telemetry` 欄位。`telemetry` 為 nil 或 `enabled: false` 時不注入任何設定。

### 注入層級：runner 層負責轉譯

各 `Build*Env` / `BuildCodexArgs` 函式新增 `telemetry *Telemetry` 參數，由 cmd/launch 層傳入。runner 內部負責將 Telemetry struct 轉譯為 target 專屬格式。

## Implementation Contract

**Behavior**：
- `byok launch <target>` 讀取 `~/.byok/config.yaml` 的 `telemetry` 區段
- 若 `telemetry.enabled == true` 且至少一個 endpoint（grpc 或 http）有值，依 target 能力注入 OTEL 設定至子程序
- 若 `telemetry` 不存在、`enabled == false`、或無任何 endpoint，行為與現有版本完全一致（不注入任何 OTEL 設定）

**Interface / data shape**：
- `internal/config.Telemetry` struct 如上述設計
- `internal/config.Config.Telemetry` 欄位（pointer，nil 表示未設定）
- 各 runner `Build*` 函式簽章新增 `telemetry *config.Telemetry` 參數

**Failure modes**：
- `telemetry` 區段 YAML 格式錯誤：設定檔載入失敗，印錯 exit 1（與現有行為一致）
- `protocol` 值非 `protobuf`/`json`：載入時驗證，印錯 exit 1
- 僅設定 `grpc` 但 target 僅支援 HTTP：靜默跳過 telemetry 注入，不報錯不影響 launch
- `headers` 含空值：照常注入，由 target 自行處理

**Acceptance criteria**：
- 單元測試驗證各 runner Build* 函式產出的環境變數/旗標包含正確的 OTEL 設定
- 單元測試驗證 telemetry 為 nil 時不注入任何 OTEL 相關項目
- 單元測試驗證 service_name 組合規則
- 單元測試驗證 protocol 映射正確
- dry-run 輸出包含 telemetry 環境變數或旗標
- `go test ./... -race` 通過

**Scope boundaries**：
- In scope：config 載入、驗證、runner 注入、dry-run 輸出、README 文件
- Out of scope：CLI 子指令、per-signal endpoint、mTLS、file exporter、byok 自身 instrumentation

## Risks / Trade-offs

- [Codex `--config` 不支援巢狀 TOML otel 設定] → 先以 dotted key 嘗試，若失敗則改寫臨時 config.toml
- [Pi 需使用者手動安裝 extension] → README 明確說明前置條件，byok 不自動安裝
- [Copilot CLI 僅設 gRPC 時無法注入] → 靜默跳過，行為與未啟用 telemetry 一致；README 建議同時設定兩組 endpoint
- [Headers 可能包含敏感 token] → config.yaml 以 0600 權限寫入；dry-run 輸出 headers 值為 `***` mask
