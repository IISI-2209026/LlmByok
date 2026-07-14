## Context

byok currently resolves a profile and selected primary model before a target runner starts the child process. Each runner configures its target using a transient native surface: Copilot combines command arguments and environment variables, Codex uses command-line `--config` overrides, Claude uses environment variables, and pi uses command arguments plus a temporary `PI_CODING_AGENT_DIR`. Optional effort and sub-model settings cross the common Cobra command and target runners. The new dry-run flow must derive the same launch inputs without starting a target process or exposing API keys.

## Goals / Non-Goals

**目標：**

- 為 `byok launch <target>` 新增選填 `--effort <level>`、`--sub-model <model>` 與 `--dry-run` 旗標。
- 任一旗標未指定時，保留各 target 的原生預設行為。
- 在啟動子程序前驗證 effort，並於值無效時列出該 target 支援的值。
- 只將 `--sub-model` 套用至 Claude 子程序環境；其餘 target 對此旗標維持 no-op。
- 讓 `--dry-run` 僅輸出平台 shell 可執行的等效命令，不啟動 target、不檢查 target PATH，且不讀取 API key。
- 在 dry-run 輸出以 `***` placeholder 取代 API key，並以目標 shell 正確引用 placeholder、profile 值與所有參數。
- 在 README.md 與 AGENTS.md 說明三個旗標及其 target-specific mapping。

**非目標：**

- 不在 BYOK YAML profile 或目標 CLI 設定檔持久化 effort 或 subagent model 偏好。
- 不變更主模型選擇、API key 解析、provider 選擇、yolo 行為或 passthrough 參數語意。
- 不和遠端 provider 協商模型支援能力、不驗證 sub-model 識別字，且不靜默降低使用者指定的 effort。
- 不在 dry-run 讀取 keychain、設定檔明碼 API key 或產生可直接存取秘密的命令。
- 不為不支援特定設定的已安裝 target CLI 版本建立相容性 fallback，也不提供非 Windows/POSIX shell 的第三種輸出格式。

## Decisions

### 使用選填的 launch effort、sub-model 與 dry-run 旗標

`byok launch <target> --effort <level> --sub-model <model> --dry-run` 是公開介面。三個旗標皆為選填，未指定時在內部以空值表示。未指定 effort 時不輸出 effort-specific 設定；未指定 sub-model 時不輸出 subagent-model 設定。`--dry-run` 改變執行模式而非輸出內容的邏輯來源：它使用相同的 profile、model、effort、sub-model、yolo 與 passthrough 解析結果。profile-level 欄位會持久化偏好並造成優先順序歧義，因此不納入本變更。

### 在啟動前依 target 驗證 effort 值域

共用驗證 helper 接收 target 與請求的 effort。Copilot 接受 `none`、`minimal`、`low`、`medium`、`high`、`xhigh`、`max`；Codex 接受 `none`、`minimal`、`low`、`medium`、`high`、`xhigh`、`max`；Claude 接受 `low`、`medium`、`high`、`xhigh`、`max`；pi 接受 `off`、`minimal`、`low`、`medium`、`high`、`xhigh`、`max`。未知或不相容的 effort 必須在 normal launch 或 dry-run 輸出前中止，並列出 target 與有效值。sub-model 為 opaque value，不做驗證，因為僅 Claude 使用它，模型名稱是否合法由 Claude 或其 gateway 決定。

### 以原生暫時性介面轉換設定

Copilot 在 yolo 與 passthrough 前接收 `--reasoning-effort <level>`。Codex 與 Codex App 在既有 BYOK override 旁接收頂層 `--config model_reasoning_effort="<level>"`；Codex App 仍以 `app` 為第一個參數。Claude 在指定 effort 時只於子程序環境設定 `CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1` 與 `CLAUDE_CODE_EFFORT_LEVEL=<level>`，在指定 sub-model 時只於子程序環境設定 `CLAUDE_CODE_SUBAGENT_MODEL=<model>`。pi 在 `--model <model>` 後、passthrough 前接收 `--thinking <level>`。Copilot、Codex、Codex App、pi 的非空 sub-model 必須刻意忽略，且不得變更子程序參數或環境。

### 產生平台原生且不含 API key 的 dry-run 命令

dry-run renderer 在 Windows 產生可貼到 PowerShell 的命令，在其他支援平台產生可貼到 POSIX shell 的命令。Copilot、Codex 與 Claude 以該 shell 的環境變數設定語法搭配 target command 呈現，並以正確 shell quoting 保留 base URL、模型、effort、sub-model 與 passthrough。API key 的位置必須一律為已引用的 `***`，且 renderer 不得呼叫 key resolver。pi 因為需要 `models.json`，renderer 必須輸出完整多行 shell 片段：建立唯一暫存目錄、以 `***` 寫入包含 API base 的 models.json、執行含模型與選填 thinking 的 pi、並在成功或失敗後清理暫存目錄。dry-run 不得以 `exec.LookPath` 檢查 target 的安裝狀態，讓使用者可在不同電腦或 PATH 尚未準備好時檢視命令。

