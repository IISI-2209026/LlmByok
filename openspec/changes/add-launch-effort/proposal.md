## Why

目前 byok 只能在啟動時選擇模型，使用者無法在不修改目標 CLI 設定檔或父程序環境的前提下，指定模型的思考程度或 Claude subagent 的模型。Copilot、Codex、Claude 與 pi 都有各自的啟動層級 effort 介面，而 Claude 可透過環境變數指定 subagent 模型；byok 應將這些暫時設定收斂為選填旗標。同時，使用者需要能檢視並自行調整 byok 即將使用的 target CLI 命令，而不啟動 target。

## What Changes

- 為 `byok launch <target>` 新增選填的 `--effort <level>` 旗標；未指定時不得覆寫任何目標 CLI 的原生思考程度預設值。
- 依 target 驗證 effort 值域，並在不支援時以明確錯誤拒絕啟動。
- 將有效 effort 僅注入目標子程序：Copilot 使用 `--reasoning-effort`、Codex 與 Codex App 使用頂層 `--config model_reasoning_effort`、Claude 使用 effort 環境變數、pi 使用 `--thinking`。
- 為 `byok launch <target>` 新增選填的 `--sub-model <model>` 旗標；只有 Claude 將其暫時注入 `CLAUDE_CODE_SUBAGENT_MODEL`，其餘 target 接受但不使用該值且不報錯。
- 為 `byok launch <target>` 新增 `--dry-run` 旗標；解析 profile、模型、effort、sub-model、yolo 與 passthrough 後，只輸出可複製執行的等效 target 命令，不啟動 target。
- 依執行 byok 的平台輸出 PowerShell 或 POSIX shell 語法；輸出 API Key 時固定使用已正確引用的 `***` placeholder，不讀取 keychain 或設定檔明碼金鑰。pi 的輸出包含建立暫存 `models.json`、執行 pi 與清理暫存目錄的完整命令片段。
- 維持正常 launch 的 BYOK 設定只在子程序或 pi 暫存目錄生效，不修改使用者設定檔與父程序環境。
- 當使用者執行 `byok launch` 而未提供 target 時，先輸出與 `byok launch --help` 相同的 launch 說明，再印出缺少 target 的錯誤並以 exit code 1 結束；其他 launch 執行期錯誤維持既有精簡錯誤輸出。
- 更新 README.md、AGENTS.md 與相關 launch 規格，說明旗標、目標差異、有效值、dry-run 安全限制與範例。

## Capabilities

### New Capabilities

- `byok-launch-effort`: 提供跨目標 CLI 的選填思考程度與 sub-model 旗標、target-specific 驗證與暫時注入行為。
- `byok-launch-dry-run`: 輸出跨平台、可執行且遮罩 API Key 的 target CLI 等效命令。

### Modified Capabilities

- `byok-launch`: 將共用 `byok launch <target>` 介面擴充為接受並分派 `--effort`、`--sub-model` 與 `--dry-run`。
- `byok-launch`: 未帶 target 的呼叫會顯示 launch help，協助使用者選擇受支援 target。
- `byok-codex-launch`: Codex 與 Codex App 啟動時可用臨時 config override 指定 reasoning effort。
- `byok-claude-launch`: Claude 啟動時可用臨時環境變數指定 effort 與 subagent model。
- `byok-pi-launch`: pi 啟動時可用臨時 `--thinking` 參數指定思考程度。

## Impact

- Affected specs: byok-launch-effort, byok-launch-dry-run, byok-launch, byok-codex-launch, byok-codex-app-launch, byok-claude-launch, byok-pi-launch
- Affected code:
  - Modified: cmd/launch.go
  - Modified: cmd/launch_codex.go
  - Modified: cmd/launch_codex_app.go
  - Modified: cmd/launch_claude.go
  - Modified: cmd/launch_pi.go
  - Modified: internal/runner/runner.go
  - Modified: internal/runner/codex.go
  - Modified: internal/runner/claude.go
  - Modified: internal/runner/pi.go
  - New: cmd/launch_dry_run.go
  - Modified: cmd/launch_test.go
  - Modified: cmd/launch_dispatch_test.go
  - Modified: cmd/launch_codex_test.go
  - Modified: cmd/launch_codex_app_test.go
  - Modified: cmd/launch_claude_test.go
  - Modified: cmd/launch_pi_test.go
  - New: cmd/launch_dry_run_test.go
  - Modified: internal/runner/runner_test.go
  - Modified: internal/runner/codex_test.go
  - Modified: internal/runner/claude_test.go
  - Modified: internal/runner/pi_test.go
  - Modified: README.md
  - Modified: AGENTS.md
  - Modified: openspec/specs/byok-launch/spec.md
  - Modified: openspec/specs/byok-codex-launch/spec.md
  - Modified: openspec/specs/byok-codex-app-launch/spec.md
  - Modified: openspec/specs/byok-claude-launch/spec.md
  - Modified: openspec/specs/byok-pi-launch/spec.md
  - New: openspec/specs/byok-launch-effort/spec.md
  - New: openspec/specs/byok-launch-dry-run/spec.md
  - Removed: none
