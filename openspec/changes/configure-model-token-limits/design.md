## Context

目前 `config.Profile.Models` 是 `[]string`，模型選擇、`config set-models` 與 `config list` 都直接處理字串。各 runner 收到選定模型後自行組裝 target-specific 環境變數或命令列；這個分層適合保留，但需要在 config/launch 邊界先解析出一致的 token 限制，再交給 runner 映射。

現況有兩個與 BYOK 模型能力無關的固定假設：Copilot runner 無條件注入 1,048,576 prompt tokens 與 131,072 output tokens；Claude runner 無條件為模型名稱補上 `[1m]`。這些值對自訂 gateway 模型未必成立。專案同時要求所有注入僅作用於子程序，不修改使用者的 Copilot、Codex、Claude 或 Pi 設定檔。

## Goals / Non-Goals

**Goals:**

- 讓 profile 能為每個候選模型保存 Context Window 與最大輸出 token，並提供 profile 級逐欄位預設值。
- 保留既有純字串 `models` YAML 的載入相容性，以及現有模型選擇、`--model` 覆寫與 `config set-models` 使用方式。
- 提供一次性 launch 旗標，並以一致、可測試的優先序產生有效 token 限制。
- 只對 target 支援的介面注入設定；不支援時提示但不阻止啟動。
- 讓 normal launch 與 dry-run 共用同一份解析結果。
- 移除 Copilot 與 Claude 的固定 token/context 假設，並完整更新 README 與 AGENTS.md。

**Non-Goals:**

- 不提供跨 target 的自動壓縮門檻、壓縮百分比、摘要保留量或手動 compact 控制。
- 不驗證 gateway 背後模型的真實容量，也不保證設定值能突破服務商上限。
- 不修改使用者的 `~/.codex/config.toml`、`~/.claude/settings.json`、`~/.pi/agent/models.json` 或父程序環境。
- 不新增 persistent config 編輯指令來設定單一模型限制；首版以 YAML 編輯持久化，launch 旗標提供一次性覆寫。
- 不為不同 target 建立不同的 profile model schema。

## Decisions

### 模型項目採用相容的 scalar-or-mapping YAML

新增 `Model` 與 `ModelLimits` 資料結構。profile 的標準資料形狀為：

```yaml
default_model_limits:
  context_window_tokens: 272000
  max_output_tokens: 16384
models:
  - name: gpt-5.4
    context_window_tokens: 1000000
    max_output_tokens: 128000
  - name: gpt-5.4-mini
    max_output_tokens: 32768
```

`context_window_tokens` 與 `max_output_tokens` 使用可選的 64-bit 正整數，以 nil 表示未設定。Model 的 YAML decoder 同時接受既有 scalar（例如 `- gpt-4o`）與新 mapping；scalar 在記憶體中正規化為只有 `Name` 的 Model。寫回時輸出 mapping 格式，legacy `default_model` 遷移也產生單一 Model。

載入驗證拒絕空白模型名稱、同一 profile 內重複名稱，以及零或負數 token。不同 target 對 context/output 的關係不一致，因此不加入 `max_output_tokens <= context_window_tokens` 的跨 target 驗證。未知 YAML 欄位延續現有 decoder 行為。

替代方案是保留 `models: []string` 並新增獨立 `model_limits` map；此方案被否決，因為模型移除或更名時容易留下孤兒設定，也違反模型資料聚合的需求。

### Token 限制逐欄位解析並保留來源

launch 層在模型名稱解析完成後產生 `ResolvedModel` 或等價值物件，包含選定名稱、兩個可選 token 值及各值來源。每個欄位獨立依下列順序解析：

1. 明確提供的 launch 旗標。
2. 選定模型物件的欄位。
3. profile 的 `default_model_limits` 欄位。
4. nil，表示 byok 不注入該參數。

`--model` 若與 profile 中某個 Model 名稱相符，使用該 Model 的個別限制；若不相符，仍依既有行為接受該模型名稱，但只使用 profile 預設與 token 旗標。這可避免臨時模型意外繼承另一個候選模型的限制。

`--context-window-tokens` 與 `--max-output-tokens` 必須是正整數。旗標解析需要保留「未提供」狀態，不能把 Cobra 的零值當成有效設定。錯誤值在讀取 API key、檢查 target executable 或啟動子程序前失敗。

