## Context

`byok` 目前以 `cmd/launch.go` 中寫死的 `copilot` 目標工具實作 BYOK 啟動：解析 profile、驗證 provider、`exec.LookPath("copilot")`，再以 `internal/runner.Launch` 注入四個 `COPILOT_*` 環境變數啟動子程序。Codex CLI 的 BYOK 模型與 Copilot 不同——Codex 由 `~/.codex/config.toml` 的 `model_provider` + `[model_providers.<id>]`（含 `base_url` 與 `env_key`）決定連線，API key 從 `env_key` 指名的環境變數讀取，且 `--config` 旗標可對單次執行做最高優先序的 TOML 覆寫。專案另有兩條 GitHub Actions workflow（`release.yml`、`pr-test.yml`）使用 `umahmood/pushover-actions` 發送通知，目前成功與失敗皆以 `priority: '1'`、`sound: 'pushover'` 發送，無法從通知音效區分結果。

## Goals / Non-Goals

**Goals:**

- 將 `byok launch` 的第一個位置參數正式化為目標工具選擇器，分派 `copilot` 與 `codex`，且不改變 copilot 既有行為。
- 以「不寫入使用者設定檔」的方式實作 `byok launch codex`：API key 透過子程序環境變數承載，其餘連線設定透過 codex `--config` 旗標覆寫。
- 讓 codex 啟動支援與 copilot 對等的 `--model`、`--profile`、`--config`、`-y`/`--yolo`、`--` 透傳。
- 讓 Pushover 通知依成功/失敗使用不同優先權層級與音效。

**Non-Goals:**

- 不修改 `~/.codex/config.toml` 或 `~/.byok/config.yaml`。
- 不支援 `openai` 以外的 provider 類型。
- 不為 codex 產生持久化或臨時的 codex profile 檔。
- 不改變 copilot 的環境變數注入機制。
- 不改變版本號管理或 Release 產物結構。

## Decisions

### Decision: 目標工具分派置於 cmd 層，runner 保持工具特定建置函式

`cmd/launch.go` 的 `RunE` 解析第一個位置參數後分派至 `runLaunchCopilot` 或新增的 `runLaunchCodex`；兩者各自負責 profile 解析、provider 驗證、`exec.LookPath` 與錯誤訊息。`internal/runner` 新增 `codex.go` 提供建置 codex 子程序環境與 `--config` 參數的函式，與既有 `runner.go` 的 `BuildEnv`（copilot）並列。不強行抽出一個泛用 `Launch(profile, target, ...)`，因為 copilot 與 codex 的注入方式（環境變數 vs `--config` 旗標）本質不同，強抽像會污染兩條路徑。

### Decision: Codex BYOK 採環境變數承載 API key + `--config` 旗標覆寫連線設定

採方案 A。具體做法：byok 在 codex 子程序環境中設定一個內部環境變數 `BYOK_CODEX_API_KEY`（值為 profile 的 `api_key`），並以 codex `--config` 旗標覆寫：

- `model_provider="byok"`（自訂 provider id，避開保留字 `openai`/`ollama`/`lmstudio`）
- `model_providers.byok.base_url="<profile.api_base>"`
- `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`
- `model="<model>"`（`--model` 有指定則覆寫 profile 的 `default_model`）

`--config` 覆寫為單次執行、最高優先序，不寫入任何檔案；子程序結束後不留痕跡，符合 copilot 既有「暫時注入」理念。`BYOK_CODEX_API_KEY` 僅存在於 codex 子程序環境，父程序與 shell 不受影響。

**Alternatives considered:**

- 方案 B（臨時 codex profile 檔 + `--profile`）：會在 `~/.codex/` 寫入新檔並需清理；當機留下殘留檔。違反「不動原設定」原則，故摒棄。
- 直接設定 codex 既有環境變數（如 `OPENAI_API_KEY` + `openai_base_url`）：只能覆寫 built-in openai provider 的 base_url，無法處理非 OpenAI 相容端點的命名 provider，彈性不足。

### Decision: `-y`/`--yolo` 對 codex 透傳為 `--yolo` 旗標

