## Problem

在 Windows 上執行 `byok update` 進行自我更新時，更新流程在原子替換執行檔步驟失敗，錯誤訊息為「更新失敗: Access is denied.」。此問題在 dev channel 與 stable channel 皆會發生（已確認 0.2.0-dev.6 → 0.2.1-dev.7 失敗）。

根本原因：Windows 不允許覆寫正在執行中的執行檔。`byok update` 是由 `byok.exe` 本身執行，因此 `byok.exe` 檔案被作業系統鎖定。目前 Windows 平台的 `renameReplace` 函式使用 `MoveFileExW` + `MOVEFILE_REPLACE_EXISTING` 旗標，當目標檔案為執行中的執行檔時，此 API 回傳 `ERROR_ACCESS_DENIED (5)`，導致替換失敗。

## Root Cause

`internal/updater/replace_windows.go` 的 `renameReplace` 函式直接以 `MoveFileExW` 搭配 `MOVEFILE_REPLACE_EXISTING` 嘗試覆寫目標執行檔。Windows 對執行中 (.exe) 檔案施加獨占鎖，任何嘗試覆寫該檔案的操作皆回傳 `ERROR_ACCESS_DENIED`。此鎖定會持續到進程結束，因此自我更新場景必定觸發此錯誤。

## Proposed Solution

採用 Windows 自我更新的標準模式：**rename-then-move**。

1. 先將執行中的執行檔重新命名為備份檔（如 `byok.exe` → `byok.exe.old`）— Windows 允許重新命名執行中的 .exe 檔案。
2. 將暫存的新二進位檔重新命名/移動至原始路徑（`temp` → `byok.exe`）— 此時原始路徑已釋放，操作成功。
3. 嘗試刪除備份檔；若因檔案仍被鎖定而無法刪除，則以 `MoveFileExW` + `MOVEFILE_DELAY_UNTIL_REBOOT` 旗標排程於下次開機時刪除，確保不殘留廢棄檔案。

此模式不改變 `byok update` 的使用者介面或 CLI 行為，僅修正 Windows 平台的檔案替換實作。Unix 平台的 `renameReplace` 維持不變（`os.Rename` 在 Unix 上可覆寫執行中檔案）。

## Non-Goals

- 不引入第三方 Go 相依（如 `golang.org/x/sys`），繼續使用 `syscall` 呼叫 Windows API。
- 不改變 `byok update` 的 CLI 介面、旗標或輸出格式。
- 不處理跨磁碟區移動場景（暫存檔與目標已在同目錄，必為同磁碟區）。
- 不修改 Unix 平台的替換邏輯。

## Success Criteria

1. 在 Windows 上 `byok update` 能成功將執行中的 `byok.exe` 更新至新版本，不再出現 "Access is denied" 錯誤。
2. 更新完成後，`byok.exe` 檔案內容為新版本二進位，可正常執行。
3. 原執行檔以備份檔名（如 `byok.exe.old`）殘留時，應被排程於下次開機刪除或立即刪除成功。
4. Unix 平台行維持不變，現有測試全部通過。
5. 新增的 Windows 替換測試以 `go test ./internal/updater/ -run TestRenameReplace -race` 通過。

## Impact

- Affected specs:
  - Modified: `byok-self-update` — 補充 Windows 自我更新替換執行檔的需求情境（執行中執行檔的 rename-then-move 模式與備份檔清理）。
- Affected code:
  - Modified: `internal/updater/replace_windows.go` — 改寫 `renameReplace` 為 rename-then-move 模式，新增備份檔排程刪除邏輯。
  - Modified: `internal/updater/updater_test.go` — 新增 Windows 平台替換執行中檔案的測試案例。
