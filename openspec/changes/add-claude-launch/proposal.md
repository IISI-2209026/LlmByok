## Why

`byok launch` 目前只支援 `copilot` 與 `codex` 兩個目標工具。使用者希望以 BYOK profile 暫時注入環境變數啟動 Claude Code（`claude`），獲得與 copilot/codex 一致的功能（profile 解析、provider 驗證、可執行檔檢查、`--model` 覆寫、`--yolo`、`--` 透傳）。Claude Code 透過 `ANTHROPIC_BASE_URL`、`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL` 等環境變數接受第三方部署設定（見 [模型配置文件](https://code.claude.com/docs/zh-TW/model-config#pin-models-for-third-party-deployments)），適合以 byok 的「僅子程序注入、不寫設定檔」機制承載。

## What Changes

- 新增 `byok launch claude` 子流程：讀取選定的 profile，將 `ANTHROPIC_BASE_URL`、`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL` 注入 `claude` 子程序環境（父程序與 shell 環境不變），stdin/stdout/stderr 透明連接。
- `byok launch` 的目標分派新增 `claude` 分支，錯誤訊息與 usage 說明同步列出 `copilot`、`codex`、`claude` 三個支援目標。
- 新增 `internal/runner` 的 Claude 啟動函式（建置子程序環境並啟動），與既有 copilot/codex 啟動函式對等。
- 維持既有行為：profile 預設解析、`--profile`/`--model`/`--config` 旗標、provider 驗證（首版僅 `openai`）、`--yolo`/`--` 透傳、可執行檔存在檢查、金鑰解析（keychain 優先、明碼 fallback）、父程序環境不變。
- 不寫入任何 Claude Code 設定檔（如 `~/.claude/settings.json`）；所有 BYOK 設定僅透過子程序環境變數傳遞。

## Non-Goals

- 不新增 `claude` 以外的目標工具。
- 不改變 copilot、codex 既有啟動行為。
- 不支援非 `openai` 的 provider 類型（首版僅 `openai`，與 copilot/codex 一致）。
- 不修改 `~/.claude/settings.json` 或任何 Claude Code 設定檔。
- 不實作 Claude 特有的固定模型（`ANTHROPIC_DEFAULT_OPUS_MODEL` 等）覆寫；首版僅注入 base url、api key、model 三項基本設定。

## Capabilities

### New Capabilities

- `byok-claude-launch`: 以 BYOK profile 啟動 Claude Code（`claude`）子程序，注入 `ANTHROPIC_BASE_URL`/`ANTHROPIC_API_KEY`/`ANTHROPIC_MODEL`，與 copilot/codex 啟動流程對等。

### Modified Capabilities

- `byok-launch`: 目標工具分派新增 `claude` 分支，錯誤訊息與 usage 同步支援 `claude`。

## Impact

- Affected specs:
  - New: `openspec/specs/byok-claude-launch/spec.md`
  - Modified: `openspec/specs/byok-launch/spec.md`
- Affected code:
  - New: `cmd/launch_claude.go`
  - New: `internal/runner/claude.go`
  - Modified: `cmd/launch.go`（目標分派新增 `claude`，usage 範例與說明同步）
  - Modified: `README.md`（一般化簡介、新增 claude target 至 Targets 表與範例、新增 Claude BYOK 運作原理小節、官方文件連結與疑難排解）
