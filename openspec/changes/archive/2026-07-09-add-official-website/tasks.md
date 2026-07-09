<!--
Each task description MUST state:
- the behavior or contract being delivered (what is observably true when the
  task is complete), and
- the verification target that proves completion (test, CLI invocation,
  analyzer check, manual assertion, or content review).

File paths are supporting context for locating the work, never the task
itself. "Edit file X" is not a valid task — it is missing both behavior and
verification.
-->

## 1. 建立靜態資源結構

- [x] 1.1 建立 `public/index.html` 基礎結構。採用 Vanilla HTML/CSS/JS 實作，包含必要的 meta tags，並連結 CSS 與 JS 檔案。驗證：瀏覽器載入 `index.html` 能看到空白但帶有正確標題的網頁，且 Console 無載入錯誤。

## 2. 樣式與視覺設計

- [x] [P] 2.1 實作 Tech Dark Theme and Glassmorphism 與 Tech Dark 風格與玻璃擬物化設計。於 `public/style.css` 定義 CSS 變數、深色背景、玻璃擬物化卡片樣式與基礎排版。驗證：在 `index.html` 插入測試元素，瀏覽器呈現具備深色主題、模糊背景與正確字體的視覺效果。
- [x] [P] 2.2 實作 Before and After Comparison 視覺佈局。於 `public/style.css` 建立對比版面結構。驗證：在 HTML 放入假內容，畫面可清楚呈現「手動設定」與「一行指令」的版面。

## 3. 網頁內容與互動實作

- [x] 3.1 實作 Terminal Typing Animation。於 `public/script.js` 撰寫打字效果邏輯，並在 Hero 區塊的特定 DOM 元素中模擬輸入。驗證：重新整理網頁時，該區塊會以動畫形式顯示「byok launch copilot」。
- [x] 3.2 實作 Copy to Clipboard Functionality 與 提供「複製到剪貼簿」功能。於 `public/script.js` 撰寫呼叫 `navigator.clipboard.writeText` 的邏輯，支援複製成功視覺回饋與優雅降級。驗證：點擊複製按鈕，剪貼簿內會有正確文字，且按鈕短暫顯示 "Copied!"。

## 4. 內容組裝與最終測試

- [x] 4.1 將真實的說明文字、流程圖、卡片與下載指令整合至 `public/index.html` 中，套用所有樣式與 JS。驗證：手動操作整頁滾動、打字動畫、Before/After 區塊與所有複製按鈕，所有功能運作正常且視覺符合 Tech Dark 規範。

## 5. Claude Context Window 支援

- [x] 5.1 實作 Launch Claude with BYOK profile 中的 Claude Context Window 後綴注入。於 `internal/runner/claude.go` 的模型設定邏輯中，將傳入的模型字串強制附加 `[1m]` 之後再設定給 `ANTHROPIC_MODEL` 環境變數。驗證：執行 `go test ./internal/runner/...` 確保測試通過，且手動測試 `byok launch claude` 時觀察送出的環境變數確實包含 `[1m]`。

## 6. 導覽列與 Quick Start 區塊

- [x] 6.1 實作 Navigation Bar 導覽列。於 `public/index.html` 頁面頂部加入 sticky 導覽列，含「特色、如何運作、核心功能、上手、安裝」快速跳轉連結。驗證：點擊導覽列連結可平滑捲動至對應區塊。
- [x] 6.2 實作 Quick Start Section 三分鐘上手區塊。於 `public/index.html` Features 與 Install 之間新增 Quick Start 區塊，Step 1 顯示 `byok config add my-profile` 指令，Step 2 以五張卡片展示各工具啟動指令（Copilot CLI、Codex CLI、Codex App、Claude、Pi）。驗證：Quick Start 區塊的兩步驟教學清晰可見，五張卡片各附啟動指令與複製按鈕。

## 7. 官方工具 Icon 與 Install 區塊重組

- [x] 7.1 新增 Tool Icons（官方 SVG 工具圖示）。於 `public/icons/` 目錄新增 copilot.svg（GitHub Copilot）、openai.svg（OpenAI）、anthropic.svg（Anthropic）、pi.svg（Pi 自製 π 圖示），並在 Quick Start 卡片中以 `<img>` 取代 Emoji。驗證：五張工具卡片正確顯示官方 SVG 圖示。
- [x] 7.2 重組 Install Section with OS Tabs 安裝區塊 OS Tab 切換。將 Install 區塊移至頁面最後，加入「無須依賴」徽章，以 OS tab（Linux / macOS / Windows）切換顯示對應平台安裝指令。驗證：點擊各 OS tab 可正確切換顯示對應安裝指令，隱藏其他平台指令。

## 8. 版面與複製按鈕 UI 調整

- [x] 8.1 拉寬版面容器。將 `.container` max-width 從 960px 調整為 1280px，內部區塊（quickstart-step、os-tabs、hero-content 等）從 720px 調整為 920px。驗證：頁面整體視覺更寬敞，命令列不再因容器過窄而截斷。
- [x] 8.2 Single-Line Command Display（命令列單行顯示且無捲軸）。於 `public/style.css` 將 `.code-block` 設為 `overflow: hidden`、`.code-block code` 設為 `white-space: nowrap` + `text-overflow: ellipsis`，確保不出現水平捲軸。驗證：所有命令列單行顯示，無水平捲軸出現。
- [x] 8.3 複製按鈕改用 SVG Icon。將所有 9 個複製按鈕的文字「Copy」替換為剪貼簿 SVG 圖示，複製成功時切換為勾選圖示，2 秒後恢復原圖示。更新 `public/script.js` 中的 `showCopied` 與 `fallbackCopy` 以操作 SVG 而非文字。驗證：所有複製按鈕顯示剪貼簿圖示，點擊後短暫顯示勾選圖示。
- [x] 8.4 How It Works Flow Diagram 流程方框等寬。於 `public/style.css` 將 `.flow-step` 加上 `flex: 1 1 0`，確保三個步驟方框等寬排列。驗證：如何運作區塊的三個方框寬度一致。
- [x] 8.5 Hero 標題不折行。於 `public/style.css` 將 `.hero-title` 設為 `white-space: nowrap`。驗證：「byok — Bring Your Own Key」標題在桌面寬度下不折行。

