## 1. 改寫 Windows renameReplace 為 rename-then-move 模式

- [x] 1.1 改寫 `internal/updater/replace_windows.go` 的 `renameReplace` 函式為 rename-then-move 三步驟：(1) 先處理既存 `.old` 檔案（以 `MoveFileExW` + `MOVEFILE_REPLACE_EXISTING` 覆寫或 `os.Remove`），(2) `os.Rename(dst, dst+".old")` 將執行中執行檔重新命名為備份，(3) `os.Rename(src, dst)` 將新二進位移至原始路徑。若 step 2 失敗回傳錯誤；若 step 3 失敗嘗試還原備份 `os.Rename(dst+".old", dst)` 後回傳錯誤。涵蓋 "Decision: Rename-then-move 模式取代直接覆寫"、"Decision: renameReplace 函式簽名不變"。行為：Windows 上 `renameReplace` 不再直接覆寫執行中執行檔，改以重新命名方式完成替換。驗證：`go build ./internal/updater/` 成功編譯。
- [x] 1.2 在 `internal/updater/replace_windows.go` 新增 `movefileDelayUntilReboot = 0x00000004` 常數，並在 `renameReplace` 最後一步嘗試 `os.Remove(dst+".old")` 刪除備份檔；若刪除失敗（檔案被執行中進程鎖定），呼叫 `MoveFileExW(dst+".old", nil, movefileDelayUntilReboot)` 排程於下次開機刪除。排程失敗為非致命錯誤，不中斷更新流程。涵蓋 "Decision: 備份檔以 MOVEFILE_DELAY_UNTIL_REBOOT 排程刪除"。行為：備份檔被排程開機刪除或立即刪除成功，不殘留廢棄檔案。驗證：`go build ./internal/updater/` 成功編譯。

## 2. 新增 Windows 平台測試

- [x] 2.1 在 `internal/updater/updater_test.go` 新增 `TestRenameReplaceWindowsRunningExe` 測試（`//go:build windows`）：編譯一個子進程 .exe（複製測試 binary 並以 `--sleep` 參數啟動），對該執行中 .exe 執行 `renameReplace`，驗證目標路徑內容為新二進位且備份檔 `.old` 存在或已排程刪除。驗證 `byok update` self-update command 需求的 "Windows self-update replaces running executable via rename-then-move" 情境。驗證：`go test ./internal/updater/ -run TestRenameReplaceWindowsRunningExe -race` 在 Windows 通過。
- [x] 2.2 在 `internal/updater/updater_test.go` 新增 `TestRenameReplaceWindowsExistingOld` 測試（`//go:build windows`）：在目標目錄預先放置一個 `.old` 檔案，模擬上一輪更新殘留，執行 `renameReplace` 後驗證更新正常完成且舊 `.old` 被覆寫。驗證 `byok update` self-update command 需求的 "Existing backup file from prior update is overwritten" 情境。驗證：`go test ./internal/updater/ -run TestRenameReplaceWindowsExistingOld -race` 在 Windows 通過。
- [x] 2.3 在 `internal/updater/updater_test.go` 新增 `TestRenameReplaceWindowsFailureRestore` 測試（`//go:build windows`）：模擬 step 3（`os.Rename(src, dst)`）失敗場景（如 src 不存在），驗證 `renameReplace` 嘗試還原備份並回傳錯誤。驗證 `byok update` self-update command 需求的 "Windows rename-then-move failure restores backup" 情境。驗證：`go test ./internal/updater/ -run TestRenameReplaceWindowsFailureRestore -race` 在 Windows 通過。

## 3. 驗證

- [x] 3.1 執行 `go test ./internal/updater/ -race` 確認所有既有測試與新增測試在 Windows 上全部通過，無 race 偵測問題。驗證：完整測試套件通過且 exit code 0。
- [x] 3.2 在 `internal/updater/replace_windows_test.go` 新增 `TestDownloadAndReplace_WindowsRunningExe` 整合測試（`//go:build windows`）：編譯 helper .exe 並啟動（模擬執行中 `byok.exe`），以 `withExecutableFn` 覆寫目標路徑，建立 stub HTTP server 回傳含新二進位的 zip 封存，組 `Release` 物件呼叫 `DownloadAndReplace`（完整生產程式碼路徑：`SelectAsset` → `downloadAndExtract` → 暫存檔 → `renameReplace`），驗證無 "Access is denied" 錯誤且目標路徑內容為新二進位。涵蓋 `byok update` self-update command 需求的完整端到端行為（自動化整合測試取代手動驗證）。驗證：`go test ./internal/updater/ -run TestDownloadAndReplace_WindowsRunningExe -race` 在 Windows 通過。
