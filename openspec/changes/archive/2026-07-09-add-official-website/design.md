## Context

目前 `byok` 缺乏一個可以快速向使用者（包含技術與非技術人員）展示價值主張的官方網站。我們需要在 `public` 資料夾下建立一個純靜態（HTML/CSS/JS）的一頁式網站。

## Goals / Non-Goals

**Goals:**

- 提供一個能立刻讓使用者理解 `byok` 「解決什麼問題」與「如何運作」的一頁式網站。
- 提供深色模式（Tech Dark）的專業科技感視覺體驗。
- 提供便於複製指令的 UI 元件。
- 網站以純 HTML + Vanilla CSS + Vanilla JS 實作，確保最低的依賴與快速載入。
- 確保 Claude CLI 在透過第三方端點執行時，能啟用 1M Token 的 Context Window 限制。

**Non-Goals:**

- 不使用任何前端框架（React, Vue, Svelte）或建置工具（Webpack, Vite）。
- 不實作多頁面路由（SPA）或部落格系統。
- 不連接任何後端 API。

## Decisions

### 採用 Vanilla HTML/CSS/JS 實作

由於只是一個輕量級介紹網站，不引入框架可以避免專案變得肥大，也免去設定 CI/CD 建置步驟的麻煩，直接將靜態檔放置於 `public/` 目錄即可。

### Tech Dark 風格與玻璃擬物化設計

網站將使用深色背景搭配高對比度的螢光色點綴，並使用半透明的背景與模糊效果（backdrop-filter）實作玻璃擬物化（Glassmorphism）卡片，打造現代、專業的開發者工具氛圍。

### 提供「複製到剪貼簿」功能

在所有指令區塊（Quick Start、Install）右側實作複製按鈕，按鈕以 SVG 剪貼簿圖示呈現（非文字），點擊後將指令複製到剪貼簿並切換為勾選圖示作為視覺回饋，2 秒後恢復原圖示。

### Quick Start Section 三分鐘上手區塊

在 Features 與 Install 之間新增獨立的 Quick Start 區塊，以兩步驟教學引導使用者：Step 1 設定 Profile（`byok config add my-profile`），Step 2 以五張卡片展示各工具啟動指令（Copilot CLI、Codex CLI、Codex App、Claude、Pi），每張卡片附官方 SVG Icon 與複製按鈕。

### 導覽列

頁面頂部加入 sticky 導覽列，含「特色、如何運作、核心功能、上手、安裝」等快速跳轉連結，方便使用者快速定位各區塊。

### Install Section with OS Tabs 安裝區塊 OS Tab 切換

Install 區塊移至頁面最後，顯示「無須依賴」徽章強調單一執行檔無需執行階段。以 OS tab（Linux / macOS / Windows）切換顯示對應平台安裝指令，避免資訊過載。

### 官方工具 Icon

各工具卡片使用官方 SVG 圖示取代 Emoji，圖示存放於 `public/icons/` 目錄：copilot.svg（GitHub Copilot）、openai.svg（OpenAI，用於 Codex CLI 與 Codex App）、anthropic.svg（Anthropic Claude）、pi.svg（Pi 自製 π 圖示）。

### 命令列單行顯示

所有命令列範例以單行顯示（`white-space: nowrap`），不出現水平捲軸（`overflow: hidden`），過長指令以 `text-overflow: ellipsis` 截斷。版面容器加寬（1280px），內部區塊加寬至 920px 以容納指令。

### Claude Context Window 後綴注入

為了讓 Claude CLI 在連接 BYOK 端點時能擁有 1M 的 Token 上限（Claude 預設行為會因非官方 API 而受限），在注入 `ANTHROPIC_MODEL` 時，無論是預設模型還是 `--model` 覆寫，皆在模型名稱後加上 `[1m]`。

## Implementation Contract

- **Behavior**:
  - 開啟 `public/index.html` 能看到具有 Tech Dark 風格的單頁網站。
  - 頁面頂部有 sticky 導覽列，含「特色、如何運作、核心功能、上手、安裝」跳轉連結。
  - Hero 區塊具備模擬終端機打字效果，標題「byok — Bring Your Own Key」不折行。
  - Feature 區塊的卡片有玻璃擬物化的視覺風格（半透明、背景模糊）。
  - How It Works 區塊的三個流程步驟方框等寬排列。
  - Quick Start 區塊以兩步驟（Step 1 設定 Profile、Step 2 啟動工具）引導使用者，五張工具卡片各附官方 SVG Icon 與複製按鈕。
  - Install 區塊位於頁面最後，顯示「無須依賴」徽章，以 OS tab 切換不同平台安裝指令。
  - 所有指令區塊的複製按鈕以 SVG 圖示呈現（剪貼簿圖示），點擊後切換為勾選圖示。
  - 命令列範例單行顯示，不出現水平捲軸。
  - `byok launch claude` 時，子程序接收到的 `ANTHROPIC_MODEL` 環境變數必定結尾包含 `[1m]`。
- **Interface / data shape**:
  - `public/index.html`: 主結構。
  - `public/style.css`: 所有樣式。
  - `public/script.js`: 負責打字動畫、複製功能與 OS tab 切換的邏輯。
  - `public/icons/`: 各工具官方 SVG 圖示（copilot.svg、openai.svg、anthropic.svg、pi.svg）。
- **Failure modes**: 若瀏覽器不支援 `navigator.clipboard`，複製按鈕應優雅降級或提示使用者手動複製。
- **Acceptance criteria**:
  - 在瀏覽器中直接開啟 `index.html` 畫面不會跑版。
  - 導覽列連結可正確跳轉至各區塊。
  - Quick Start 區塊的兩步驟教學清晰可見，五張工具卡片附帶官方 Icon。
  - OS tab 切換正常，各平台安裝指令正確顯示。
  - 複製按鈕以 SVG 圖示呈現，功能運作正常。
  - 命令列不出現水平捲軸。
  - 能夠順暢捲動瀏覽各個區段。
  - 執行 `byok launch claude --model foo` 時，能從外部驗證或經測試確保環境變數注入 `ANTHROPIC_MODEL=foo[1m]`。

## Risks / Trade-offs

- **純手工維護 CSS**: 若未來需要擴展頁面，維護原生的 CSS 可能會比較費時。
  → Mitigation: 在 `style.css` 頂端使用 CSS 變數（Custom Properties）來管理色彩、間距與字體，保持設計系統的一致性。

