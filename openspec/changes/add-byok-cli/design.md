## Context

目前 Copilot CLI 支援 BYOK（Bring Your Own Key），透過設定環境變數 `COPILOT_PROVIDER_BASE_URL`、`COPILOT_PROVIDER_TYPE`、`COPILOT_PROVIDER_API_KEY`、`COPILOT_MODEL` 來連接 OpenAI 相容端點。使用者每次啟動前需手動 export 多個環境變數，且這些設定會汙染目前 shell 的環境。本工具 `byok` 目標是讓使用者以單一指令從設定檔讀取金鑰設定，臨時注入子行程環境變數後啟動 Copilot CLI，結束後不留下任何痕跡。

本專案以 Go 語言開發，使用 Cobra 作為 CLI 框架，設定檔以 YAML 格式儲存於使用者家目錄 `~/.byok/config.yaml`。首版僅支援 Copilot CLI 與 OpenAI 相容格式。

## Goals / Non-Goals

**Goals:**

- 提供 `byok launch copilot [--model <model>] [--profile <name>]` 指令，以子行程環境變數臨時注入 BYOK 設定後啟動 `copilot`。
- 提供設定檔管理指令 `byok config add/list/remove`，設定檔位於 `~/.byok/config.yaml`。
- 設定檔每組設定包含四個欄位：Provider、API Base、API Key、Default Model。
- 透過修改子行程環境變數（`os.Environ()` + 覆寫 BYOK 變數）方式注入，不寫入系統環境變數或 shell 設定檔。
- 提供 README.md，讓有程式經驗但未曾寫過 Go 的開發者能完成 Go 環境建置、編譯與執行。

**Non-Goals:**

- 不支援 Codex CLI、Claude CLI 等其他工具（首版僅 Copilot CLI）。
- 不支援 Azure、Anthropic 專屬流程，僅支援 OpenAI 相容格式。
- 不提供金鑰加密儲存或金鑰管理服務整合。
- 不提供 GUI 介面。
- 不永久修改系統環境變數。

## Decisions

### 使用 Go 語言與 Cobra CLI 框架

選擇 Go 是因為使用者指定，且 Go 編譯為單一可執行檔、跨平台、無執行期依賴，適合 CLI 工具。使用 Cobra（業界事實標準 CLI 框架）提供子指令、旗標、說明文字自動產生。

替代方案：使用 Go 內建 `flag` 套件（功能不足，子指令支援差）、使用 urfave/cli（次流行框架，但生態系較小）。選擇 Cobra 因生態成熟、文件完整。

### 設定檔格式為 YAML 並存放於 ~/.byok/config.yaml

YAML 可讀性高且支援多 profile 結構。路徑 `~/.byok/config.yaml` 遵循慣例且不汙染專案目錄。使用 `gopkg.in/yaml.v3` 套件解析。

設定檔結構：

```yaml
profiles:
  - name: openai-official
    provider: openai
    api_base: https://api.openai.com/v1
    api_key: sk-xxxx
    default_model: gpt-4o
  - name: local-ollama
    provider: openai
    api_base: http://localhost:11434
    api_key: ""
    default_model: llama3.2
default_profile: openai-official
```

替代方案：JSON（不支援註解、可讀性較差）、TOML（結構對多 profile 較不直覺）。選擇 YAML 平衡可讀性與結構表達力。

### 透過子行程環境變數注入 BYOK 設定

啟動 Copilot 時，從目前行程環境（`os.Environ()`）複製一份，覆寫以下四個 BYOK 環境變數後以 `exec.Command` 啟動子行程，並將 stdin/stdout/stderr 串接：

- `COPILOT_PROVIDER_BASE_URL`
- `COPILOT_PROVIDER_TYPE`（預設 `openai`）
- `COPILOT_PROVIDER_API_KEY`
- `COPILOT_MODEL`

子行程結束後，父行程環境不受影響，使用者 shell 環境也不變更。

替代方案：寫入系統環境變數（會汙染且需清理）、使用 shell wrapper script（跨平台困難）。選擇子行程環境變數最乾淨且跨平台。

### 預設 Provider 為 openai 且僅支援 OpenAI 相容格式

