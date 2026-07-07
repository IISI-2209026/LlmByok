## Context

`byok` 現行設定檔中每個 profile 只攜帶單一 `default_model` 欄位，`byok launch <target>` 在未帶 `--model` 時直接套用該值。使用者若想切換模型，必須記住模型名稱並每次以 `--model` 指定，體驗不便。

相關既有規格：
- `openspec/specs/byok-config/spec.md`：profile 結構含 `default_model`；`add/delete/set-default/update` 透過 `--name` 旗標指定 profile；`list` 顯示 `default_model` 欄位。
- `openspec/specs/byok-launch/spec.md`：`launch <target>` 以 `--model` 覆寫 `default_model`，未帶時使用 profile 的 `default_model`。

本變更為跨 `internal/config`、`cmd`（config 與各 launch 指令）、`internal/runner` 的資料模型與 CLI 介面調整，且涉及舊設定檔遷移，故建立設計文件。

## Goals / Non-Goals

**Goals:**

- profile 改攜帶候選模型清單 `models`，支援單一 profile 對應多個可用模型。
- `byok launch <target>` 在未帶 `--model` 時，依候選數量決定模型：單一直接使用、多個於終端機互動選擇、為空則錯誤。
- config CLI 改以位置參數接收 profile 名稱，並新增 `config set-models` 子指令維護候選清單。
- `byok config list` 顯示每個 profile 的模型清單。
- 提供舊設定檔（含 `default_model`）的自動遷移路徑。

**Non-Goals:**

- 不改動 keychain / `--key-storage` 機制。
- 不改動 `--profile`、`--config`、`--yolo`、`--` 透傳語意。
- 不新增模型排序、分群、favorite 標記等後續擴充。
- 不改動各 target 的環境變數注入邏輯，僅調整「模型如何決定」。

## Decisions

### Profile 資料模型改為候選模型清單

`Profile` 結構將 `DefaultModel string` 欄位替換為 `Models []string`（YAML 鍵 `models`）。載入時若發現舊欄位 `default_model` 且 `models` 為空，則將該值遷移為單元素 `models` 清單並視為已遷移；儲存時一律只寫 `models`，不再寫 `default_model`。此舉讓單一 profile 可承載多個候選模型，同時保持對既有設定檔的向後相容。

### Config 子指令改以位置參數接收 profile 名稱

`byok config add/delete/set-default/update` 改為 `byok config add <profile name>` 形式，profile 名稱作為第一位置參數；`--name` 旗標移除。這是 BREAKING 變更，但能讓指令更簡潔並與新增的 `byok config set-models <profile name>` 形式一致。`update` 的互動模式觸發條件改為「未提供任何欄位旗標且位置參數已給定 profile 名稱」時進入互動（位置參數為必填）。

### 新增 `byok config set-models <profile name>` 子指令

`byok config set-models <profile name> --model <m1> --model <m2> ...` 以可重複的 `--model` 旗標設定候選模型清單，整批覆寫該 profile 的 `models`。未提供任何 `--model` 時進入終端機互動模式，逐行提示輸入模型（空行結束）。此子指令取代過去以 `--default-model` 設定單一模型的用途。`set-models` 註冊為 `config` 的子指令（與 `add`/`delete`/`set-default`/`update`/`list` 同層），不再作為頂層指令，讓所有 profile 管理操作收攏在 `byok config` 之下。profile 不存在時印出錯誤並退出代碼 1。

### Launch 模型解析為候選清單邏輯

`byok launch <target>` 模型決定流程如下，在讀取 profile 後、注入環境變數前執行：
1. 帶 `--model` → 一律使用 `--model` 值，不顯示互動選單。
2. 未帶 `--model` 且 `models` 恰一個 → 直接使用該模型。
3. 未帶 `--model` 且 `models` 多個且 stdin 為終端機 → 顯示上下鍵互動選單，回傳使用者選取的模型。
4. 未帶 `--model` 且 `models` 多個但 stdin 非終端機 → 印出錯誤（提示於非互動環境需指定 `--model`）並退出代碼 1。
5. `models` 為空 → 印出錯誤並提示執行 `byok config set-models <profile name>`，退出代碼 1。

互動選單實作集中於 `internal/config` 共用模組（如 `internal/config/models.go` 的選單函式），各 launch 指令與 runner 之間只傳遞「解析後的單一模型字串」，runner 注入環境變數的邏輯維持不變。

### `byok config list` 顯示模型清單

`list` 將原本的 `default_model` 欄位改為顯示 `models` 清單；多個模型以逗號分隔呈現於同一欄（如 `gpt-4o, gpt-4o-mini`），空清單顯示為空字串。其餘欄位（name、provider、api_base、masked api_key、key source）維持不變。

### 互動式模型選單的終端機處理

