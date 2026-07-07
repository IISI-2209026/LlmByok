## Why

目前每個 profile 只能設定單一 `default_model`，使用者每次啟動不同模型都得記住並透過 `--model` 指定，缺乏便利性。希望在 `byok launch <target>` 未帶 `--model` 時，能從該 profile 的候選模型清單中互動選擇（上下鍵），只有一個候選時直接使用，讓切換模型更直覺。為了承載多個候選模型，config 的 profile 結構與 CLI 介面需要一併調整。

## What Changes

- **BREAKING**：`byok config add/delete/set-default/update` 改為以位置參數接收 profile 名稱（`byok config add <profile name>`），取代原本的 `--name` 旗標。
- **BREAKING**：profile 結構以候選模型清單 `models` 取代單一 `default_model` 欄位；`default_model` 欄位移除。
- 新增 `byok config set-models <profile name>` 子指令（掛於 `config` 之下），可設定該 profile 的多個候選模型（取代既往 default model 的設定用途）。
- `byok config list` 輸出每個 profile 的模型清單（多個模型以逗號或換行列出）。
- `byok launch <target>` 在未帶 `--model` 時的模型解析行為調整：
  - 候選模型僅一個 → 直接使用該模型。
  - 候選模型多個且 stdin 為終端機 → 顯示互動式上下鍵選單讓使用者選擇。
  - 候選模型多個但 stdin 非終端機 → 印出錯誤並退出（避免在非互動環境卡住）。
  - 帶 `--model` 時 → 一律以 `--model` 指定值覆寫，不做互動選單。
  - 候選模型為空 → 印出錯誤並退出，提示使用 `byok config set-models`。
- 互動式模型選單的終端機處理與視覺回饋（實作精煉）：
  - 進入選單時將 stdin 切換為 raw mode（即時讀鍵、關閉本地回顯與行緩衝），使方向鍵以 ANSI 序列正確送達；離開時還原。
  - 於 stdout 啟用虛擬終端機（VT）處理（Windows 透過 `SetConsoleMode`，Unix 原生支援），使 ANSI 反白與游標控制序列正確渲染。
  - 選單以「❯ 游標 + 反白」標記選取列原地重繪；Ctrl-C 或 Esc 取消選擇並以非零結束碼退出。
- `byok config list` 表格欄位對齊（實作精煉）：
  - 改以「顯示寬度感知」動態計算各欄寬（每欄寬 = 標題與各列該欄顯示寬度的最大值），避免 CJK 標題（名稱/模型/來源佔 2 欄）與超長模型欄造成欄位錯位。

## Non-Goals (optional)

- 不改變 keychain / 金鑰儲存機制與 `--key-storage` 行為。
- 不改動 `--profile`、`--config`、`--yolo`、`--` 透傳等既有旗標語意。
- 不在此變更中新增模型清單的排序、分群或標記 favorite 等後續擴充功能。
- 不改動 launch 對各 target（copilot/codex/codex-app/claude/pi）的環境變數注入邏輯，僅調整「模型如何決定」這一步。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `byok-config`: profile 結構由單一 `default_model` 改為候選模型清單 `models`；`add/delete/set-default/update` 改以位置參數接收 profile 名稱；新增 `config set-models` 子指令；`list` 顯示每個 profile 的模型清單，並以顯示寬度感知的動態欄寬對齊各欄。
- `byok-launch`: `launch <target>` 未帶 `--model` 時的模型解析改為依候選模型數量決定（單一直接使用 / 多個互動選單 / 無則錯誤），`--model` 仍為明確覆寫；互動選單以 raw mode + VT 處理渲染「❯ 游標 + 反白」並支援 Ctrl-C/Esc 取消。

## Impact

- Affected specs: `byok-config`, `byok-launch`
- Affected code:
  - Modified:
    - `internal/config/config.go` — `Profile` 結構調整（`DefaultModel` → `Models []string`）、載入/儲存與遷移舊格式
    - `internal/config/interactive.go` — 互動模式欄位調整（模型清單輸入）
    - `cmd/config.go` — `add/delete/set-default/update` 改位置參數、`list` 輸出模型清單
    - `cmd/launch.go`、`cmd/launch_codex.go`、`cmd/launch_codex_app.go`、`cmd/launch_claude.go`、`cmd/launch_pi.go` — 模型解析改為候選清單邏輯與互動選單
    - `internal/runner/runner.go` 及各 target runner — 接收解析後的單一模型
    - `README.md`、`AGENTS.md` — CLI 用法與範例更新
  - New:
    - `cmd/set_models.go` — `byok config set-models <profile name>` 子指令實作（掛於 `config` 指令下）
    - `internal/config/models.go` — 候選模型清單的設定/驗證與互動選單邏輯（raw mode、VT 處理、游標反白、Ctrl-C/Esc 取消）
    - `internal/config/models_windows.go` — Windows console 虛擬終端機處理（`SetConsoleMode` 啟用 `ENABLE_VIRTUAL_TERMINAL_PROCESSING`）
    - `internal/config/models_unix.go` — 非 Windows 平台 VT 處理 no-op
  - Removed: (none)
