## Why

`byok launch copilot` 目前只支援以 BYOK profile 啟動 Copilot CLI，無法將額外參數傳遞給 Copilot。使用者若想使用 Copilot 的 `--yolo` 模式或 `--continue` 等選項，必須離開 byok 另外執行 copilot，失去 BYOK 暫時切換金鑰的便利性。

## What Changes

- 新增 `-y` / `--yolo` 旗標至 `byok launch copilot`；設定時自動在傳給 copilot 的參數尾端附加 `--yolo`
- 新增 `--` 透傳機制：`byok launch copilot -- <args>` 將 `<args>` 原樣轉發給 copilot 可執行檔
- `runner.Launch` 函式新增 `extraArgs []string` 參數，用來組合 copilot 的命令列參數

## Non-Goals

- 不支援其他 CLI 工具（Codex CLI、Claude CLI），僅限 Copilot CLI
- 不解析或驗證透傳參數的內容，由 copilot 自行處理
- 不改變 BYOK 環境變數注入邏輯

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `byok-launch`: 新增 `-y`/`--yolo` 旗標與 `--` 透傳參數轉發能力

## Impact

- Affected specs: byok-launch
- Affected code:
  - Modified: cmd/launch.go, internal/runner/runner.go
  - Modified: cmd/launch_test.go, internal/runner/launch_integration_test.go
