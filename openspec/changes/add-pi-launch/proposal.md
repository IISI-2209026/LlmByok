## Why

Pi（pi.dev）是一個極簡 terminal coding harness，支援多種 provider。使用者希望像 copilot、codex、claude 一樣，以 BYOK profile 暫時注入自己的 OpenAI 相容 API 金鑰與端點來啟動 pi，而不修改任何 pi 設定檔。

## What Changes

- 新增 `byok launch pi` 子指令分派，與現有 copilot/codex/codex-app/claude 並列。
- pi 的 BYOK 注入採用 `PI_CODING_AGENT_DIR` 環境變數指向臨時目錄，目錄內放置 `models.json` 覆寫 `openai` provider 的 `baseUrl`（profile 的 `api_base`）與 `apiKey`（profile 的 `api_key`），再透過 `--model` CLI 旗標傳遞模型。父程序環境與使用者 pi 設定檔永不被修改。
- `--yolo` 旗標對 pi 映射為 `--approve`（自動信任專案本機檔案）。
- 更新 README.md 將 pi 列為支援的目標工具。
- 更新 AGENTS.md 套件職責與開發規範段落。

## Capabilities

### New Capabilities

- `byok-pi-launch`: 以 BYOK profile 啟動 pi CLI 的注入機制（PI_CODING_AGENT_DIR + 臨時 models.json + --model 旗標）。

### Modified Capabilities

- `byok-launch`: 「Target tool selection and dispatch」需求新增 `pi` 為支援的目標工具。

## Impact

- Affected specs: `byok-pi-launch`（new）、`byok-launch`（modified）
- Affected code:
  - New: `cmd/launch_pi.go`、`internal/runner/pi.go`、`cmd/launch_pi_test.go`、`internal/runner/pi_test.go`
  - Modified: `cmd/launch.go`（dispatch switch、usage template、examples、error messages）、`cmd/launch_dispatch_test.go`、`README.md`、`AGENTS.md`
