## Context

`byok` 已透過 Release workflow（`byok-release` spec）在 develop/main 推送時產生多平台預建二進位資產，命名為 `byok-<version>-<os>-<arch>.<ext>`：main 為穩定版 `<base>`（prerelease=false），develop 為預發布 `<base>-dev.<run_number>`（prerelease=true）。`internal/version/version.go` 的 `Version` 字面值為 canonical base，建置時以 ldflags 注入完整版號。目前沒有任何機制讓使用者得知是否過舊或一鍵升級，安裝也只說明從原始碼建置一途。

本變更在不引入第三方相依的前提下，以 Go 標準庫實作 channel 感知的自我更新與啟動版本檢查，並補完 README 安裝說明。

## Goals / Non-Goals

**Goals:**

- `byok update` 依當前版本 channel（或 `--channel` 覆寫）下載並替換執行檔。
- 非 launch/update 指令執行後印出有新版本的提示。
- README 提供兩種安裝方式（`go install` 與 Releases 預建二進位）。
- Release workflow 以 conventional commit 分類產生 changelog 作為 release body。
- 僅用 Go 標準庫。

**Non-Goals:**

- 不做排程/常駐自動更新。
- 不自動切換 channel（預設）；僅在使用者明確 `--channel` 時覆寫。
- 不處理套件管理器（Homebrew/winget/scoop）。
- 啟動檢查不快取（every-run），但短 timeout 且失敗靜默。
- 不下載/安裝 copilot 或 codex 目標工具。
- changelog 不處理跨多個 release tag 的歷史彙整，僅取自上一個 tag 至 HEAD。

## Decisions

### Decision: Channel 判定以版本字串含 `-dev.` 為準

當前版本（`internal/version.Version`）含子字串 `-dev.` 視為 dev channel，否則視為 stable channel。dev channel 查詢時只考慮 `prerelease=true` 的 release，stable channel 只考慮 `prerelease=false`。理由：與 Release workflow 既有版號格式（`<base>-dev.<run_number>` vs `<base>`）完全對齊，無需額外 metadata。替代考量：以 Git tag pattern 過濾，但 GitHub Releases 的 `prerelease` 旗標已由 workflow 設定，直接用更可靠。

### Decision: 以 GitHub Releases API 列舉並選最新 release

呼叫 `GET https://api.github.com/repos/IISI-2209026/LlmByok/releases`，依 channel 過濾 `prerelease` 旗標，並取符合 channel 中 `tag_name` 語意最新者（dev 以 tag 中 `-dev.<run_number>` 的數字最大者；stable 以 semver 最大者）。理由：單一端點即可涵蓋兩 channel，無需分開呼叫 `/releases/latest`（該端點只回 stable）。替代：呼叫 `/releases/latest` 處理 stable、`/releases` 處理 dev——增加分支，統一用 `/releases` 較簡潔。

### Decision: 資產選擇以 GOOS/GOARCH 對應命名比對

依 `runtime.GOOS` 與 `runtime.GOARCH` 組出資產名樣板 `byok-<version>-<os>-<arch>.<ext>`，`<ext>` 為 `zip`（windows）或 `tar.gz`（linux/darwin），在所選 release 的 `assets` 中以名稱完全比對找出下載 URL。理由：直接沿用 `byok-release` spec 既有的資產命名契約，不需 release 端任何變更。

### Decision: 原子自我替換採平台對應策略

Unix（linux/darwin）：將新二進位寫入同目錄暫存檔，`os.Chmod 0755` 後以 `os.Rename` 覆蓋原執行檔（rename 在同檔案系統為原子。Windows：`os.Rename` 對使用中執行檔會失敗，改以 `MoveFileEx` 配 `MOVEFILE_REPLACE_EXISTING` 旗標透過 `syscall` 完成。新二進位來源檔案路徑以 `os.Executable()` 解析。理由：跨平台可靠替換正在執行的二進位是本功能最大風險點，必須明確分平台處理。替代：要求使用者手動重裝——違反 `update_action=self-replace` 的需求。

### Decision: 啟動版本檢查置於 root 指令 PostRunE，對 launch 與 update 跳過

在 `cmd/root.go` 的根指令設定 `PostRunE`，於子指令完成後執行版本檢查：呼叫 updater 查詢當前 channel 最新 release，若較新則在 stderr 印出一行提示（含更新指令 `byok update`）。`launch` 子指令（互動式長時間）與 `update` 子指令（自己做檢查）透過在各自 `RunE` 內標記跳過（例如回傳一個 sentinel 或在 PostRunE 檢查 `cmd.Name()`）。每次皆查詢（every-run），HTTP client timeout 設 3 秒，任何網路/解析錯誤靜默不印。理由：before-command 會拖慢每次啟動；after-command 對 launch 不適合，故 launch 跳過。

