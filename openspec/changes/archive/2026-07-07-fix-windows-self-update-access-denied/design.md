## Context

`byok update` 在 Windows 上執行自我更新時失敗，錯誤為 `ERROR_ACCESS_DENIED`。目前 Windows 平台的 `renameReplace` 函式以 `MoveFileExW` + `MOVEFILE_REPLACE_EXISTING` 直接覆寫目標執行檔，但 Windows 對執行中的 `.exe` 檔案施加獨占鎖，導致覆寫操作被拒絕。此問題在所有 Windows 平台的自我更新場景必定發生，因為 `byok.exe` 本身就是執行 `byok update` 的進程。

## Goals / Non-Goals

**Goals:**

- 在 Windows 上讓 `byok update` 能成功替換執行中的 `byok.exe`，不再出現 "Access is denied" 錯誤。
- 替換後新版本可正常執行。
- 清理舊執行檔備份，避免殘留廢棄檔案。

**Non-Goals:**

- 不引入第三方 Go 相依（如 `golang.org/x/sys`），繼續使用 `syscall`。
- 不改變 CLI 介面、旗標或輸出格式。
- 不處理跨磁碟區移動（暫存檔與目標已在同目錄）。
- 不修改 Unix 平台的替換邏輯。

## Decisions

### Decision: Rename-then-move 模式取代直接覆寫

在 Windows 上，先將執行中的執行檔重新命名為備份檔（`byok.exe` to `byok.exe.old`），再將暫存檔移至原始路徑。Windows 允許重新命名執行中的 `.exe`（但不允許覆寫或刪除），因此此兩步操作可成功完成。

替代方案考量：
- `MOVEFILE_DELAY_UNTIL_REBOOT` 直接排程替換：使用者需重開機才能完成更新，體驗差，不採用。
- 產生 helper 進程在主進程結束後替換：實作複雜、跨進程協調易出錯，不採用。

### Decision: 備份檔以 MOVEFILE_DELAY_UNTIL_REBOOT 排程刪除

重新命名後的 `byok.exe.old` 仍被執行中進程鎖定，無法立即刪除。嘗試刪除失敗後，以 `MoveFileExW` + `MOVEFILE_DELAY_UNTIL_REBOOT` 旗標排程於下次開機刪除。若刪除成功（進程已釋放鎖定的罕見情況），則不需排程。

### Decision: renameReplace 函式簽名不變

`renameReplace(src, dst string) error` 介面維持不變。備份檔名稱由 `dst` 衍生（加上 `.old` 副檔名），不需額外參數。此保持 `replaceAt` 呼叫端不需修改。

## Implementation Contract

**Behavior:** 使用者在 Windows 執行 `byok update` 時，更新流程將下載的新二進位寫入暫存檔後，先將目標執行檔重新命名為 `<target>.old`，再將暫存檔重新命名為目標路徑。若 `byok.exe.old` 無法立即刪除，則排程於下次開機刪除。使用者觀察到的行為與更新成功訊息不變。

**Interface:** `renameReplace(src, dst string) error` 函式簽名不變。內部行為變更為：
1. 嘗試 `os.Rename(dst, dst+".old")`；若失敗且非「檔案不存在」錯誤，回傳錯誤。
2. 嘗試 `os.Rename(src, dst)`；若失敗，嘗試將備份還原（`os.Rename(dst+".old", dst)`）並回傳錯誤。
3. 嘗試 `os.Remove(dst + ".old")`；若失敗（檔案被鎖定），呼叫 `MoveFileExW(dst+".old", nil, MOVEFILE_DELAY_UNTIL_REBOOT)` 排程開機刪除。排程失敗為非致命錯誤，僅記錄不中斷流程。

**Failure modes:**
- 重新命名目標為備份失敗（如權限不足）：回傳錯誤，暫存檔由 `replaceAt` 的 defer 清理。
- 暫存檔移至目標路徑失敗：嘗試還原備份，回傳錯誤。
- 備份檔刪除或排程刪除失敗：非致命，更新仍視為成功（新執行檔已就位）。

**Acceptance criteria:**
- `go test ./internal/updater/ -run TestRenameReplaceWindows -race` 在 Windows 上通過：測試模擬目標檔案被鎖定（以獨占開啟），驗證 rename-then-move 成功且目標內容為新二進位。
- `go test ./internal/updater/ -run TestDownloadAndReplace_WindowsRunningExe -race` 在 Windows 上通過：整合測試模擬完整 `DownloadAndReplace` 流程搭配執行中執行檔，驗證端到端行為。
- `go test ./internal/updater/ -race` 在 Windows 上全部通過（含既有測試）。

**Scope boundaries:**
- In scope: `internal/updater/replace_windows.go` 的 `renameReplace` 改寫、對應測試。
- Out of scope: Unix 平台 `replace_unix.go`、`replaceAt` 流程、cmd 層、CLI 介面。

## Risks / Trade-offs

- [備份檔殘留] 若 `MOVEFILE_DELAY_UNTIL_REBOOT` 排程也失敗（極罕見），`byok.exe.old` 會殘留於安裝目錄。此為非致命問題，不影響新版本運作，下次更新時會再次嘗試。可在文件中註明使用者可手動刪除。
- [備份檔還原失敗] 暫存檔移至目標失敗後嘗試還原備份，若還原也失敗則目標路徑可能空無一物。此情境極罕見（兩次 Rename 皆失敗），錯誤訊息會提示使用者重新下載安裝。
- [既存 `.old` 檔案衝突] 若安裝目錄已存在上一輪更新殘留的 `byok.exe.old`，第一次 `os.Rename(dst, dst+".old")` 會嘗試覆寫。`os.Rename` 在 Windows 上無法覆寫已存在檔案，需先刪除既存 `.old` 或使用 `MoveFileExW` + `MOVEFILE_REPLACE_EXISTING`（`.old` 非執行中檔案，可覆寫）。
