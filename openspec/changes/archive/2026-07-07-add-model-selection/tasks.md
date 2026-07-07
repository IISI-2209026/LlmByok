## 1. Profile 資料模型與設定檔遷移

- [x] 1.1 將 `internal/config/config.go` 的 `Profile` 結構 `DefaultModel string` 欄位替換為 `Models []string`（YAML 鍵 `models`）— 對應 design 決策「profile 資料模型改為候選模型清單」，並滿足 spec 「Config file location」對 `models` 欄位與舊 `default_model` 遷移的要求：讓 `Config.Load` 在載入含舊 `default_model` 且 `models` 為空的 profile 時自動遷移為單元素 `models` 清單，`Config.Save` 不再寫出 `default_model` 欄位。驗證：`internal/config/config_test.go` 新增「舊檔載入後 `models` 為 `["gpt-4o"]`」與「儲存後檔案不含 `default_model`」兩情境通過。
- [x] 1.2 [P] 新增 `internal/config/models.go`，提供候選模型互動選單函式（接收 `models []string`、`io.Reader`、`io.Writer`，回傳選取模型字串與錯誤），實作上下鍵選擇並以 `golang.org/x/term` 偵測終端機；stdin 非終端機時回傳錯誤。驗證：`internal/config/models_test.go` 注入 mock reader/writer 測試選單回傳正確模型與非終端機錯誤情境通過。

## 2. Config 子指令位置參數化

- [x] 2.1 [P] 修改 `cmd/config.go` 的 `newConfigAddCmd` — 對應 design 決策「config 子指令改以位置參數接收 profile 名稱」並滿足 spec 「Add a profile」：`Use` 改為 `add <profile name>`，profile 名稱取自第一位置參數，移除 `--name` 旗標，移除 `--default-model` 旗標，新 profile 建立時 `models` 為空清單，互動模式不再提示模型。驗證：`cmd/config_test.go` 的「Add new profile to empty config via parameters」「Missing positional profile name rejected」「Interactive mode prompts for all fields」情境通過。
- [x] 2.2 [P] 修改 `cmd/config.go` 的 `newConfigDeleteCmd` — 對應 design 決策「config 子指令改以位置參數接收 profile 名稱」並滿足 spec 「Remove a profile」：`Use` 改為 `delete <profile name>`，profile 名稱取自位置參數，移除 `--name` 旗標。驗證：`cmd/config_test.go` 的 delete 各情境（含非存在 profile 退出 1）通過。
- [x] 2.3 [P] 修改 `cmd/config.go` 的 `newConfigSetDefaultCmd` — 對應 design 決策「config 子指令改以位置參數接收 profile 名稱」並滿足 spec 「Set default profile」：`Use` 改為 `set-default <profile name>`，profile 名稱取自位置參數，移除 `--name` 旗標。驗證：`cmd/config_test.go` 的「Change default via set-default」情境通過。
- [x] 2.4 [P] 修改 `cmd/config.go` 的 `newConfigUpdateCmd` — 對應 design 決策「config 子指令改以位置參數接收 profile 名稱」並滿足 spec 「Update an existing profile」：`Use` 改為 `update <profile name>`，profile 名稱取自位置參數，移除 `--name` 旗標與 `--default-model` 旗標，未提供欄位旗標時進入互動模式（位置參數仍必填），`models` 不被 update 修改。驗證：`cmd/config_test.go` 的「Update api base and keep key」「Non-existent profile rejected」情境通過。

## 3. 新增 config set-models 子指令

- [x] 3.1 新增 `cmd/set_models.go` 實作 `byok config set-models <profile name>`（掛於 `config` 指令下） — 對應 design 決策「新增 `byok config set-models <profile name>` 子指令」並滿足 spec 「Set candidate models for a profile」：位置參數為 profile 名稱，可重複 `--model` 旗標（`StringArrayVar`）整批覆寫 `models`，無 `--model` 且終端機時逐行互動輸入至空行，結果為空時退出 1，profile 不存在時列出可用 profile 並退出 1，非終端機 stdin 觸發互動時退出 1。驗證：新增 `cmd/set_models_test.go` 涵蓋「Set multiple models via flags」「Replace existing model list」「Empty model list rejected」「Non-existent profile rejected」「Interactive mode collects models until empty line」「Interactive mode rejected on non-tty stdin」情境全數通過。
- [x] 3.2 [P] 在 `cmd/config.go` 將 `set-models` 註冊為 `config` 的子指令（與 `add`/`delete`/`set-default`/`update`/`list` 同層），不再於 `cmd/root.go` 註冊為頂層指令。驗證：執行 `byok config set-models --help` 顯示用法且 `byok config --help` 列出 `set-models`，`byok --help` 不再列出頂層 `set-models`。

## 4. config list 顯示模型清單

- [x] 4.1 [P] 修改 `cmd/config.go` 的 `newConfigListCmd` 輸出 — 對應 design 決策「`byok config list` 顯示模型清單」並滿足 spec 「List profiles with masked API key」對 `models` 欄位的顯示要求：將 `default_model` 欄位改為 `models` 清單，多個模型以逗號分隔（如 `gpt-4o, gpt-4o-mini`），空清單顯示為空字串，其餘欄位不變。驗證：`cmd/config_test.go` 的「List with masked keys and models」「List profile with empty models」情境通過。

## 5. Launch 模型解析