### Decision: 不引入第三方 Go 相依

資產解壓以標準庫 `archive/zip`（windows）與 `compress/gzip` + `archive/tar`（linux/darwin）實作；HTTP 以 `net/http`；GitHub API 回應以 `encoding/json` 解析。理由：byok 目前無任何第三方執行期相依（cobra/yaml 為既有），新增自我更新不應引入重依賴。

### Decision: `byok update` 提供 `--check` 旗標只檢查不替換

預設 `byok update` 執行下載與替換；`--check` 僅查詢並印出是否有新版本與最新版號，不下載、不替換。理由：讓使用者能在腳本中先檢查；也作為啟動檢查的對應手動查詢入口。

### Decision: `byok update` 提供 `--channel` 旗標覆寫 channel 判定

`byok update` 新增 `--channel` 旗標，接受 `prerelease` 或 `release`（映射至內部 `dev`/`stable` channel）；未傳時依當前版本自動判定（含 `-dev.` → dev，否則 stable）。傳入 `--channel prerelease` 時即使當前為 stable 版也查 `prerelease=true` release，反之亦然。不合法值（非 `prerelease`/`release`）印錯誤 exit 1。理由：使用者可能以 stable 版安裝但想提前測試 dev 版，或反之；讓使用者明確覆寫比強迫重裝更友善，同時保留預設安全的 in-channel 行為。替代：獨立 `byok switch-channel` 指令——過度設計，單一旗標即足。

### Decision: Release changelog 以 conventional commit 分類產生

Release workflow 的 `release` job 在建立 GitHub Release 前，以 `git log --pretty=format:"%s" <prev_tag>..HEAD` 取得自上一個 release tag 至 HEAD 的 commit subject，依 conventional commit prefix 分類：`feat:` → 新增功能、`refactor:`/`perf:` → 優化功能、`fix:` → 修復功能，其餘 prefix（`docs:`/`chore:`/`ci:`/`build:` 等）歸「其他」段落或省略。分類後輸出 Markdown 作為 release body（取代目前 `generate_release_notes: true`）。無上一個 tag 時（首次發布）以全部 commit 為範圍。以 workflow 內 shell step + `git log`/`grep`/`sed` 實作，不引入第三方 GitHub Action。理由：專案已採 conventional commits（feat/fix/docs/refactor），分類 changelog 對使用者更有意義；GitHub 自動產生的 release notes 僅列 PR/commit 清單不夠友善。替代：`mikepenz/release-changelog-builder-action`——引入第三方相依且設定複雜，shell 方案足夠。

## Implementation Contract

### Behavior

- 執行 `byok update`：解析當前版本 → 判定 channel（或以 `--channel` 覆寫）→ 查詢 GitHub Releases 該 channel 最新版 → 比較版本。若已有最新，印「已是最新版本 (<version>)」並 exit 0。若有新版，印「更新中：<current> → <latest>」→ 下載對應平台資產 → 解壓取出 `byok`（或 `byok.exe`）→ 原子替換 `os.Executable()` 回傳的當前執行檔路徑 → 印「已更新至 <latest>，請重新執行 byok」並 exit 0。
- 執行 `byok update --check`：只印「最新版本：<latest>（目前：<current>）」或「已是最新版本」，不修改檔案。
- 執行 `byok update --channel prerelease`（或 `release`）：以指定 channel 覆寫自動判定，查詢該 channel 最新版並依上述流程更新；`--channel` 與 `--check` 可合併使用。
- 執行任何非 `launch`／非 `update` 的 `byok` 子指令後：若查到較新版本，在 stderr 印一行「新版本可用：<latest>（目前：<current>）；執行 `byok update` 更新」。若查詢失敗或無新版，不印任何訊息。

### Interface / data shape

- `internal/updater` 套件公開：
  - `func Channel(v string) string` — 回傳 `"dev"` 或 `"stable"`。
  - `func LatestRelease(ctx context.Context, channel string) (Release, error)` — `Release` 含 `Tag string`（如 `v0.1.1` 或 `v0.1.1-dev.42`）、`Version string`（去 `v` prefix）、`Prerelease bool`、`Assets []Asset`；`Asset` 含 `Name string`、`DownloadURL string`。
  - `func IsNewer(current, latest string) (bool, error)` — 跨 channel 不比較（僅同 channel 內比較）；dev 以 `-dev.<N>` 數字比較，stable 以 semver 比較。
  - `func DownloadAndReplace(ctx context.Context, rel Release, goos, goarch string) error` — 選資產、下載、解壓、原子替換 `os.Executable()`。
