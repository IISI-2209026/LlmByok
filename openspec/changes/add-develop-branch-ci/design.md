## Context

目前 `.github/workflows/release.yml` 只在 push 至 `main` 時觸發，產出穩定版 GitHub Release（tag `v<version>`，非 prerelease）。版本號來源是 `internal/version/version.go` 的 `var Version`，由 workflow 透過 `sed` 讀取後以 ldflags 注入二進位檔。

專案目前只有 `main` 與 feature 分支，沒有 `develop` 分支。使用者希望新增 `develop` 分支做整合測試，合併至 develop 時也要觸發 CI，但產出帶 `-dev` 後綴的測試版本（prerelease），與穩定版區隔。

## Goals / Non-Goals

**Goals:**

- 讓 push 至 `develop` 分支觸發 release workflow
- develop 分支的二進位檔版本號注入 `<version>-dev`
- develop 分支的 GitHub Release 標記為 prerelease，tag 為 `v<version>-dev`
- main 分支維持現有穩定版行為不變

**Non-Goals:**

- 不實作完整 GitFlow（release 分支、hotfix 分支）
- 不為 feature 分支提供 CI build
- 不修改 `internal/version/version.go` 的版本號來源
- 不拆分為獨立的 dev-build workflow 檔案

## Decisions

### 單一 workflow 檔案搭配 branch 條件判斷

採用單一 `release.yml` 同時處理 main 與 develop，而非拆成兩個 workflow 檔案。

理由：matrix build 邏輯（4 平台、ldflags 注入、打包）完全相同，差異只有版本號後綴、tag 格式與 prerelease 標記。拆兩個檔案會重複 build steps，增加維護成本。單一檔案以 `github.ref` 判斷分支，差異點集中在一處。

### 版本後綴由 workflow 動態附加，不修改 version.go

`internal/version/version.go` 的 `Version` 仍是單一真相來源（base 版本號）。`-dev` 後綴由 workflow 在讀取版本後動態組裝，僅在 develop 分支注入二進位檔與 Release tag。

理由：避免在 version.go 中維護分支差異，降低衝突風險；main 與 develop 共用同一個 base 版本號語意清楚。

### 以 github.ref 分支名稱判斷後綴與 prerelease

workflow 在「Read version」step 後新增一步判斷 `github.ref`：

- `refs/heads/main` → `VERSION_SUFFIX=""`、`PRERELEASE=false`、`TAG=v<version>`
- `refs/heads/develop` → `VERSION_SUFFIX="-dev"`、`PRERELEASE=true`、`TAG=v<version>-dev`

最終注入版本為 `${VERSION}${VERSION_SUFFIX}`，tag 為 `${TAG}`，release action 的 `prerelease` 欄位為 `${PRERELEASE}`。

理由：直接使用 GitHub 內建的 `github.ref`，無需額外標籤或外部參數，判斷邏輯透明可審。

## Implementation Contract

**行為（operator 可觀察）**：

- push 至 `main`：workflow 執行，產出 `v<version>` tag 的 GitHub Release，非 prerelease，二進位檔 `byok version` 輸出 `<version>`。
- push 至 `develop`：workflow 執行，產出 `v<version>-dev` tag 的 GitHub Release，標記為 prerelease，二進位檔 `byok version` 輸出 `<version>-dev`，archive 檔名為 `byok-<version>-dev-<os>-<arch>.<ext>`。
- push 至其他分支：workflow 不執行。

**介面 / 資料形狀**：

- workflow `on.push.branches` 改為 `[main, develop]`。
- 新增 step 輸出三個 step outputs：`version_suffix`、`prerelease`、`tag`。
- build step 的 ldflags 版本改用 `${{ steps.version.outputs.full_version }}`（base + suffix）。
- release action 的 `tag_name` 改用 `${{ steps.version.outputs.tag }}`，新增 `prerelease: ${{ steps.version.outputs.prerelease }}`。

**失敗模式**：

- 若 `github.ref` 不符合 main 也不符合 develop（不應發生，因 on.push 已限制），suffix 預設為空、prerelease 為 false，行為等同 main，避免意外產生 dev tag。

**驗收條件**：

- 在 `develop` 分支 push 後，GitHub Actions 執行 release workflow，產生 tag 形如 `v0.1.0-dev` 的 prerelease，附帶 4 個平台 archive，archive 檔名含 `-dev`。
- 在 `main` 分支 push 後，行為與改動前一致（tag `v0.1.0`、非 prerelease）。
- 下載 develop build 的二進位執行 `byok version`，輸出 `0.1.0-dev`。
- 在 feature 分支 push，workflow 不執行（可由 `on.push.branches` 限制保證）。

**範圍邊界**：

- In scope：修改 `.github/workflows/release.yml`。
- Out of scope：`internal/version/version.go`、Makefile、README 版本管理說明（除非需要補充 develop 流程，本變更不更動）。

## Risks / Trade-offs

- [Risk] 同一 base 版本號在 develop 多次 push 會嘗試建立已存在的 `v<version>-dev` tag，導致 release action 失敗 → Mitigation：develop 分支每次 push 前應更新 base 版本號，或後續變更改用含 commit SHA 的 tag（本變更不做，列為 Non-Goal）。
- [Risk] 單一 workflow 檔案條件分支增多後可讀性下降 → Mitigation：差異點集中於一個 step，並以 step outputs 命名清楚表達語意。
- [Trade-off] 不使用 GitHub Environment 區分 staging/production → 較簡單但缺乏環境層級的 secret 隔離，目前專案無此需求。