`config set-models` 仍以名稱清單整批決定候選順序；建立新清單時，名稱仍存在的模型保留兩個個別限制，新名稱建立無個別限制的 Model，被移除名稱的 Model 與限制一併刪除。`config list` 與互動選單只顯示 Model 名稱。

### Target adapter 只注入官方支援的參數

runner 接收解析後的可選限制，不再自行產生預設值。映射如下：

| Target | `context_window_tokens` | `max_output_tokens` |
| --- | --- | --- |
| Copilot | `COPILOT_PROVIDER_MAX_PROMPT_TOKENS` | `COPILOT_PROVIDER_MAX_OUTPUT_TOKENS` |
| Codex | `--config model_context_window=<value>` | 不支援 |
| Codex App | `--config model_context_window=<value>` | 不支援 |
| Claude | `CLAUDE_CODE_MAX_CONTEXT_TOKENS` | `CLAUDE_CODE_MAX_OUTPUT_TOKENS` |
| Pi | `providers.openai.modelOverrides.<model>.contextWindow` | `providers.openai.modelOverrides.<model>.maxTokens` |

Copilot 的官方名稱是 maximum prompt tokens，而不是嚴格的 total context window；byok 直接映射，不做 `context - output` 換算，README 明確揭露這項語意差異。

Codex 與 Codex App 收到有效 `max_output_tokens` 時，launch 層向 stderr 寫出包含 target、參數、值與來源的 warning，runner 不加入對應設定且啟動流程繼續。Codex App 維持 `app` 為第一個子命令參數。沒有有效值時不顯示 warning。

Claude 的模型名稱原樣傳遞；byok 不新增也不移除 `[1m]`。需要 extended context 的使用者必須在模型名稱中明確提供 Claude Code 支援的名稱。由 byok 管理的 token 環境變數在建置子程序環境時去重，父程序環境保持不變。

若兩個限制皆未設定，各 runner 不產生任何 token override；其餘 BYOK 連線、telemetry、effort、sub-model、yolo 與 passthrough 行為維持不變。

### Pi 以暫存 model override 並從輸出量衍生 headroom

Pi 沿用 `PI_CODING_AGENT_DIR` 暫存目錄，不修改使用者設定。有效 token 值寫入暫存 `models.json` 的 `providers.openai.modelOverrides.<selected-model>`；只有存在有效值時才建立 model override 欄位。

Pi 的自動壓縮使用 `contextWindow - reserveTokens` 保留下一次回應空間。當 `max_output_tokens` 有效時，暫存目錄同時寫入 `settings.json`，令 `compaction.reserveTokens` 等於有效最大輸出值，以確保完整回應有對應 headroom。這是 Pi adapter 的安全衍生值，不是新的使用者壓縮設定。未設定最大輸出時不建立 byok 提供的 compaction 設定，交由 Pi 原生預設處理。

normal launch 結束後仍刪除整個暫存目錄；dry-run 只輸出建立、寫入、執行與清理片段，不真正建立檔案。

替代方案是完全不調整 Pi reserveTokens；此方案可能在 maxTokens 大於 Pi 預設 reserve 時太晚壓縮，因此不採用。

### Normal launch 與 dry-run 共用有效限制契約

profile/model/CLI 的 token 解析只實作一次，normal launch 與 dry-run 都接收相同的 resolved limits。dry-run 必須呈現 target 實際會收到的環境變數、Codex config、Pi `models.json` 與可選 `settings.json`，同時維持 API key 和 telemetry header masking。

不支援 warning 在 normal launch 與 dry-run 都寫入 stderr；dry-run stdout 保持可執行的等效命令，不混入說明文字。這讓腳本可以分別處理 command output 與相容性提示。

### Context 容量與壓縮策略保持分離

`context_window_tokens` 表示傳給 target 的模型容量或最接近的官方能力欄位，不代表 byok 自己監控對話 token 或執行摘要。Copilot、Codex、Claude 與 Pi 的原生自動壓縮機制維持啟用且不被統一覆寫；除 Pi 由最大輸出衍生 response headroom 外，本變更不注入 compact threshold。

未來若需要控制壓縮，必須另行設計 target-specific options，因為 Codex 使用絕對 token threshold、Claude 使用 effective window/percentage、Pi 使用 reserveTokens，而 Copilot 沒有公開可調門檻。

## Implementation Contract

**Observable behavior**