### 保持 runner 與 renderer contract 明確

各 runner 與 renderer 分別接收已解析的 model、effort、sub-model、extraArgs 與 profile connection data。test double 與 renderer tests 必須斷言子程序參數順序、環境 entry、shell quoting、key masking 及 pi cleanup 片段。這避免將旗標視為非結構化 passthrough input，並避免使用者透過 `--` 的參數覆寫 byok 管理的設定。

## Implementation Contract

**行為：** `byok launch claude --sub-model claude-haiku-4-5` 啟動 Claude 時，僅在子程序帶入 `CLAUDE_CODE_SUBAGENT_MODEL=claude-haiku-4-5`。相同 `--sub-model` 指定給 Copilot、Codex、Codex App 或 pi 時，會啟動該 target，但不會帶入 sub-model 參數或環境設定，也不會報錯。`byok launch codex --model gpt-5 --effort high --dry-run` 只輸出包含 Codex BYOK config override、`model_reasoning_effort` 和被遮罩 key 的命令，不會執行 Codex。Windows 輸出 PowerShell；其他平台輸出 POSIX shell。

**介面與 mapping：** launch command 接受 `--effort <level>`、`--sub-model <model>` 與 `--dry-run`。Copilot 僅將 effort 映射為 `--reasoning-effort`；Codex 與 Codex App 僅將 effort 映射為 `--config model_reasoning_effort`；Claude 將 effort 映射為 `CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1` 加上 `CLAUDE_CODE_EFFORT_LEVEL`，並將 sub-model 映射為 `CLAUDE_CODE_SUBAGENT_MODEL`；pi 僅將 effort 映射為 `--thinking`。dry-run 將上述映射渲染為平台 shell 命令，任何 API key 位置均為引用的 `***`。

**失敗模式：** 非空 effort 若不在所選 target 的支援值域內，印出包含 target、請求值與有效值的錯誤，以 exit code 1 結束，且不得啟動 target 或輸出命令。非空 sub-model 不得造成非 Claude target 失敗，也不得為該 target 產生 override。dry-run 仍須驗證 config、profile、provider 與模型解析，但不得解析 API key 或檢查 target executable。未指定任一旗標時，不產生驗證錯誤或 override。

**驗收條件：** command 與 runner tests 覆蓋有效 effort 注入、有效 Claude sub-model 注入、四個非 Claude target 的 sub-model no-op、精確參數順序、旗標省略與無效 effort 在未執行時遭拒絕。dry-run tests 覆蓋 Windows PowerShell、POSIX shell、五個 target、API key `***` masking、shell quoting、pi 暫存檔與 cleanup、無 key resolver 呼叫及未安裝 target 時仍可輸出。`go test ./... -race` 通過。README.md 與 AGENTS.md 說明三個公開旗標及 mappings。

**範圍邊界：** 本變更修改 launch parsing、runner injection、命令渲染、tests、specifications 與 documentation；不修改 profile YAML schema、不寫入 target configuration files、不驗證 sub-model identifier，也不支援 fallback 或 downgrade。

## Risks / Trade-offs

- [各原生 CLI 版本可接受的 effort level 可能不同] → 將 target allowlist 寫入文件與測試，並於啟動前拒絕不支援值。
- [provider-backed custom model 可能拒絕原生 CLI 有效的 effort 或 sub-model] → 保留 target CLI 的原始錯誤，不猜測替代值。
- [參數順序或 shell quoting 可能改變 CLI parsing] → 使用既有 stub integration tests 與 renderer unit tests 斷言 argument sequences 與平台輸出，特別覆蓋 Codex App 與 pi。
- [使用者複製 dry-run 輸出時忘記替換 placeholder] → 固定使用明顯的 `***` 值，並在 README 清楚要求先替換。
- [pi 需要設定檔才能等效啟動] → 輸出建立、使用與清理專屬暫存目錄的完整 shell 片段，而非宣稱單一 pi 命令即可完成。
- [使用者可向非 Claude target 傳遞 sub-model] → 將 no-op 定義為可測試的刻意行為，不以錯誤阻擋可攜啟動指令。

## Migration Plan

本變更為 additive，不需要設定檔遷移。未指定旗標的既有 invocation 維持不變。回復本變更時，只會移除選填旗標，不會留下持久化資料或被修改的 target configuration files。

## Open Questions

無。
