## 1. 共用 launch effort、sub-model 與 dry-run 介面

- [ ] 1.1 在 cmd/launch.go 實作「Optional target-specific reasoning effort」、「Dispatch optional reasoning effort」、「Optional Claude subagent model selection」與「Dispatch optional subagent model」：新增選填 `--effort` 與 `--sub-model`、依 target 驗證 effort allowlist、將有效 effort 傳入五個 launch flow，僅將 sub-model 傳入 Claude flow，其他四個 target 對 sub-model 保持 no-op；以 cmd/launch_test.go 和各 target launch test 驗證有效分派、無效 effort 拒絕、sub-model no-op 與未啟動子程序。
- [ ] 1.2 在 cmd/launch.go 與相關 command help 落實「使用選填的 launch effort、sub-model 與 dry-run 旗標」：註冊 `--dry-run`，並讓三個旗標遺漏時維持既有模型、profile、yolo 與 passthrough 行為；以 CLI help 輸出與既有無 override 測試確認沒有新增 argument 或 environment entry。
- [ ] 1.3 在共用 launch validation 落實「在啟動前依 target 驗證 effort 值域」：依 target allowlist 在 normal launch 或 dry-run 輸出前拒絕無效 effort，並讓 sub-model 維持未驗證 opaque value；以 cmd/launch_test.go 驗證錯誤訊息包含 target、請求值與有效值，並確認不會執行 child process 或輸出命令。

## 2. 目標 CLI 暫時注入

- [ ] [P] 2.1 在 internal/runner/runner.go 與 cmd/launch_test.go 落實「以原生暫時性介面轉換設定」的 Copilot adapter，讓有效 Copilot effort 值以 `--reasoning-effort <level>` 出現在 yolo 與 passthrough 前，並使 `--sub-model` 不產生參數或環境覆寫；以 internal/runner/runner_test.go 驗證精確參數順序、effort 省略與 sub-model no-op。
- [ ] [P] 2.2 在 internal/runner/codex.go、cmd/launch_codex.go 與 cmd/launch_codex_app.go 落實「Launch Codex with an optional reasoning effort」和「Launch Codex App with an optional reasoning effort」，讓有效值成為頂層 `--config model_reasoning_effort="<level>"`，維持 `codex app` 第一個參數與 config、yolo、passthrough 順序，且 `--sub-model` 不改變 Codex child invocation；以 internal/runner/codex_test.go、internal/runner/codex_launch_test.go、internal/runner/codex_app_test.go、cmd/launch_codex_test.go、cmd/launch_codex_app_test.go 驗證注入、省略與 no-op。
- [ ] [P] 2.3 在 internal/runner/claude.go 與 cmd/launch_claude.go 實作「Launch Claude with an optional reasoning effort」與「Launch Claude with an optional subagent model」，讓明確 effort 僅在子程序環境加入 `CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1` 和 `CLAUDE_CODE_EFFORT_LEVEL=<level>`，讓明確 sub-model 僅加入未驗證的 `CLAUDE_CODE_SUBAGENT_MODEL=<model>`；以 internal/runner/claude_test.go 與 cmd/launch_claude_test.go 驗證子程序環境、父程序隔離、個別省略及兩旗標同時使用。
- [ ] [P] 2.4 在 internal/runner/pi.go 與 cmd/launch_pi.go 實作「Launch Pi with an optional thinking level」，讓有效值以 `--model <model> --thinking <level>` 排在 yolo 與 passthrough 前，讓 `--sub-model` 不新增 Pi argument 或暫存設定，並維持 `PI_CODING_AGENT_DIR` 與 cleanup 行為；以 internal/runner/pi_test.go 與 cmd/launch_pi_test.go 驗證順序、無 effort 時未傳 `--thinking`、sub-model no-op 與暫存目錄清理。
- [ ] 2.5 在所有 runner 實作「保持 runner 與 renderer contract 明確」：以分離的 model、effort、sub-model、extraArgs 與 profile 連線資料建立子程序設定，避免 passthrough 覆寫 byok 管理的設定；以所有 runner stub tests 斷言 argument ordering 與 environment entries。

## 3. dry-run 命令渲染

- [ ] 3.1 在 cmd/launch_dry_run.go 實作「Print a masked equivalent target command」：`--dry-run` 解析 config、profile、provider、model、effort、sub-model、yolo、passthrough 後輸出命令且不呼叫 key resolver、`exec.LookPath` 或 target runner；以 cmd/launch_dry_run_test.go 驗證 keychain 與明碼 key 不被讀取、`***` 被引用、target 未安裝仍能輸出且無子程序啟動。
- [ ] 3.2 在 cmd/launch_dry_run.go 實作「Render platform-specific equivalent commands」：Windows 渲染可貼上的 PowerShell，非 Windows 渲染 POSIX shell；Copilot、Codex、Codex App、Claude 以等效環境與參數輸出，pi 產生含唯一暫存目錄、masked models.json、`PI_CODING_AGENT_DIR`、pi invocation 與清理的完整片段；以 cmd/launch_dry_run_test.go 針對五個 target 斷言 shell quoting、yolo/passthrough 順序與 pi cleanup。
- [ ] 3.3 在 cmd/launch.go 實作「Dispatch dry-run command rendering」，使 `--dry-run` 使用相同的 profile、模型、effort、sub-model 解析與 validation，但繞過所有 target launch flow；以 cmd/launch_dry_run_test.go 驗證 Copilot、Codex、Codex App、Claude、pi 都只輸出命令。

## 4. 文件與規格同步

- [ ] [P] 4.1 更新 README.md，使使用者可查到 `byok launch <target> --effort <level> --sub-model <model> --dry-run` 的選填語意、effort 映射、僅 Claude 注入 `CLAUDE_CODE_SUBAGENT_MODEL`、其他 target 的 no-op、dry-run 平台輸出、`***` API key 替換要求與範例；以文件內容檢閱確認未宣稱 dry-run 會讀取或輸出實際金鑰。
- [ ] [P] 4.2 更新 AGENTS.md 的 CLI 介面與 BYOK 注入規範，記錄 common effort/sub-model/dry-run 旗標、Claude subagent model 環境變數、其他 target no-op、dry-run 的跨平台 shell、masked key 與不執行 target 限制；以文件內容檢閱確認與 README.md 及 design 的「產生平台原生且不含 API key 的 dry-run 命令」一致。
- [ ] 4.3 以 spectra verify add-launch-effort、spectra analyze add-launch-effort --json 與 spectra validate add-launch-effort 確認所有新增 requirement 與 design decision 已被規格、設計與任務覆蓋。

## 5. 整合驗證

- [ ] 5.1 執行 go test ./... -race，確認 `--effort`、`--sub-model`、`--dry-run` 在 Claude 注入、四個非 Claude target no-op、五個 target dry-run、valid、omitted 與 invalid cases 均不破壞既有 launch 行為，並修正本變更導致的測試失敗。