- 舊的 scalar `models` config 可正常載入、選擇與啟動；新 mapping 可提供模型個別限制，profile 可提供逐欄位預設。
- token 旗標、模型值與 profile 預設按照固定優先序解析；未設定欄位不產生 byok token override。
- supported target 收到表格指定的值；Codex/Codex App 對最大輸出顯示 warning 後繼續。
- Claude 收到完全相同的模型字串，不再自動取得 `[1m]`。
- Pi 的暫存設定在程序結束後移除，使用者設定檔與父程序環境不變。
- dry-run stdout 與 normal launch 映射一致，且所有 secret 仍為 `***`。

**Interface and data shape**

- Profile YAML 新增選填 `default_model_limits.context_window_tokens` 與 `default_model_limits.max_output_tokens`。
- `models` 每項接受 scalar name 或包含 `name`、`context_window_tokens`、`max_output_tokens` 的 mapping。
- `byok launch <target>` 新增 `--context-window-tokens <positive-int>` 與 `--max-output-tokens <positive-int>`。
- runner contract 接收已解析的可選 token limits；runner 不決定設定優先序或跨 target 預設值。

**Failure modes**

- config 中空白/重複模型名稱或非正 token 值使 config load 失敗，錯誤指出 profile、模型或欄位。
- 非正 CLI token 值使 launch exit 1，且不讀 API key、不檢查 executable、不啟動子程序。
- 不支援的 target/參數組合不是失敗：stderr warning 後忽略該參數。
- target 自身拒絕超過實際模型容量的數值時，維持既有子程序錯誤傳遞；byok 不宣稱本地設定能擴張模型能力。

**Acceptance criteria**

- config tests 覆蓋 scalar/mapping 載入、canonical 寫回、legacy `default_model` 遷移、重複名稱及 token 邊界。
- launch tests 以 CLI、模型、profile 預設和 nil 組合驗證逐欄位解析與陌生 `--model` 行為。
- 每個 runner 的單元測試驗證 supported mapping、unset omission、環境去重、Claude 模型原樣傳遞及 Codex warning。
- Pi tests 解析產生的 JSON，驗證 modelOverrides、settings headroom、secret 權限與清理。
- dry-run tests 驗證 Windows/POSIX 輸出、warning channel、Pi 暫存片段與 secret masking。
- `go test ./...`、`go vet ./...` 與 Spectra validation 全部通過。

**Scope boundaries**

- In scope: config schema、遷移/驗證、launch 解析、五個 target mapping、dry-run、文件與相關測試。
- Out of scope: 自動探測模型 metadata、服務商能力查詢、持久化 config 編輯旗標、統一壓縮門檻與修改使用者 target 設定檔。

## Risks / Trade-offs

- [Risk] 新 mapping 寫回後，舊版 byok 無法把該 `models` 項目解析為字串 → 新版可讀舊格式，README 記錄降版時需將 mapping 改回 scalar；launch 本身不自動寫回 config。
- [Risk] 使用者把 Context Window 設得高於 gateway 實際容量 → README 明確說明設定是能力宣告，保留 target 的錯誤或 clamp 行為。
- [Risk] Copilot 的 prompt-token 欄位與通用 context 名稱不完全相同 → 不做隱含算術，支援矩陣直接揭露映射。
- [Risk] Pi 同步 reserveTokens 會比原生策略更早或更晚壓縮 → 只在使用者明確設定最大輸出時衍生相同 token 數；未設定時保留 Pi 預設。
- [Risk] `config set-models` 整批更新造成限制遺失 → 以名稱 join 保留仍存在項目的限制並加入回歸測試。
- [Risk] normal/dry-run 分別組裝造成漂移 → 兩條路徑共用 resolved limits，並以 target mapping parity tests 驗證。

## Migration Plan

1. 發布新版後，既有 scalar `models` 與 legacy `default_model` 可直接載入，不需要預先改檔。
2. 使用者可逐步把需要個別限制的項目改成 mapping，或只新增 profile 的 `default_model_limits`。
3. 任何會儲存 config 的新版命令將模型寫成 mapping；文件提供新舊格式與降版範例。
4. 回滾至舊版前，先移除 `default_model_limits`，並將 mapping 模型還原為 scalar 名稱；API key/keychain 與其他 profile 欄位不需遷移。

## Open Questions

無。壓縮門檻已明確排除，Pi response headroom 是由有效最大輸出值衍生的 adapter 行為。
