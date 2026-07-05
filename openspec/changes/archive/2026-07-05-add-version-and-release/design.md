## Context

byok CLI 目前沒有任何版本號管理或發布自動化。Makefile 的 build target 直接 `go build`，不注入版本資訊。每次開發完成後無版本可追蹤、無可分發執行檔。

現況：
- `Makefile` build target：`go build -o byok .`，無 ldflags
- `main.go`：註冊 root + config + launch 子指令，無 version 子指令
- 無 `.github/workflows/` 目錄

## Goals / Non-Goals

**Goals:**

- 建置時透過 ldflags 注入 SemVer 版本號至 `internal/version` 套件
- 提供 `byok version` 指令顯示當前版本
- GitHub Action 於 main 分支 push 時自動建置 Windows (amd64)、Linux (amd64)、macOS (amd64+arm64) 執行檔
- 自動建立 GitHub Release 並附加執行檔
- 以 git tag 標記版本號

**Non-Goals:**

- 不自動計算版本號遞增（由開發者手動更新 `version.go` 的預設值）
- 不支援 pre-release 版本號格式
- 不做套件管理器發布
- 不做程式碼簽章
- 不做多架構 Docker image

## Decisions

### Decision: ldflags 注入版本號

使用 Go 標準的 `-ldflags "-X <importpath>.Version=<ver>"` 機制，在建置時將版本號字串注入 `internal/version.Version` 變數。未注入時預設值為 `dev`。

**理由**：ldflags 是 Go 社群標準做法，不需額外依賴，建置時才決定版本號符合不變更原始碼的原則。

### Decision: 版本號預設值與手動更新

`internal/version/version.go` 定義 `Version = "dev"` 作為預設值。每次開發合併回 main 前，開發者手動更新此預設值為新版本號（如 `0.1.0` → `0.1.1`），GitHub Action 從此變數讀取版本號。

**理由**：MVP 階段不引入自動版本號計算邏輯（如 git describe、conventional commits），降低複雜度。

### Decision: 版本子指令為獨立 cobra command

新增 `byok version` 子指令，輸出 `byok version <Version>`（如 `byok version 0.1.0`）。

**理由**：與 CLI 工具慣例一致，使用者可快速確認安裝的版本。

### Decision: GitHub Action 使用 matrix 策略建置多平台

使用 GitHub Actions 的 `matrix` 策略，定義 `os × arch` 組合（windows/amd64、linux/amd64、darwin/amd64、darwin/arm64），每個 job 設定 `GOOS`/`GOARCH` 並以 ldflags 注入版本號後 `go build`，產出壓縮執行檔。

**理由**：matrix 策略可平行建置，產出檔命名為 `byok-<version>-<os>-<arch>.zip`（Windows 用 zip，其他用 tar.gz）。

### Decision: 使用 softprops/action-gh-release 建立 Release

workflow 最後一個 job 使用 `softprops/action-gh-release` 第三方 action，附加所有平台執行檔至 Release 並建立 git tag。

**理由**：此 action 為社群常用且維護活躍的 Release 建立 action，簡化 API 呼叫邏輯。

## Implementation Contract

**行為：**
- `byok version` → 印出 `byok version <版本號>`（如 `byok version 0.1.0`）
- 未注入 ldflags 時 → 印出 `byok version dev`
- `make build` → 注入 Makefile 中定義的版本號（預設讀取 `version.go` 的 `Version` 變數）
- push 至 main 分支 → GitHub Action 自動建置 4 個平台執行檔、建立 Release、附加執行檔

**介面：**
- `internal/version/version.go`：`var Version = "dev"`（可被 ldflags 覆寫）
- `cmd/version.go`：`NewVersionCmd()` 回傳 `*cobra.Command`
- `main.go`：`rootCmd.AddCommand(cmd.NewVersionCmd())`
- `.github/workflows/release.yml`：trigger on push to main，matrix build + release job
- `Makefile`：build target 加入 `VERSION ?= $(shell grep -oP '"[^"]*"' internal/version/version.go | tr -d '"')` 與 ldflags

**失敗模式：**
- ldflags 未注入 → `Version` 為 `dev`，`byok version` 正常輸出 `dev`
- GitHub Action 建置失敗 → 該 matrix job 失敗，Release 不建立
- 版本號格式錯誤 → 不驗證格式，由開發者確保 SemVer 合規

**驗收條件：**
- `go build ./...` 通過
- `go test ./...` 全綠（含 version_test.go 驗證 `byok version` 輸出格式）
- `byok version` 輸出 `byok version dev`（未注入）或 `byok version <指定版本>`
- GitHub Action workflow 檔案通過 YAML 語法檢查
- Makefile build 注入版本號後 `./byok version` 輸出對應版本

**範圍邊界：**
- In scope：internal/version、cmd/version、main.go、Makefile、.github/workflows/release.yml
- Out of scope：config、launch、runner 相關檔案；其他子指令

## Risks / Trade-offs

- [版本號需手動更新] 開發者忘記更新 `version.go` → 版本號停留在舊值，Release 標籤錯誤 → 文件提醒每次合併前更新
- [GitHub Action 需寫入權限] 預設 `GITHUB_TOKEN` 需 `contents: write` 權限建立 Release → workflow yaml 明確設定 `permissions`
- [第三方 action 依賴] `softprops/action-gh-release` 為第三方 → 使用固定版本 SHA pin 降低供應鏈風險