第一版互動選單以 cooked（行緩衝）模式讀取 stdin，方向鍵被 Windows console 的行編輯攔截而未送達程式，導致送出後仍使用第一個模型；且重繪用的 ANSI 序列在未啟用 VT 處理的 Windows console 上被當垃圾字元印出，視覺回饋崩壞。修正方向：

- **stdin 切 raw mode**：以 `term.MakeRaw` 在選單期間關閉本地回顯與行緩衝，使方向鍵以 ANSI 序列即時送達、Enter 不被行緩衝；離開時以 `term.Restore` 還原。raw mode 失敗時最佳努力地以 cooked 模式繼續。
- **stdout 啟用 VT 處理**：Windows 透過 `SetConsoleMode` 加 `ENABLE_VIRTUAL_TERMINAL_PROCESSING`，Unix 終端機原生支援，集中於平台特定檔（`models_windows.go` / `models_unix.go`）。
- **游標 + 反白渲染**：以「❯ 游標 + 反白（`\x1b[7m`/`\x1b[0m`）」標記選取列，原地重繪（游標上移 + 清除）。
- **取消**：Ctrl-C（`\x03`）或單獨 Esc（ESC 後非 `[`）清除選單並回傳 `config.ErrSelectionCancelled`，呼叫端印「已取消模型選擇。」並以非零結束碼退出。
- **可測試性**：輸入非 `*os.File`（測試 buffer）時跳過 raw/VT 切換，仍以注入按鍵序列驅動，使單元測試無需真實 TTY。

### `byok config list` 欄位對齊

原表格以 Go 格式動詞 `%-20s` 等固定欄寬渲染，有兩類錯位：（1）中文標題（名稱/模型/來源）以 rune 數填充但顯示寬度為 2，標題行多填空白而與 ASCII 資料行錯位；（2）模型欄內容超過固定欄寬（如 `glm-5.2, kimi-k2.7-code`）把後續欄位往右擠。修正方向：改以「顯示寬度感知」動態計算各欄寬（每欄寬 = 標題與各列該欄顯示寬度的最大值），欄間固定 2 空白，最末欄不加尾隨空白；以 `displayWidth`/`runeWidth`（CJK 與全形算 2 欄）與 `padWidth`（以顯示寬度補空白）輔助。

## Implementation Contract

**行為（end user 可觀察）**

- `byok config add <name> [--provider ... --api-base ... --api-key ... --key-storage ...]`：以位置參數指定 profile 名稱建立 profile；新 profile 不再要求 `--default-model`，模型改由 `config set-models` 設定。未提供任何欄位旗標時進入終端機互動模式（提示 name、provider、api_base、api_key、key storage；不再提示 default_model）。
- `byok config delete <name>`、`byok config set-default <name>`、`byok config update <name> [...]`：profile 名稱為位置參數。
- `byok config set-models <name> --model a --model b`：覆寫該 profile 候選模型為 `[a, b]`；無 `--model` 且為終端機時互動輸入；profile 不存在 → 錯誤退出 1。
- `byok config list`：輸出含 `models` 欄位列出候選模型（逗號分隔）；各欄以顯示寬度感知的動態欄寬對齊，CJK 標題與超長模型值不造成錯位。
- `byok launch <target>`：模型決定依上述 5 條規則；`--model` 仍為明確覆寫。多候選互動選單以 raw mode + VT 處理渲染「❯ 游標 + 反白」，方向鍵驅動選取（非回退到第一個），Ctrl-C/Esc 取消並退出 1。

**介面 / 資料形狀**

- `config.Profile` 結構：`Models []string`（YAML `models`），移除 `DefaultModel`。
- 新子指令 `config set-models`：位置參數 `<profile name>` + 可重複 `--model` 旗標（`StringArrayVar`/`StringSliceVar`）；註冊於 `config` 指令下，非頂層指令。
- `add`/`delete`/`set-default`/`update`：`Use` 含 `<profile name>` 位置參數，移除 `--name` 旗標定義。
- 互動選單函式 `config.SelectModel(models, in, out, isTerminal)`：接收候選模型切片與 `io.Reader`/`io.Writer`，回傳選取模型字串與錯誤；非終端機時回傳 `ErrNotInteractive`，空候選回傳 `ErrNoCandidates`，Ctrl-C/Esc 取消回傳 `ErrSelectionCancelled`。當 `in` 為 `*os.File` 時切換 raw mode（`term.MakeRaw`/`term.Restore`），當 `out` 為 `*os.File` 時啟用 VT 處理；非 `*os.File`（測試）跳過切換。
- 平台特定 VT 處理：`models_windows.go` 以 `SetConsoleMode` 啟用 `ENABLE_VIRTUAL_TERMINAL_PROCESSING`；`models_unix.go` 為 no-op。
- `byok config list` 表格輔助：`displayWidth`/`runeWidth`（CJK 與全形算 2 欄）、`padWidth`（以顯示寬度補空白）、`profileRow`/`pickCol` 動態欄寬計算。
- runner 介面不變：仍接收單一模型字串注入環境變數。

