## Why

開發者想在使用 Copilot CLI 時臨時切換到自己的 API 金鑰（BYOK, Bring Your Own Key），以使用 OpenAI 相容端點的模型。目前 Copilot CLI 透過環境變數設定 BYOK，使用者每次都要手動設定多個環境變數，既繁瑣又容易出錯。需要一個命令列工具，讓使用者以單一指令快速切換金鑰並啟動 Copilot，且不影響系統原本的環境設定。

## What Changes

- 新增一支以 Go 語言開發的命令列工具 `byok`，提供 `byok launch copilot --model <model>` 指令，臨時注入 BYOK 環境變數後啟動 Copilot CLI。
- 新增 YAML/JSON 設定檔（位於使用者家目錄，例如 `~/.byok/config.yaml`），記錄每組金鑰設定的四個欄位：Provider、API Base、API Key、Default Model。
- 新增 `byok` 子指令管理設定檔：`byok config add`、`byok config list`、`byok config remove`，用於新增、列出、移除金鑰設定。
- 透過修改子行程（child process）環境變數的方式臨時注入金鑰，不寫入系統環境變數，啟動的 Copilot 結束後即恢復原狀，不影響日常使用。
- 新增 README.md，說明如何建置 Go 開發環境與執行程式，目標讀者為有程式經驗但未曾寫過 Go 的開發者。

## Non-Goals

- 不支援 Codex CLI、Claude CLI 等其他工具的 BYOK 切換（本版僅支援 Copilot CLI）。
- 不支援非 OpenAI 相容格式的 API 提供者（如 Azure、Anthropic 專屬流程）。
- 不永久修改系統環境變數或 shell 設定檔（僅以子行程環境變數臨時注入）。
- 不提供金鑰加密儲存或金鑰管理服務整合（設定檔以明文或簡單遮罩方式儲存，由使用者自行保管）。
- 不提供 GUI 介面，僅提供命令列介面。

## Capabilities

### New Capabilities

- `byok-launch`: 依據設定檔中的金鑰設定，臨時注入 BYOK 環境變數並啟動目標 CLI 工具（首版僅 Copilot CLI），支援以 `--model` 旗標覆寫預設模型。
- `byok-config`: 管理本機 BYOK 金鑰設定檔，包含新增、列出、移除金鑰設定，設定內容包含 Provider、API Base、API Key、Default Model 四個欄位。
- `byok-setup`: 協助使用者建立 Go 開發環境並建置 `byok` 工具，透過 README.md 與專案結構提供給有程式經驗但未曾寫過 Go 的開發者清楚的建置與執行指引。

### Modified Capabilities

(none)

## Impact

- Affected specs: `byok-launch`、`byok-config`、`byok-setup`（皆為新規格）
- Affected code:
  - New: `main.go`、`cmd/launch.go`、`cmd/config.go`、`internal/config/config.go`、`internal/runner/runner.go`、`go.mod`、`README.md`、`Makefile`
  - Modified: (none)
  - Removed: (none)