依 Codex 官方文件（[Sandbox & approvals](https://developers.openai.com/codex/agent-approvals-security) 之「Common sandbox and approval combinations」表），`--dangerously-bypass-approvals-and-sandbox`（alias: `--yolo`）為 Codex 唯一的「no sandbox / no approvals / full access」模式，語意等同於「allow all」。其效果與 copilot 的 `--yolo` 並不相同（codex 同時關閉 sandbox 與 approvals；copilot 僅關閉 approvals），但兩者皆以 `--yolo` 為 CLI 旗標名稱，故 byok 的 `-y`/`--yolo` 在 codex 路徑直接附加 `--yolo` 至 codex 命令列即可，不需組合 `--sandbox danger-full-access --ask-for-approval never`；`--` 之後的透傳參數接在其後，順序與 copilot 一致（yolo 旗標在前，透傳在後）。

### Decision: Pushover 通知以 job/needs 結果決定 priority 與 sound

成功時 `priority: '0'`（無聲/低打擾）搭配 `sound: 'pushover'`（或正向音）；失敗時 `priority: '1'`（高優先）搭配警示音效（如 `falling` 或 `alarm`）。緊急優先權（`2`）需 Pushover 額外參數且會強制打擾，本變更不使用。在 `release.yml` 的 `notify` job 以 `needs.build.result` 與 `needs.release.result` 推導整體狀態；在 `pr-test.yml` 以 `job.status` 推導。兩處皆以 ternary 表達式依狀態選取 `priority` 與 `sound`。

### Decision: AGENTS.md 作為架構與規範的權威文件並隨變更同步

`AGENTS.md` 目前僅含 Spectra 指令區塊（`<!-- SPECTRA:START -->`…`<!-- SPECTRA:END -->`，由 CLI 自動維護）。本變更在其後新增「專案架構」與「開發規範」兩個區塊，內容取自現有原始碼事實（Go 1.26+ cobra CLI、模組路徑 `github.com/IISI-2209026/LlmByok`、`cmd`/`internal/config`/`internal/runner`/`internal/version` 套件職責、設定檔 `~/.byok/config.yaml`、BYOK 注入僅作用於子程序且不寫入使用者設定檔、profile 解析與錯誤處理、測試約定）。此區塊由人工維護（非自動產生），並明訂維護規則：任何改變套件結構、BYOK 注入機制、設定檔格式、CLI 介面或開發規範的變更，須在相同變更內同步更新 `AGENTS.md` 對應段落。Spectra 區塊仍由 CLI 管理，不手動編輯。

## Implementation Contract

**Behavior:**

- `byok launch copilot ...` 行為與現行完全一致。
- `byok launch codil`（拼錯）→ 印出 `錯誤：不支援的工具 "codil"（目前支援 copilot、codex）` 並 exit 1。
- `byok launch codex`（無 profile/設定檔）→ 與 copilot 對等的錯誤訊息（找不到設定檔、找不到 profile、provider 非 openai）並 exit 1。
- `byok launch codex` 成功啟動時，codex 子程序環境包含 `BYOK_CODEX_API_KEY=<api_key>`，命令列包含 `--config model='"<model>"'`、`--config model_provider='"byok"'`、`--config model_providers.byok.base_url='"<api_base>"'`、`--config model_providers.byok.env_key='"BYOK_CODEX_API_KEY"'`；`-y` 時附加 `--yolo`，`--` 後參數原樣附加在後。
- 父程序 `byok` 與 shell 環境於啟動前後完全不變；`~/.codex/config.toml` 不被讀寫修改（codex 仍會自行載入使用者既有 config，byok 的 `--config` 覆寫優先序最高）。
- Release workflow 失敗時 Pushover 通知音效為警示音、優先權為高；成功時為低打擾音效。PR Tests workflow 同理。

**Interface / data shape:**

- 新增 `internal/runner.BuildCodexArgs(profile, modelOverride) (env []string, configArgs []string)`：`env` 含 `BYOK_CODEX_API_KEY=<api_key>` 與其他現有環境（過濾掉既存 `BYOK_CODEX_API_KEY`）；`configArgs` 為上述 `--config` 旗標切片。
- 新增 `internal/runner.LaunchCodex(profile, modelOverride, exePath, extraArgs, stdin, stdout, stderr) error`：以 `BuildCodexArgs` 組裝 `codex [--config ...] [--yolo] [passthrough...]` 並啟動。
- `cmd/launch.go` 的 `newLaunchCmd` 更新 `Use`/`Long` 說明與目標分派；新增 `runLaunchCodex`。
- workflow YAML：以 `${{ <status> == 'success' && '0' || '1' }}` 形式選取 priority，sound 同理。

**Failure modes:**

- 設定檔不存在、profile 不存在、provider 非 openai、`codex` 不在 PATH：皆印出對應錯誤訊息並 exit 1，與 copilot 路徑一致。
- codex 子程序非零結束：靜默傳遞 exit code，不額外印訊息（與 copilot 一致）。
- 使用者既有 `[model_providers.byok]` 衝突：因 byok 以 `--config` 覆寫該 table 的 `base_url`/`env_key` 鍵，會覆寫同名鍵； provider id 採 `byok` 已避開保留字，若使用者已自訂 `byok` provider 則其鍵被覆寫（可接受，屬罕見情形）。

**Acceptance criteria:**

- `go test ./...` 通過，包含新增的 `cmd/launch_codex_test.go`、`internal/runner/codex_test.go`，以及更新後的 `cmd/launch_test.go`（目標分派）。
- `byok launch codex --profile <p>` 以 stub codex 驗證命令列含正確 `--config` 旗標與 `BYOK_CODEX_API_KEY` 環境變數（整合測試，仿照 `launch_integration_test.go` 的 stub 架構）。
- `byok launch copilot` 既有測試全數通過且無行為退化。
- workflow YAML 中 pushover 步驟的 `priority` 與 `sound` 可由靜態檢視確認依成功/失敗分支。
- `AGENTS.md` 在既有 Spectra 區塊後新增「專案架構」與「開發規範」區塊，內容與現有原始碼事實一致，並含明確的同步維護規則。

**Scope boundaries:**

- In scope：`byok launch` 目標分派、codex 啟動、README 更新、兩條 workflow 的 pushover priority/sound、AGENTS.md 架構與規範區塊。
- Out of scope：其他 provider 類型、codex profile 檔管理、copilot 環境變數機制變更、版本/Release 產物結構變更、新增其他目標工具。

## Risks / Trade-offs

- [Codex `--config` 對 `model_providers.<id>.<key>` 的 dot-notation 支援] → 依官方文件 `--config` 支援 dot notation 與 TOML 值；實作時以整合測試對 stub 驗證實際產出的命令列格式，避免依賴未驗證假設。
- [provider id `byok` 與使用者自訂 provider 衝突] → 採用不易衝突的固定 id 並在 README 說明；使用者自訂 `byok` provider 的鍵會被覆寫（罕見，可接受）。
- [Pushover 音效名稱因帳號設定而異] → 採 Pushover 通用內建音效名稱（如 `pushover`、`falling`），並在 workflow 註解標示可自訂。
- [codex CLI 未來版本變更 `--yolo` 或 `--config` 行為] → 以 stub 整合測試驗證 byok 產出的命令列，而非依賴真實 codex；CLI 變更時僅需調整 byok 對應邏輯。
