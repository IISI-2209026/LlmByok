<!-- SPECTRA:START v1.0.2 -->

# Spectra Instructions

This project uses Spectra for Spec-Driven Development(SDD). Specs live in `openspec/specs/`, change proposals in `openspec/changes/`.

## Use `$spectra-*` skills when:

- A discussion needs structure before coding → `$spectra-discuss`
- User wants to plan, propose, or design a change → `$spectra-propose`
- Tasks are ready to implement → `$spectra-apply`
- There's an in-progress change to continue → `$spectra-ingest`
- User asks about specs or how something works → `$spectra-ask`
- Implementation is done → `$spectra-archive`
- Commit only files related to a specific change → `$spectra-commit`

## Workflow

discuss? → propose → apply ⇄ ingest → archive

- `discuss` is optional — skip if requirements are clear
- Requirements change mid-work? `ingest` → resume `apply`

## Parked Changes

Changes can be parked（暫存）— temporarily moved out of `openspec/changes/`. Parked changes won't appear in `spectra list` but can be found with `spectra list --parked`. To restore: `spectra unpark <name>`. The `$spectra-apply` and `$spectra-ingest` skills handle parked changes automatically.

<!-- SPECTRA:END -->


# 專案架構

`byok` 是一支以 Go 1.26+ 與 [cobra](https://github.com/spf13/cobra) 建構的命令列工具，模組路徑為 `github.com/IISI-2209026/LlmByok`，入口為 `cmd/byok/main.go`。它以 BYOK（Bring Your Own Key）profile 暫時啟動 Copilot 或 Codex CLI，不修改父程序環境或使用者設定檔。

## 套件職責

| 套件                 | 職責                                                                 |
| -------------------- | -------------------------------------------------------------------- |
| `cmd/byok`           | 程式入口（`main` package），呼叫 `cmd.NewRoot` 建立根指令。            |
| `cmd`                | cobra 指令定義與目標工具分派（`launch copilot` / `launch codex` / `launch codex-app` / `launch claude`）、`config` 子指令（`add`/`update`/`delete`/`list`/`set-default`）、`update` 子指令。 |
| `internal/config`    | YAML profile 的載入、儲存與驗證；設定檔預設位於 `~/.byok/config.yaml`；金鑰解析（`KeyResolver` 介面、`DefaultResolver`：keychain 優先 → 明碼 fallback）。 |
| `internal/runner`    | BYOK 環境變數建置與子程序啟動（`Launch` for copilot、`LaunchCodex` for codex、`LaunchCodexApp` for codex app、`LaunchClaude` for claude）。 |
| `internal/secret`    | OS keychain 抽象層（zalando/go-keyring）：`Store`/`Load`/`Delete`/`Exists`，service=`byok`、key=`profile:<name>`。 |
| `internal/updater`   | 自我更新：channel 判定、GitHub Releases 查詢、平台資產選擇、下載與跨平台執行檔原子替換。 |
| `internal/version`   | 版本號嵌入（透過 ldflags 注入）。                                      |

## 設定檔

- 設定檔位置：`~/.byok/config.yaml`（可用 `--config` 覆寫）。
- 每個 profile 包含 `name`、`provider`、`api_base`、`api_key`（omitempty，可選）、`default_model`。
- 預設 provider 為 `openai`（空字串回退為 `openai`）；首版僅支援 `openai` provider 類型。
- API 金鑰以 OS keychain 為主要儲存（`byok config add`/`update` 時以 `--key-storage keychain`（預設）指定），明碼 `api_key` 為 fallback；`launch` 時由 `KeyResolver` 自動解析。

# 開發規範

- **BYOK 注入僅作用於子程序** — 環境變數只注入到 `copilot` / `codex` / `codex app` / `claude` 子行程，父程序（Shell）與系統環境永不被改變。
- **不寫入使用者設定檔** — `byok` 不會修改 `~/.byok/config.yaml`、`~/.codex/config.toml`、`~/.claude/settings.json` 或任何 Copilot/Codex/Claude 設定檔；codex 連線覆寫僅透過命令列 `--config` 旗標傳遞；claude 僅透過環境變數注入。
- **Profile 解析錯誤印訊息並 exit 1** — 設定檔不存在、profile 找不到、未設 `default_profile`、非 `openai` provider 等情境，皆印出錯誤與提示後以非零結束碼退出。
- **預設 provider 為 `openai`** — `provider` 欄位為空時回退為 `openai`；非 `openai` 一律拒絕。
- **金鑰以 OS keychain 為主要儲存、明碼 `api_key` 為 fallback** — `byok config add`/`update` 預設以 `--key-storage keychain` 將金鑰存入 keychain（service=`byok`、key=`profile:<name>`）並清除設定檔明碼；可用 `--key-storage plaintext` 改存明碼至設定檔。`delete` 移除 profile 時同步清理 keychain（盡力）。`launch` 時 `KeyResolver` 依 keychain → 明碼順序解析，兩者皆無則報錯。Linux 需 secret-service daemon（gnome-keyring/KWallet）；無 daemon 時回傳 backend-unavailable，可改用 `--key-storage plaintext`。`add`/`update` 支援終端互動模式（未傳欄位旗標時觸發，需 TTY，非 TTY 印錯 exit 1）。
- **測試以 `go test ./... -race` 執行** — 新增功能須伴隨單元/整合測試，並以 `-race` 確認無資料競爭。
- **`byok update` 自我更新** — `byok update` 依當前版本 channel（含 `-dev.` 為 dev、否則 stable）查詢 GitHub Releases，下載對應平台資產（`byok-<version>-<goos>-<goarch>.<ext>`）並原子替換執行檔。`--check` 只查詢不替換；`--channel prerelease|release` 覆寫 channel 判定。啟動版本檢查：`launch`/`update` 以外子指令完成後以 3 秒 timeout 查詢，較新時在 stderr 印提示；`BYOK_NO_UPDATE_CHECK=1` 停用；任何錯誤靜默不影響 exit code。
- **Release changelog 以 conventional commit 分類產生** — Release workflow 於建立 GitHub Release 前以 `git log` 取 commit subject，依 prefix 分類（`feat:` → 新增功能、`refactor:`/`perf:` → 優化功能、`fix:` → 修復功能）輸出 Markdown 至 `changelog.md`，作為 release body。

# 版本號機制

- **Canonical base 來源**：`internal/version/version.go` 的 `Version` 字面值為 canonical base 版號（semver、無 `v` prefix、無後綴），目前為 `0.1.0`。Makefile 與 Release workflow 皆以 `sed` 讀取此字面值，不引入額外 VERSION 檔或以 Git tag 為來源。
- **develop 預發布**：推送 develop → Release workflow 產生預發布，二進位版號 `<base>-dev.<run_number>`、tag `v<base>-dev.<run_number>`、`prerelease: true`。`run_number` 取自 `github.run_number`，確保每次推送唯一、不撞 tag。
- **main 穩定發布**：推送 main → Release workflow 產生穩定發布，二進位版號 `<base>`、tag `v<base>`、`prerelease: false`。
- **晉升流程**：
  1. develop 累積預發布至可發布狀態。
  2. merge develop → main 並推送 main → Release workflow 自動產生穩定發布 `v<base>`。
  3. 於 develop 執行 `byok-bump-version` skill 將 base 晉升到下一個 patch（或 minor/major）。
  4. push 到 develop，使下一輪預發布使用更高的 base（如 `0.1.1-dev.N`），下一輪 main 發布即為 `0.1.1`。
- **bump skill**：`.github/skills/byok-bump-version/SKILL.md` 負責 bump + commit + push 到 develop；不建立 Git tag、不 push 到 main、不強推。在 main 分支執行時中止。

# 維護規則

任何改變以下項目的變更，**必須在相同變更內更新 `AGENTS.md` 對應段落**：

- 套件結構（新增/移除/重新命名套件、變更套件職責）
- BYOK 注入機制（環境變數名稱、`--config` 覆寫格式、子程序啟動方式）
- 設定檔格式（`~/.byok/config.yaml` 欄位、預設路徑）
- CLI 介面（指令、旗標、位置參數、錯誤訊息）
- 已記錄於「開發規範」的行為

> ⚠ Spectra 區塊（`<!-- SPECTRA:START -->` 至 `<!-- SPECTRA:END -->`）由 Spectra CLI 自動管理，**不得手動編輯**。