- [x] 5.1 修改 `cmd/launch.go` 在讀取 profile 後、呼叫各 target runner 前插入模型解析 — 對應 design 決策「launch 模型解析為候選清單邏輯」並滿足 spec 「Launch Copilot with BYOK profile」的五條模型解析規則：帶 `--model` 直接用該值；未帶且 `models` 恰一個用該模型；未帶且多個且終端機呼叫 `internal/config` 選單函式；未帶且多個且非終端機印錯誤退出 1；`models` 為空印錯誤提示 `byok config set-models` 退出 1。解析後將單一模型字串傳入各 runner，runner 環境變數注入邏輯不變。驗證：`cmd/launch_test.go` 新增「single candidate model used」「--model overrides candidates」「interactive selection」「multiple candidates rejected on non-tty stdin」「empty candidate models rejected」情境通過。
- [x] 5.2 [P] 將模型解析邏輯套用至所有 target：`cmd/launch_codex.go`、`cmd/launch_codex_app.go`、`cmd/launch_claude.go`、`cmd/launch_pi.go` 皆共用同一解析函式並注入各 target 的模型環境變數。驗證：`cmd/launch_codex_test.go`、`cmd/launch_codex_app_test.go`、`cmd/launch_claude_test.go`、`cmd/launch_pi_test.go` 各新增「single candidate model」「--model override」情境通過，對應 spec 「Model resolution shared across launch targets」。

## 6. 文件與範例更新

- [x] 6.1 [P] 更新 `README.md` 與 `AGENTS.md`：將 `byok config add/delete/set-default/update --name` 範例改為位置參數形式，移除 `--default-model` 說明，新增 `byok config set-models <profile name> --model ...` 用法與 `byok launch <target>` 多模型互動選擇說明，更新 `byok config list` 輸出範例顯示 `models` 欄位。驗證：文件內容審查，所有 CLI 範例可對應至新指令形式且不再出現 `--name`/`--default-model`。

## 7. 互動式模型選單的終端機處理與取消（實作精煉）

- [x] 7.1 [P] 修改 `internal/config/models.go` 的 `SelectModel`：於 `in` 為 `*os.File` 時以 `term.MakeRaw` 切換 stdin 為 raw mode（即時讀鍵、關閉本地回顯與行緩衝，使方向鍵以 ANSI 序列送達），`defer` 以 `term.Restore` 還原；raw 失敗時最佳努力以 cooked 模式繼續。行為：按向下鍵再 Enter 注入第二個候選（而非永遠第一個），向上鍵自第一個繞回最後一個。驗證：`internal/config/models_test.go` 的 `TestSelectModel_DownThenEnterSelectsSecond`、`TestSelectModel_UpWrapsToLast` 通過。
- [x] 7.2 [P] 新增 `internal/config/models_windows.go` 與 `internal/config/models_unix.go`：`platformEnableVT` 於 Windows 以 `SetConsoleMode` 啟用 stdout 的 `ENABLE_VIRTUAL_TERMINAL_PROCESSING`（回傳 restore 還原原 mode），非 Windows 為 no-op；`SelectModel` 於 `out` 為 `*os.File` 時呼叫之。行為：ANSI 反白（`\x1b[7m`/`\x1b[0m`）與游標控制序列在 Windows console 正確渲染。驗證：`go build ./...` 於 windows/linux/darwin 三平台皆通過；`go test ./internal/config/... -race` 通過。
- [x] 7.3 [P] 修改 `internal/config/models.go` 的 `renderMenu`：以「❯ 游標 + 反白」標記選取列、未選取列以兩空白對齊，原地重繪（`clearMenu` 游標上移 + 清除）；`SelectModel` 新增 Ctrl-C（`\x03`）與單獨 Esc（ESC 後非 `[`）取消路徑，回傳新增的 `ErrSelectionCancelled`。行為：取消時清除選單並回傳取消錯誤。驗證：`TestSelectModel_CtrlCancelsSelection`、`TestSelectModel_EscapeCancelsSelection` 通過。
- [x] 7.4 修改 `cmd/launch.go` 的 `resolveModelForLaunch`：偵測 `config.ErrSelectionCancelled` 並印「已取消模型選擇。」後回傳 `errExit`（非零結束碼），其他 `SelectModel` 錯誤沿用「模型選擇失敗」訊息；加 `errors` import。行為：Ctrl-C/Esc 取消時不啟動子程序並退出 1。驗證：`go build ./cmd/...` 通過；`go test ./cmd/ -race` 通過。

## 8. `byok config list` 欄位對齊（實作精煉）

- [x] 8.1 [P] 修改 `cmd/config.go` 的 `runConfigList`：改以顯示寬度感知動態計算各欄寬（每欄寬 = 標題與各列該欄顯示寬度最大值），欄間固定 2 空白，最末欄不加尾隨空白；新增 `displayWidth`/`runeWidth`（CJK 與全形算 2 欄，其餘 1 欄）、`padWidth`（以顯示寬度補空白）、`profileRow`/`pickCol` 輔助。行為：CJK 標題（名稱/模型/來源）與超長模型值（如 `glm-5.2, kimi-k2.7-code`）不造成後續欄位錯位，標題行與資料行各欄首顯示寬度位置一致。驗證：`cmd/config_test.go` 新增 `TestConfigList_ColumnsAligned` 通過，且 `TestConfigList_Output`、`TestConfigList_EmptyModelsShownEmpty` 仍通過。