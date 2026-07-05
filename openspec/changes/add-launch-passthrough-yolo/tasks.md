## 1. runner 層：Launch 簽名變更與整合測試

- [x] 1.1 修改 `internal/runner/runner.go` 的 `Launch` 函式簽名，新增 `extraArgs []string` 參數，將 extraArgs 透過 `exec.Command(exePath, extraArgs...)` 傳給 copilot 子行程（對應 Decision: extraArgs 在 runner.Launch 中組合）。行為：runner.Launch 接收 extraArgs 並原樣轉發，不做事實判斷。驗證：`go build ./internal/runner/` 通過。
- [x] 1.2 更新 `internal/runner/launch_integration_test.go`：呼叫 `Launch` 時傳入 extraArgs，驗證 stub 輸出包含傳入的參數、且父行程環境變數不變（對應 MODIFIED 規格「Launch Copilot with BYOK profile」的環境隔離保證）。行為：傳入 `["--yolo"]` 時 stub 收到 `--yolo`；父行程環境不受影響。驗證：`go test ./internal/runner/` 全綠。

## 2. cmd 層：新增 yolo 旗標與透傳機制

- [x] 2.1 在 `cmd/launch.go` 新增 `--yolo` / `-y` 布林旗標（對應 Decision: yolo 旗標以布林 flag 實作），並使用 cobra 原生 `--` 分隔符接收透傳參數（對應 Decision: 以 cobra Args 解析透傳參數）；`RunE` 將 yolo 旗標為 true 時的 `--yolo` 與 `cmd.Args()` 透傳參數組合為 extraArgs（yolo 在前、透傳在後）傳給 `runner.Launch`。行為：`byok launch copilot -y` → copilot 收到 `--yolo`；`byok launch copilot -- --continue` → copilot 收到 `--continue`；`byok launch copilot -y -- --continue` → copilot 收到 `--yolo --continue`；無旗標時 copilot 收到零參數（對應規格「YOLO mode flag」與「Argument passthrough via double dash」）。驗證：`go build ./...` 通過。
- [x] 2.2 在 `cmd/launch_test.go` 新增測試案例涵蓋：`--yolo` 附加 `--yolo`、`-y` 短形式、`--` 透傳單一參數、`--` 透傳多參數、`-y` 與 `--` 合用時順序為 `--yolo` 在前、無旗標無透傳時行為不變（對應規格「YOLO mode flag」與「Argument passthrough via double dash」的所有情境）。行為：測試斷言 copilot 收到的參數符合預期順序。驗證：`go test ./cmd/` 全綠。

## 3. 文件更新

- [x] 3.1 更新 `README.md` 使用說明段落，補充 `-y` / `--yolo` 旗標與 `--` 透傳用法範例（含組合使用情境）。行為：README 涵蓋新旗標說明與範例指令。驗證：人工審閱 README 內容涵蓋 `byok launch copilot -y`、`byok launch copilot -- <args>`、組合使用三種範例。