**失敗模式**

- profile 不存在（config set-models / update / delete / set-default）→ 印出錯誤訊息列出可用 profile，退出代碼 1，不修改檔案。
- 候選模型為空且 launch 未帶 `--model` → 錯誤並提示 `byok config set-models`，退出代碼 1。
- 多候選模型且 stdin 非終端機且未帶 `--model` → 錯誤並提示指定 `--model`，退出代碼 1。
- 互動模式於非終端機 stdin 觸發 → 錯誤並提示改用參數模式，退出代碼 1（沿用既有 add 行為）。
- 互動選單以 Ctrl-C/Esc 取消 → 印「已取消模型選擇。」並退出代碼 1，不啟動子程序。
- 設定檔解析錯誤 → 沿用既有錯誤訊息與退出代碼 1。

**驗收條件**

- `byok launch <target>` 於單一候選模型 profile 且未帶 `--model` 時，注入該模型（測試 `cmd/launch_test.go` 對應情境）。
- `byok launch <target>` 於多候選模型 profile 且未帶 `--model` 且 stdin 為終端機時，呼叫選單函式並以其回傳值注入（注入 mock 選單測試）。
- `byok launch <target>` 於多候選模型且 stdin 非終端機且未帶 `--model` 時，退出代碼 1 並印出指定 `--model` 的錯誤。
- `byok launch <target> --model x` 一律注入 `x`，不論候選數量。
- `byok launch <target>` 多候選互動選單：按向下鍵再 Enter 注入第二個候選（`TestSelectModel_DownThenEnterSelectsSecond`、`cmd/launch_model_test.go` 對應情境），向上鍵自第一個繞回最後一個（`TestSelectModel_UpWrapsToLast`），Ctrl-C 與單獨 Esc 回傳 `ErrSelectionCancelled` 並退出 1（`TestSelectModel_CtrlCancelsSelection`、`TestSelectModel_EscapeCancelsSelection`）。
- `byok config set-models <name> --model a --model b` 後 `byok config list` 顯示 `a, b`。
- `byok config list` 表格各欄在顯示寬度上對齊：CJK 標題與超長模型值（如 `glm-5.2, kimi-k2.7-code`）不造成後續欄位錯位（`TestConfigList_ColumnsAligned`）。
- `byok config add <name>` 以位置參數建立 profile；`byok config update <name> --api-base ...` 僅更新該欄位。
- 含舊 `default_model` 的設定檔載入後 `models` 為單元素清單，`launch` 可正常使用該模型。
- `cmd/config_test.go`、`cmd/launch_test.go`、`internal/config/config_test.go`、`internal/config/models_test.go` 更新涵蓋上述情境。

**範圍邊界**

- in scope：`internal/config` 結構與遷移、`cmd/config.go` 子指令位置參數化與 `list` 顯示寬度對齊、新 `cmd/set_models.go`、`cmd/launch*.go` 模型解析、`internal/config/models.go` 選單（raw mode、VT 處理、游標反白、取消）、`internal/config/models_windows.go`/`models_unix.go` 平台 VT、`README.md`/`AGENTS.md` 用法更新、相關測試更新。
- out of scope：keychain 機制、`--key-storage`、各 target 環境變數注入邏輯、`--yolo`/`--` 透傳、模型排序/分群/favorite、完整 East Asian Width 實作（`displayWidth` 為涵蓋本表格的近似）。

## Risks / Trade-offs

- **BREAKING CLI 格式**：`--name` 移除、profile 名稱改位置參數，既有腳本與文件需更新 → 於 README/AGENTS 更新範例，並在設計中明確標示 BREAKING；不提供舊 `--name` 回退以避免雙介面維護負擔。
- **設定檔遷移**：舊檔含 `default_model`，新程式碼不寫回該欄 → 載入時自動遷移為 `models` 單元素清單；首次儲存後舊欄位自然消失，無需一次性遷移腳本。
- **非終端機環境卡住風險**：多候選模型在 CI/piped stdin 下無法互動 → 明確偵測非終端機並錯誤退出，要求 `--model`。
- **互動選單終端機相容性**：raw mode 與 VT 處理跨平台差異 → raw 以 `golang.org/x/term` 抽象、VT 以平台特定檔封裝；raw 失敗最佳努力繼續；非 `*os.File` 輸入跳過切換維持可測試性。
- **`displayWidth` 為近似**：非完整 East Asian Width 實作，僅涵蓋本表格會出現的 CJK 標題與全形符號 → 足以讓 `byok config list` 對齊；若未來欄位內容擴及罕用組合字元可再升級為完整寬度表。
- **互動選單跨平台**：上下鍵選擇需處理 Windows/Unix 終端機 → 選用既有終端機讀取抽象（`golang.org/x/term` 已為依賴），實作集中於共用模組並以可注入 reader/writer 便於測試。
