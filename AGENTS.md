<!-- SPECTRA:START v1.0.2 -->

# Spectra Instructions

This project uses Spectra for Spec-Driven Development(SDD). Specs live in `openspec/specs/`, change proposals in `openspec/changes/`.

## Use `$spectra-*` skills when:

- A discussion needs structure before coding → `$spectra-discuss`
- User wants to plan, propose, or design a change → `$spectra-propose`
- Tasks are ready to implement → `$spectra-apply`
- There's an in-progress change to continue → `$spectra-ingest`
- User asks about specs or how something works → `$spectra-ask`
- Implementation is done → `$spectra-archive`
- Commit only files related to a specific change → `$spectra-commit`

## Workflow

discuss? → propose → apply ⇄ ingest → archive

- `discuss` is optional — skip if requirements are clear
- Requirements change mid-work? `ingest` → resume `apply`

## Parked Changes

Changes can be parked（暫存）— temporarily moved out of `openspec/changes/`. Parked changes won't appear in `spectra list` but can be found with `spectra list --parked`. To restore: `spectra unpark <name>`. The `$spectra-apply` and `$spectra-ingest` skills handle parked changes automatically.

<!-- SPECTRA:END -->



# 專案架構

`byok` 是一支以 Go 1.26+ 與 [cobra](https://github.com/spf13/cobra) 建構的命令列工具，模組路徑為 `github.com/IISI-2209026/LlmByok`，入口為 `cmd/byok/main.go`。它以 BYOK（Bring Your Own Key）profile 暫時啟動 Copilot 或 Codex CLI，不修改父程序環境或使用者設定檔。

## 套件職責

| 套件                 | 職責                                                                 |
| -------------------- | -------------------------------------------------------------------- |
| `cmd/byok`           | 程式入口（`main` package），呼叫 `cmd.NewRoot` 建立根指令。            |
| `cmd`                | cobra 指令定義與目標工具分派（`launch copilot` / `launch codex` / `launch codex-app` / `launch claude` / `launch pi`）、`config` 子指令（`add`/`update`/`delete`/`list`/`set-default`/`set-models`）、`update` 子指令。 |
| `internal/config`    | YAML profile 的載入、儲存與驗證；設定檔預設位於 `~/.byok/config.yaml`；金鑰解析（`KeyResolver` 介面、`DefaultResolver`：keychain 優先 → 明碼 fallback）。 |
| `internal/runner`    | BYOK 環境變數建置與子程序啟動（`Launch` for copilot、`LaunchCodex` for codex、`LaunchCodexApp` for codex app、`LaunchClaude` for claude、`LaunchPi` for pi）。 |
| `internal/secret`    | OS keychain 抽象層（zalando/go-keyring）：`Store`/`Load`/`Delete`/`Exists`，service=`byok`、key=`profile:<name>`。 |
| `internal/updater`   | 自我更新：channel 判定、GitHub Releases 查詢、平台資產選擇、下載與跨平台執行檔原子替換。 |
| `internal/version`   | 版本號嵌入（透過 ldflags 注入）。                                      |

## 設定檔

- 設定檔位置：`~/.byok/config.yaml`（可用 `--config` 覆寫）。
- 每個 profile 包含 `name`、`provider`、`api_base`、`api_key`（omitempty，可選）、`models`（候選模型清單）與 profile 層 `default_model_limits`（選填，僅含 `context_window_tokens` 與 `max_output_tokens` 兩個選填欄位，nil = 未設定）。`models` 清單項目接受 YAML 純量（僅模型名稱）或 mapping 形式：必填非空 `name`，另可選正整數 `context_window_tokens` 與 `max_output_tokens`（nil = 未設定）。載入時驗證：模型名稱不可為空、不可重複，token 欄位不可為零或負值；違規時錯誤訊息標明所屬 profile 與模型名稱／欄位。儲存時一律輸出 mapping 形式且不再寫出 legacy `default_model`；載入時 legacy 單一 `default_model` 仍自動遷移為單元素 `models` 清單。
- 候選模型由 `byok config set-models <profile name> --model ...` 維護（整批覆寫，`set-models` 為 `config` 子指令）；`config add`/`update` 不設定模型。`set-models` 以「名稱」比對替換候選清單：同名模型保留其原有的 `context_window_tokens`/`max_output_tokens` 個別限制、清單中的新名稱不帶任何限制、被移除的名稱連同其限制一併消失；profile 層 `default_model_limits` 不受影響。`config list` 與 launch 互動選單一律僅顯示模型名稱。
- `config add`/`delete`/`set-default`/`update` 以第一位置參數接收 profile 名稱（不再使用 `--name` 旗標）。
- `byok launch <target>` 模型解析：帶 `--model` 一律使用之；未帶時依 profile `models` 清單——僅一個直接使用、多個且 stdin 為終端機時顯示上下鍵互動選單（`internal/config.SelectModel`）、多個但非終端機時報錯、為空時報錯並提示 `byok config set-models`。互動選單於真實終端機下將 stdin 切換為 raw mode（`term.MakeRaw`，即時讀鍵、關閉回顯與行緩衝，使方向鍵以 ANSI 序列送達）並於 stdout 啟用虛擬終端機處理（Windows 透過 `SetConsoleMode`，Unix 原生支援，集中於 `internal/config/models_windows.go`/`models_unix.go`），以「❯ 游標 + 反白（`\x1b[7m`）」標記選取列原地重繪；Ctrl-C 或 Esc 取消（回傳 `config.ErrSelectionCancelled`，呼叫端以非零結束碼退出）。解析後的單一模型字串傳入 runner 注入環境變數，runner 不再自行回退 default_model。
- 預設 provider 為 `openai`（空字串回退為 `openai`）；首版僅支援 `openai` provider 類型。
- `byok launch <target>` 另接受選填 `--effort <level>`、`--sub-model <model>`、`--dry-run`、`--context-window-tokens <positive-int>` 與 `--max-output-tokens <positive-int>`。effort 依 target 驗證：Copilot/Codex/Codex App 為 `none|minimal|low|medium|high|xhigh|max`、Claude 為 `low|medium|high|xhigh|max`、pi 為 `off|minimal|low|medium|high|xhigh|max`。effort 只暫時注入子程序：Copilot 使用 `--reasoning-effort`、Codex/Codex App 使用頂層 `--config model_reasoning_effort`、Claude 使用 `CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1` 與 `CLAUDE_CODE_EFFORT_LEVEL`、pi 使用 `--thinking`。`--sub-model` 僅 Claude 注入 `CLAUDE_CODE_SUBAGENT_MODEL`，其他 target 接受但 no-op；token 限制旗標契約見下方「Token 限制注入」；旗標未指定時維持既有行為。
- `--dry-run` 解析設定檔、provider、模型與旗標（含 token 限制，與正常 launch 共用同一套解析結果）後，只輸出平台原生的 PowerShell 或 POSIX shell 等效命令，不讀取 API key、不檢查 target PATH、不啟動子程序；token 限制的平台原生等效輸出（token 環境變數 / codex `--config` / pi masked `models.json` 與選填 `settings.json` 加暫存目錄建立/清理片段）完整呈現於 stdout，對應的 token 警告僅送 stderr；輸出中的 API key 一律為已引用的 `***`。
- API 金鑰以 OS keychain 為主要儲存（`byok config add`/`update` 時以 `--key-storage keychain`（預設）指定），明碼 `api_key` 為 fallback；`launch` 時由 `KeyResolver` 自動解析。
- 頂層 `telemetry` 區段（選填）統一管理 OpenTelemetry 遙測注入：`enabled`（bool）、`service_name`（選填前綴，組合為 `<name>-<agent-name>`）、`headers`（map，認證用）、`grpc.endpoint`、`http.endpoint` + `http.protocol`（`protobuf`|`json`）。`telemetry` 為 nil 或 `enabled: false` 時不注入任何 OTEL 設定。各 runner `Build*` 函式新增 `telemetry *config.Telemetry` 參數；Copilot/Pi 僅使用 HTTP endpoint、Codex/Claude 優先 gRPC 再 fallback HTTP；無相容 endpoint 時靜默跳過。`--dry-run` 輸出包含 telemetry 環境變數/旗標，headers 以 `***` mask。

### Token 限制注入

- **解析優先序**：`--context-window-tokens` 與 `--max-output-tokens` 兩個欄位各自獨立解析，優先序皆為 CLI 旗標 → 所選模型的 `context_window_tokens`/`max_output_tokens` → profile 層 `default_model_limits` → 未設定；未設定即表示 byok 不注入任何 token override（不自行補預設值）。`--model` 給予候選清單外的未知名稱時仍可啟動，其 token 限制僅來自 CLI 旗標與 profile 預設。
- **旗標驗證**：`--context-window-tokens` / `--max-output-tokens` 必須為正整數；值無效時於 API key 解析、target PATH 檢查與子程序啟動之前印錯並以 exit 1 結束。
- **Runner 注入映射（確切名稱；未解析出值的欄位一律不注入）**：
  - **Copilot**：context → 環境變數 `COPILOT_PROVIDER_MAX_PROMPT_TOKENS`；output → 環境變數 `COPILOT_PROVIDER_MAX_OUTPUT_TOKENS`。直接映射、無任何算術運算（先前寫死的 `1048576`/`131072` 已移除）；未設定的變數一律省略。
  - **Codex / Codex App**：context → 頂層 `--config model_context_window=<value>`（未加引號的整數，置於 yolo/passthrough 之前；`codex-app` 仍以 `app` 作為第一參數）；output 不支援 → 由 cmd 層於 stderr 印警告（含 target、參數、數值與來源）後繼續啟動；context 未設定時不附加 `model_context_window`。
  - **Claude**：context → 環境變數 `CLAUDE_CODE_MAX_CONTEXT_TOKENS`；output → 環境變數 `CLAUDE_CODE_MAX_OUTPUT_TOKENS`。破壞性變更：不再自動為模型字串附加 `[1m]` 後綴，模型字串一律原樣傳遞；兩個 token 環境變數皆與父程序環境去除重複。
  - **Pi**：token 限制寫入臨時 `models.json` 的 `providers.openai.modelOverrides.<model>`（`contextWindow` / `maxTokens`）；有效 max_output_tokens 另建立臨時 `settings.json`，其 `compaction.reserveTokens` 設為同值（為 pi 輸出量預留回應 headroom）；兩檔權限皆為 `0600`；無任何有效值時僅產生 provider 層 `models.json`（不含 `modelOverrides`）且不建立 `settings.json`；Pi 結束後刪除整個暫存目錄。
- **僅警告、不視為錯誤**：target/參數不支援的組合（codex 與 codex-app 的 max output）不是錯誤——stderr 印警告、該值忽略、launch 繼續、exit code 不變；未設定的值永不觸發警告。

# 開發規範

- **BYOK 注入僅作用於子程序** — 環境變數只注入到 `copilot` / `codex` / `codex app` / `claude` / `pi` 子行程，父程序（Shell）與系統環境永不被改變；token 限制注入同受此邊界約束——只影響子程序環境變數、codex 命令列旗標或 pi 臨時檔。
- **不寫入使用者設定檔** — `byok` 不會修改 `~/.byok/config.yaml`、`~/.codex/config.toml`、`~/.claude/settings.json`、`~/.pi/agent/models.json` 或任何 Copilot/Codex/Claude/pi 設定檔；codex 連線與 token 限制覆寫僅透過命令列 `--config` 旗標傳遞；claude 僅透過環境變數注入（token 限制環境變數亦同）；pi 透過 `PI_CODING_AGENT_DIR` 環境變數指向臨時目錄（含 `models.json` provider override 與選填 `settings.json`），不修改使用者 pi 設定目錄。
- **不支援的 token 限制組合僅警告、不報錯** — codex 與 codex-app 不支援 max output 限制：於 stderr 印警告（含 target、參數、數值與來源）、該值忽略、launch 繼續進行、exit code 不變；未設定的 token 值一律不產生警告。
- **Profile 解析錯誤印訊息並 exit 1** — 設定檔不存在、profile 找不到、未設 `default_profile`、非 `openai` provider 等情境，皆印出錯誤與提示後以非零結束碼退出。`byok launch` 未帶 target 時例外：stdout 印出與 `byok launch --help` 相同的 launch help（Usage、Targets、Flags、Examples），stderr 印出缺少 target 的錯誤後以非零結束碼退出；不支援的 target 與後續 launch 錯誤不額外印出 help。
- **預設 provider 為 `openai`** — `provider` 欄位為空時回退為 `openai`；非 `openai` 一律拒絕。
- **金鑰以 OS keychain 為主要儲存、明碼 `api_key` 為 fallback** — `byok config add`/`update` 預設以 `--key-storage keychain` 將金鑰存入 keychain（service=`byok`、key=`profile:<name>`）並清除設定檔明碼；可用 `--key-storage plaintext` 改存明碼至設定檔。`delete` 移除 profile 時同步清理 keychain（盡力）。`launch` 時 `KeyResolver` 依 keychain → 明碼順序解析，兩者皆無則報錯。Linux 需 secret-service daemon（gnome-keyring/KWallet）；無 daemon 時回傳 backend-unavailable，可改用 `--key-storage plaintext`。`add`/`update` 支援終端互動模式（未傳欄位旗標時觸發，需 TTY，非 TTY 印錯 exit 1）。
- **測試以 `go test ./...` 執行** — 新增功能須伴隨單元/整合測試。
- **`byok update` 自我更新** — `byok update` 依當前版本 channel（含 `-dev.` 為 dev、否則 stable）查詢 GitHub Releases，下載對應平台資產（`byok-<version>-<goos>-<goarch>.<ext>`）並原子替換執行檔。`--check` 只查詢不替換；`--channel prerelease|release` 覆寫 channel 判定。啟動版本檢查：`launch`/`update` 以外子指令完成後以 3 秒 timeout 查詢，較新時在 stderr 印提示；`BYOK_NO_UPDATE_CHECK=1` 停用；任何錯誤靜默不影響 exit code。
- **Release changelog 以 conventional commit 分類產生** — Release workflow 於建立 GitHub Release 前以 `git log` 取 commit subject，依 prefix 分類（`feat:` → 新增功能、`refactor:`/`perf:` → 優化功能、`fix:` → 修復功能）輸出 Markdown 至 `changelog.md`，作為 release body。

# 版本號機制

- **Canonical base 來源**：`internal/version/version.go` 的 `Version` 字面值為 canonical base 版號（semver、無 `v` prefix、無後綴），目前為 `0.1.1`。Makefile 與 Release workflow 皆以 `sed` 讀取此字面值，不引入額外 VERSION 檔或以 Git tag 為來源。
- **develop 預發布**：推送 develop → Release workflow 產生預發布，二進位版號 `<base>-dev.<run_number>`、tag `v<base>-dev.<run_number>`、`prerelease: true`。`run_number` 取自 `github.run_number`，確保每次推送唯一、不撞 tag。
- **main 穩定發布**：推送 main → Release workflow 產生穩定發布，二進位版號 `<base>`、tag `v<base>`、`prerelease: false`。
- **晉升流程**：
  1. develop 累積預發布至可發布狀態。
  2. merge develop → main 並推送 main → Release workflow 自動產生穩定發布 `v<base>`。
  3. 執行 `byok-bump-version` skill 將 base 晉升到下一個 patch（或 minor/major），透過 Pull Request 合併至 develop。
  4. PR 合併至 develop 後，下一輪預發布使用更高的 base（如 `0.1.2-dev.N`），下一輪 main 發布即為 `0.1.2`。
- **bump skill**：`.github/skills/byok-bump-version/SKILL.md` 負責 bump + commit + push branch + 發 PR 到 develop；不建立 Git tag、不直接 push 到 develop 或 main、不強推。在 main 分支執行時中止。**發送 PR 前與合併前必須通知使用者確認，未經確認不得建立 PR 或合併。**

# 維護規則

任何改變以下項目的變更，**必須在相同變更內更新 `AGENTS.md` 對應段落**：

- 套件結構（新增/移除/重新命名套件、變更套件職責）
- BYOK 注入機制（環境變數名稱、`--config` 覆寫格式、子程序啟動方式）
- 設定檔格式（`~/.byok/config.yaml` 欄位、預設路徑）
- CLI 介面（指令、旗標、位置參數、錯誤訊息）
- 已記錄於「開發規範」的行為

## 分支保護與通知規則

- **版號晉升必須透過 Pull Request** — `byok-bump-version` skill 的變更必須透過 PR 合併至 develop，**禁止直接 push 到 develop 或 main**。
- **發送 PR 前必須通知使用者** — 建立任何 Pull Request 前（包含版號晉升），必須使用 `AskUserQuestion` 工具通知使用者 PR 內容（分支、目標、標題、摘要）並等待確認。未經確認不得建立 PR。
- **合併 PR 前必須通知使用者** — 合併任何 Pull Request 前（包含版號晉升），必須使用 `AskUserQuestion` 工具通知使用者 PR 狀態（PR 編號、CI 狀態）並等待確認。未經確認不得合併。

> ⚠ Spectra 區塊（`<!-- SPECTRA:START -->` 至 `<!-- SPECTRA:END -->`）由 Spectra CLI 自動管理，**不得手動編輯**。
