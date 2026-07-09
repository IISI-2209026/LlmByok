## Why

我們需要一個一頁式的官方介紹網站，讓不管是開發者或是非技術人員的潛在使用者，都能快速理解 `byok` 的價值：一行指令，把自己的 API Key 暫時注入 Copilot / Codex / Claude / Pi，Shell 環境完全不受污染。因為每次手動設定變數既繁瑣又容易污染環境，我們希望讓大家知道使用自己的 API 金鑰變得很容易。此外，針對 Claude CLI，官方要求透過第三方端點連線時，需在模型名稱後加上 `[1m]` 來啟用 1M token 的 context window 限制，這也需要一併在此變更中實作。

## What Changes

- 新增 `public/` 目錄用於存放網站靜態資源（不使用框架，純 HTML + CSS + Vanilla JS）。
- 實作一頁式介紹頁面，採用「深色終端機」風格（Tech Dark）及玻璃擬物化（Glassmorphism）卡片設計。
- **Navigation 導覽列**：頂部 sticky 導覽列，含「特色、如何運作、核心功能、上手、安裝」快速跳轉連結。
- **Hero 區塊**：展示核心價值與終端打字動畫，標題不折行，提供下載連結。
- **Problem 區塊**：重點傳達「使用自己的API金鑰變的很容易」，展示手動 export 與一行指令的 Before/After 比較。
- **How It Works 區塊**：展示 API 金鑰注入的流程圖，三個步驟方框等寬排列。
- **Features 區塊**：列出核心功能（Profile 管理、支援多種工具等）。
- **Quick Start 區塊**：三分鐘上手，兩步驟教學 — Step 1 設定 Profile（`byok config add my-profile`），Step 2 啟動工具（Copilot CLI、Codex CLI、Codex App、Claude、Pi 五張卡片，各附官方 SVG Icon 與複製按鈕）。
- **Install 區塊**：移至頁面最後，顯示「無須依賴」徽章，以 OS tab 切換（Linux / macOS / Windows）顯示對應安裝指令，所有指令範例附帶 SVG Icon 複製按鈕（非文字）。
- 新增 `public/icons/` 目錄存放各工具官方 SVG 圖示（copilot.svg、openai.svg、anthropic.svg、pi.svg）。
- 命令列範例單行顯示，不出現水平捲軸。
- 針對 `byok launch claude` 的執行邏輯，自動在注入的 `ANTHROPIC_MODEL` 模型名稱後方附加 `[1m]`，以確保 Claude CLI 擁有正確的 Context Window。

## Capabilities

### New Capabilities

- `official-website`: `byok` 一頁式官方介紹網站。

### Modified Capabilities

- `byok-claude-launch`: 修改 Claude 模型注入邏輯，確保模型名稱後方附加 `[1m]` 以支援 context window。

## Impact

- Affected specs: `official-website`, `byok-claude-launch`
- Affected code:
  - New: `public/index.html`
  - New: `public/style.css`
  - New: `public/script.js`
  - New: `public/icons/copilot.svg`
  - New: `public/icons/openai.svg`
  - New: `public/icons/anthropic.svg`
  - New: `public/icons/pi.svg`
  - Modified: `internal/runner/claude.go`