首版僅支援 `openai` provider type，對應 Copilot CLI 的預設值。設定檔 provider 欄位保留擴充空間但驗證時只接受 `openai`。

### --model 旗標覆寫預設模型

`byok launch copilot --model glm-5.2` 會以旗標值覆寫 profile 的 default_model。未指定旗標時使用 profile 的 default_model。

### README.md 面向 Go 新手

README 包含：Go 環境安裝指引（各平台）、`go build`/`go install` 編譯方式、`go run main.go` 執行方式、設定檔範例、使用範例。不假設讀者具備 Go 知識。

## Implementation Contract

**行為：**

- 執行 `byok launch copilot` 後，以 profile 中的設定注入環境變數並啟動 `copilot` 子行程，stdin/stdout/stderr 透通串接，使用者與 Copilot 互動如常。
- 執行 `byok config add --name <name> --provider <p> --api-base <url> --api-key <key> --default-model <m>` 將一組設定寫入 `~/.byok/config.yaml`。
- 執行 `byok config list` 列出所有 profile 名稱與四個欄位（api_key 以遮罩顯示，僅顯示前 4 與後 4 字元）。
- 執行 `byok config remove --name <name>` 從設定檔移除指定 profile。

**介面 / 資料形狀：**

- 設定檔路徑：`~/.byok/config.yaml`（可由 `--config` 旗標覆寫）。
- 設定檔結構如 Decisions 中 YAML 範例，每個 profile 含 `name`、`provider`、`api_base`、`api_key`、`default_model`。
- CLI 指令介面：
  - `byok launch copilot [--model <model>] [--profile <name>] [--config <path>]`
  - `byok config add --name <name> --provider <p> --api-base <url> --api-key <key> --default-model <m> [--config <path>]`
  - `byok config list [--config <path>]`
  - `byok config remove --name <name> [--config <path>]`
- 注入之環境變數：`COPILOT_PROVIDER_BASE_URL`、`COPILOT_PROVIDER_TYPE`、`COPILOT_PROVIDER_API_KEY`、`COPILOT_MODEL`。

**失敗模式：**

- 設定檔不存在：`byok launch` 顯示錯誤訊息並退出（exit code 1），提示使用者先執行 `byok config add`。
- 指定 profile 不存在：顯示錯誤訊息列出可用 profile 名稱，退出碼 1。
- `copilot` 執行檔不存在於 PATH：顯示錯誤訊息提示安裝 Copilot CLI，退出碼 1。
- 設定檔格式錯誤：顯示 YAML 解析錯誤訊息與檔案路徑，退出碼 1。
- provider 非 `openai`：顯示錯誤訊息說明首版僅支援 openai，退出碼 1。

**驗收條件：**

- `byok config add` 寫入後，`byok config list` 能列出該 profile 且 api_key 遮罩。
- `byok launch copilot --model glm-5.2` 啟動的子行程環境中 `COPILOT_MODEL` 為 `glm-5.2`、`COPILOT_PROVIDER_BASE_URL` 為 profile 中 api_base（可透過 `byok launch copilot --dry-run` 或測試驗證）。
- 父行程（`byok` 本身）執行前後環境變數不變更。
- README.md 步驟可被未寫過 Go 的開發者依序完成環境建置與執行。

**範圍邊界：**

- In scope：Copilot CLI 啟動、OpenAI 相容 profile 設定檔管理、子行程環境變數注入、Go 環境建置說明。
- Out of scope：Codex CLI / Claude CLI 啟動、Azure / Anthropic provider 專屬流程、金鑰加密、GUI、系統環境變數永久寫入。

## Risks / Trade-offs

- [設定檔明文儲存 API Key] → 文件中明確警告，建議檔案權限設為 600；未來可擴充加密。
- [copilot 執行檔路徑因平台/安裝方式不同] → 使用 `exec.LookPath` 檢查並給出明確錯誤訊息。
- [子行程環境變數注入在 Windows 與 Unix 行為差異] → Go `exec.Cmd.Env` 跨平台一致，測試覆蓋 Windows。
- [設定檔 schema 變更導致向前相容問題] → 首版無舊資料，後續變更需 migration 時再處理。
