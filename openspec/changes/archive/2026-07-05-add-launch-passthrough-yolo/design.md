## Context

`byok launch copilot` 目前透過 `runner.Launch` 以 BYOK 環境變數啟動 copilot 子行程，但無法傳遞任何額外參數給 copilot。使用者希望能在 BYOK 模式下使用 copilot 的 `--yolo`、`--continue` 等選項，而不必放棄 BYOK 暫時切換金鑰的便利性。

現況：
- `cmd/launch.go` 的 `runLaunchCopilot` 接收 `--model`、`--profile`、`--config` 三個旗標，呼叫 `runner.Launch(profile, modelOverride, exePath, stdin, stdout, stderr)`。
- `runner.Launch` 使用 `exec.Command(exePath)` 建立子行程，不傳遞任何參數。

## Goals / Non-Goals

**Goals:**

- 讓使用者能透過 `-y`/`--yolo` 快速啟用 copilot 的 yolo 模式
- 讓使用者能透過 `--` 透傳任意參數給 copilot
- 維持 BYOK 環境變數注入邏輯不變
- 維持父行程環境不受影響的隔離保證

**Non-Goals:**

- 不支援其他 CLI 工具（Codex CLI、Claude CLI）
- 不解析或驗證透傳參數內容，由 copilot 自行處理
- 不改變 BYOK 環境變數注入邏輯
- 不改變設定檔格式或 profile 結構

## Decisions

### Decision: 以 cobra Args 解析透傳參數

使用 cobra 的 `Args: cobra.ArbitraryArgs`（或 `cobra.MinimumNArgs(0)`）搭配 `cmd.Args()` 取得 `--` 之後的參數。cobra 預設會將 `--` 之後的位置參數放入 `Args` 切片，不需手動解析。

**理由**：cobra 原生支援 `--` 分隔符，手動解析會增加複雜度且容易出錯。

### Decision: yolo 旗標以布林 flag 實作

新增 `--yolo` / `-y` 布林旗標，預設 `false`。當為 `true` 時，在傳給 copilot 的參數尾端附加 `--yolo` 字串。

**理由**：布林旗標最直覺，與 copilot 本身的 `--yolo` 選項語意一致。

### Decision: extraArgs 在 runner.Launch 中組合

`runner.Launch` 新增 `extraArgs []string` 參數，直接傳給 `exec.Command(exePath, extraArgs...)`。cmd 層負責組合 yolo 旗標與透傳參數為單一切片，runner 層不做事實判斷。

**理由**：保持 runner 層職責單一（注入環境變數 + 啟動行程），參數組合邏輯集中在 cmd 層。

## Implementation Contract

**行為：**
- `byok launch copilot -y` → copilot 收到 `--yolo` 參數
- `byok launch copilot --yolo` → 同上
- `byok launch copilot -- --continue` → copilot 收到 `--continue` 參數
- `byok launch copilot -y -- --continue --model x` → copilot 收到 `--yolo --continue --model x`
- 不加任何旗標時行為與現況完全相同（零參數傳給 copilot）

**介面變更：**
- `cmd/launch.go`：`runLaunchCopilot` 新增 `args []string`（透傳參數）與 `yolo bool` 輸入；cobra command 新增 `--yolo`/`-y` 旗標；RunE 從 `cmd.Flags().GetBool("yolo")` 與 `cmd.Args()` 取值後傳入
- `internal/runner/runner.go`：`Launch` 簽名變更為 `Launch(profile *config.Profile, modelOverride string, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer) (int, error)`
- `cmd/launch_test.go`：新增測試案例涵蓋 yolo 與透傳
- `internal/runner/launch_integration_test.go`：更新 `Launch` 呼叫簽名並驗證 extraArgs 出現在 stub 輸出

**失敗模式：**
- 透傳參數為空且未加 yolo → 行為不變，copilot 收到零參數
- yolo 與透傳同時使用 → yolo 參數在前，透傳參數在後

**驗收條件：**
- `go build ./...` 通過
- `go test ./...` 全綠
- 新增測試驗證：yolo 模式下 stub 收到 `--yolo`；透傳模式下 stub 收到指定參數；兩者合用時順序正確

**範圍邊界：**
- In scope：cmd/launch.go、internal/runner/runner.go、對應測試
- Out of scope：config 相關檔案、README（除非需要更新使用說明）、其他子指令

## Risks / Trade-offs

- [參數衝突] 透傳參數可能與 copilot 的 flag 衝突 → 由 copilot 自行報錯，byok 不介入
- [簽名變更] `runner.Launch` 簽名改變會影響所有呼叫端 → 僅 launch.go 與測試呼叫，影響面可控
