## Why

`byok` 首版僅支援 `byok launch copilot`，將 `copilot` 寫死為唯一目標。使用者希望以相同的 BYOK profile 體驗啟動 OpenAI Codex CLI，並讓 `launch` 的目標工具成為可擴充參數。同時，現有 GitHub Actions 的 Pushover 通知對成功與失敗使用相同的優先權（`1`）與音效（`pushover`），無法在第一時間從通知音效區分建置/測試結果，需要依結果區分優先權層級與音效。

## What Changes

- 將 `byok launch` 的第一個位置參數正式化為「目標工具」參數，支援 `copilot` 與 `codex` 兩個值，並保留向後相容。
- 新增 `byok launch codex`：以 BYOK profile 啟動 Codex CLI，採「暫時注入、不寫入使用者設定檔」原則——在 codex 子程序環境中設定一個內部環境變數承載 API key，並透過 codex `--config` 旗標覆寫 `model`、`model_provider` 與 `[model_providers.<id>]` 的 `base_url`/`env_key`，完全不修改 `~/.codex/config.toml`。
- `byok launch codex` 支援與 copilot 對等的旗標：`--model`、`--profile`、`--config`（byok 設定檔路徑）、`-y`/`--yolo`（透傳為 codex `--yolo`）、`--` 透傳參數。
- 在執行 codex 前以 `exec.LookPath` 檢查 `codex` 可執行檔是否存在於 PATH，缺失時印出安裝提示並以 exit code 1 結束。
- 保留 copilot 既有行為不變（環境變數注入機制、旗標、錯誤處理）。
- 調整 `.github/workflows/release.yml` 與 `.github/workflows/pr-test.yml` 中的 Pushover 通知：成功時使用較低優先權（`0`，無聲/notification 音）與正向音效；失敗時使用較高優先權（`1` 或緊急優先權）與警示音效，使使用者能從通知音效立即分辨結果。
- 更新 `README.md`：新增 `byok launch codex` 操作說明、Codex BYOK 運作原理段落，並補上 Copilot CLI BYOK 與 Codex CLI BYOK 的官方文件連結。
- 擴充 `AGENTS.md`：在既有 Spectra 指令區塊之外，新增本專案的架構總覽（Go cobra CLI、模組路徑、`cmd`/`internal/config`/`internal/runner`/`internal/version` 套件職責、設定檔位置）與開發規範（BYOK 注入僅作用於子程序、父程序環境不變、不寫入使用者設定檔、profile 解析與錯誤處理、測試約定），並明訂「任何架構或規範變更須同步更新 AGENTS.md」的維護規則。

## Non-Goals (optional)

本變更不會：

- 修改使用者的 `~/.codex/config.toml` 或 `~/.byok/config.yaml`（BYOK 採執行期覆寫，啟動結束後不留痕跡）。
- 支援 `openai` 以外的 provider 類型（copilot 與 codex 皆維持首版僅支援 OpenAI 相容端點）。
- 為 codex 引入持久化的 codex profile 檔（方案 B 的臨時 profile 檔做法已評估並摒棄）。
- 變更 copilot 既有的環境變數注入機制。
- 變更版本號管理或 Release 產物結構。

## Capabilities

### New Capabilities

- `byok-codex-launch`: 以 BYOK profile 暫時注入環境變數與 codex `--config` 覆寫來啟動 Codex CLI，不寫入使用者設定檔；涵蓋 `--model`、`--profile`、`--config`、`-y`/`--yolo`、`--` 透傳、codex 可執行檔存在性檢查、profile 解析與錯誤處理。
- `byok-ci-notification`: GitHub Actions 中 Pushover 通知的優先權層級與音效規範，依建置/測試成功或失敗區分，涵蓋 Release workflow 的 `notify` job 與 PR Tests workflow 的通知步驟。
- `byok-agent-docs`: `AGENTS.md` 紀錄本專案的架構總覽與開發規範，並規定任何改變架構或規範的變更須同步更新 `AGENTS.md`，作為 AI agents 與人類貢獻者共用的權威文件。

### Modified Capabilities

- `byok-launch`: `byok launch` 的第一個位置參數正式化為目標工具選擇器，依值分派至 copilot 或 codex 啟動流程；不支援的目標工具印出錯誤並 exit 1。

## Impact

- Affected specs:
  - New: `byok-codex-launch`, `byok-ci-notification`, `byok-agent-docs`
  - Modified: `byok-launch`
- Affected code:
  - New:
    - `internal/runner/codex.go`（Codex BYOK 環境變數與 `--config` 參數建置）
    - `internal/runner/codex_test.go`
    - `cmd/launch_codex.go`（codex 啟動子流程，與 copilot 對等）
    - `cmd/launch_codex_test.go`
  - Modified:
    - `cmd/launch.go`（目標工具參數分派、保留 copilot 行為）
    - `cmd/launch_test.go`
    - `internal/runner/runner.go`（共用啟動邏輯，必要時抽出泛用 Launch）
    - `.github/workflows/release.yml`（Pushover 優先權/音效依結果區分）
    - `.github/workflows/pr-test.yml`（Pushover 優先權/音效依結果區分）
    - `README.md`（新增 codex 操作說明、運作原理、官方文件連結）
    - `AGENTS.md`（新增架構總覽與開發規範區塊，保留既有 Spectra 指令區塊）
- Affected systems: 無外部系統變更；Pushover 通知僅調整 `priority`/`sound` 參數。
