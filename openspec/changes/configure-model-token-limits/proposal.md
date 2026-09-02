## Why

自訂 BYOK gateway 經常使用未收錄於各 coding agent 內建目錄的模型；目前 byok 無法宣告模型的 Context Window 與最大輸出量，且 Copilot 與 Claude runner 還帶有固定 token 值或自動加上 `[1m]` 的假設。這會造成提早壓縮、錯誤的模型能力宣告，或對不支援 1M context 的模型傳入不正確名稱。

## What Changes

- 將每個 profile 的 `models` 從純字串候選清單擴充為可帶 `name`、`context_window_tokens` 與 `max_output_tokens` 的模型物件清單，並繼續接受既有字串清單。
- 在 profile 增加選填的 `default_model_limits`，讓未設定個別限制的模型逐欄位繼承預設值。
- 在 `byok launch <target>` 增加 `--context-window-tokens` 與 `--max-output-tokens`，解析順序固定為命令列、選定模型、profile 預設、未設定。
- 將有效值映射至 Copilot、Codex、Codex App、Claude 與 Pi 的官方支援介面；target 不支援個別參數時在 stderr 顯示 warning、忽略該值並繼續啟動。
- 移除 Copilot 固定注入的 prompt/output token 數值；未解析出有效值時不注入對應環境變數。
- **BREAKING**：移除 Claude 對所有模型名稱自動附加 `[1m]` 的行為；需要 extended-context 模型的使用者必須在模型名稱中明確提供官方支援的 suffix。
- Pi 透過暫存 `models.json` 的 model override 傳入 context/output 限制；明確設定最大輸出量時，暫存設定同時保留足夠的 response headroom，但不公開跨 target 的壓縮門檻欄位。
- 擴充 dry-run，使輸出反映相同的 token 設定、Pi 暫存檔內容與不支援 warning，且不改變 API key masking。
- 更新 README 與 AGENTS.md，記錄 YAML 格式、舊格式相容性、優先序、target 支援矩陣、warning 行為、官方語意差異與移除的硬編碼。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `byok-config`: profile 模型結構、預設模型限制、驗證、舊格式載入與 set-models 保留同名模型限制。
- `byok-launch`: token 旗標、逐欄位優先序、模型限制解析、驗證與不支援 target 的 warning。
- `byok-launch-dry-run`: dry-run 命令與暫存設定必須反映有效 token 限制及 warning。
- `byok-codex-launch`: Codex context window override 與不支援最大輸出的相容行為。
- `byok-codex-app-launch`: Codex App context window override 與不支援最大輸出的相容行為。
- `byok-claude-launch`: Claude context/output 環境變數與不再自動附加 `[1m]`。
- `byok-pi-launch`: Pi model override、輸出 headroom 與暫存設定生命週期。
- `byok-agent-docs`: README 與 AGENTS.md 的設定格式及注入契約。

## Impact

- Affected specs: byok-config, byok-launch, byok-launch-dry-run, byok-codex-launch, byok-codex-app-launch, byok-claude-launch, byok-pi-launch, byok-agent-docs
- Affected code:
  - New: none
  - Modified:
    - `internal/config/config.go`
    - `internal/config/config_test.go`
    - `internal/config/models.go`
    - `internal/config/models_test.go`
    - `cmd/config.go`
    - `cmd/config_test.go`
    - `cmd/set_models.go`
    - `cmd/set_models_test.go`
    - `cmd/launch.go`
    - `cmd/launch_model_test.go`
    - `cmd/launch_dispatch_test.go`
    - `cmd/launch_dry_run.go`
    - `cmd/launch_dry_run_test.go`
    - `internal/runner/runner.go`
    - `internal/runner/runner_test.go`
    - `internal/runner/codex.go`
    - `internal/runner/codex_test.go`
    - `internal/runner/claude.go`
    - `internal/runner/claude_test.go`
    - `internal/runner/pi.go`
    - `internal/runner/pi_test.go`
    - `README.md`
    - `AGENTS.md`
  - Removed: none