- `cmd/update.go`：`byok update` 子指令，旗標 `--check bool` 與 `--channel string`（接受 `prerelease`/`release`，空字串代表自動判定）；`RunE` 內呼叫 updater 並印訊息。
- `cmd/root.go`：根指令 `PostRunE` 執行啟動檢查；以 `cmd.Name()` 為 `"launch"` 或 `"update"` 時跳過。
- GitHub API 呼叫：`GET https://api.github.com/repos/IISI-2209026/LlmByok/releases`，需 `Accept: application/vnd.github+json` header；不加 token（公開 repo 讀取限額足夠，遇 rate limit 靜默失敗）。

### Failure modes

- 網路失敗／timeout／rate limit：`byok update` 印錯誤並 exit 1；啟動檢查靜默不印。
- 找不到對應平台資產：`byok update` 印「找不到 <goos>/<goarch> 資產」並 exit 1。
- 下載雜湊不符/解壓失敗：印錯誤並 exit 1，不改動原執行檔。
- 替換失敗（如權限不足）：印錯誤並 exit 1；暫存檔清理。
- 當前版本為 `dev`（開發建置、無 `-dev.`）: channel 判定為 stable，查 stable release；`IsNewer("dev", ...)` 一律不視為更新（避免開發中誤升）。
- `--channel` 值非 `prerelease`/`release`：印「不合法的 channel 值」並 exit 1，不發 API 請求。
- Release changelog step 無上一個 tag：以全部歷史 commit 為範圍，不中斷發布。

### Acceptance criteria

- `go test ./internal/updater/...`：以 stub HTTP server 模擬 GitHub Releases 回應，測試 `Channel`、`LatestRelease`（dev/stable 過濾）、`IsNewer`（同 channel 各情境）、`DownloadAndReplace`（以 fake release/asset 與 stub 二進位，驗證暫存檔被寫入並 chmod、原路徑被替換；以介面注入替換目標路徑以便測試）。
- `go test ./cmd/...`：`TestUpdateCmd_NoUpdate`（已是最新，exit 0）、`TestUpdateCmd_CheckOnly`（`--check` 不修改檔案）、`TestUpdateCmd_ChannelOverride`（`--channel prerelease` 以 stable 版執行，斷言查 prerelease release）、`TestUpdateCmd_InvalidChannel`（不合法值 exit 1）、`TestUpdateCmd_NetworkError`（exit 1）、`TestRootPostRun_SkipsLaunch`（launch 後不觸發檢查）、`TestRootPostRun_PrintsHintOnNewer`（以 stub updater 注入新版本，stderr 含 `byok update` 字樣）。
- 手動：`go run main.go update --check` 對真實 repo 顯示最新 stable 版號；`go run main.go update --channel prerelease --check` 顯示最新 dev 版號。
- `go vet ./...` 通過。
- Release workflow：觸發一次 develop push，檢查產生的 GitHub Release body 含「新增功能」「優化功能」「修復功能」分類標題且內容對應 commit subject。

### Scope boundaries

- In scope：`internal/updater` 套件、`cmd/update.go`、`cmd/root.go` 的 PostRunE、`internal/version` 的 channel 輔助、README 安裝段落與 update 說明、`.github/workflows/release.yml` 的 changelog 產生步驟。
- Out of scope：套件管理器 formula、copilot/codex 安裝、channel 自動切換（預設）、跨多 release tag 的歷史 changelog 彙整。

## Risks / Trade-offs

- [正在執行的二進位自我替換跨平台失敗] → Windows 用 `MoveFileEx` + `MOVEFILE_REPLACE_EXISTING`；Unix 用 `os.Rename`；先寫暫存檔成功才替換，失敗不動原檔。
- [GitHub API 未授權限額] → 公開 repo 讀取限額對單機每日 every-run 足夠；遇 403 rate limit 靜默失敗，不阻斷指令。
- [every-run 啟動檢查拖慢] → HTTP timeout 3 秒、查詢在 PostRunE（指令已完成）、失敗靜默；使用者可設 `BYOK_NO_UPDATE_CHECK=1` 環境變數跳過（本變更順帶支援）。
- [dev 版 `IsNewer` 比較需 parse run_number] → 以 `-dev.<N>` 正則取數字比對；parse 失敗視為不可比較、不更新。
- [資產 tar.gz 內含目錄結構] → 解壓時在 tar 成員中找名為 `byok`（或 `byok.exe`）的檔案，不限路徑前綴。
