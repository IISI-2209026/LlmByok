## 1. 版本號套件與子指令

- [x] 1.1 建立 `internal/version/version.go`，定義 `var Version = "dev"` 可被 ldflags 覆寫；建立 `internal/version/version_test.go` 驗證預設值為 `dev`（對應 Decision: ldflags 注入版本號、Decision: 版本號預設值與手動更新、規格「Version variable injection via ldflags」）。行為：未注入 ldflags 時 `Version` 為 `dev`。驗證：`go test ./internal/version/` 全綠。
- [x] 1.2 建立 `cmd/version.go` 的 `NewVersionCmd()` 回傳 `*cobra.Command`，執行時印出 `byok version <Version>`；建立 `cmd/version_test.go` 驗證輸出格式（對應 Decision: 版本子指令為獨立 cobra command、規格「Version subcommand」）。行為：`byok version` 印出 `byok version <當前 Version>`。驗證：`go test ./cmd/` 全綠。
- [x] 1.3 在 `main.go` 註冊 version 子指令（`rootCmd.AddCommand(cmd.NewVersionCmd())`）；更新 `Makefile` build target 加入 ldflags 注入版本號（`-ldflags "-X github.com/IISI-2209026/LlmByok/internal/version.Version=<ver>"`）。行為：`make build` 產出的 `./byok` 執行 `version` 子指令時輸出注入的版本號而非 `dev`。驗證：`go build ./...` 通過、`make build && ./byok version` 輸出非 `dev` 的版本號。

## 2. GitHub Actions Release Workflow

- [x] 2.1 建立 `.github/workflows/release.yml`，trigger on push to `main`，使用 matrix 策略建置 `windows/amd64`、`linux/amd64`、`darwin/amd64`、`darwin/arm64` 四個平台的 `byok` 執行檔（對應 Decision: GitHub Action 使用 matrix 策略建置多平台、規格「Multi-platform build via GitHub Actions matrix」）。每個 job 設定 `GOOS`/`GOARCH`、注入 ldflags 版本號、產出 `byok-<version>-<os>-<arch>.<ext>` 壓縮檔（Windows 用 zip、其他用 tar.gz）。行為：push 至 main 時四個平台執行檔自動建置並壓縮。驗證：YAML 語法檢查通過、matrix 包含四個目標組合。
- [x] 2.2 在 release workflow 新增 release job，於所有 matrix build jobs 完成後使用 `softprops/action-gh-release`（固定版本 SHA pin）建立 GitHub Release 並附加所有平台壓縮檔、以版本號為 git tag，workflow 設定 `permissions: contents: write`（對應 Decision: 使用 softprops/action-gh-release 建立 Release、規格「GitHub Release creation on main branch push」）。行為：Release 建立後附有四個平台壓縮檔、tag 為當前版本號。驗證：YAML 中 release job 依賴 build job、`permissions` 區塊含 `contents: write`、action 使用 SHA pin。

## 3. 文件更新

- [x] 3.1 更新 `README.md` 補充版本號管理說明（SemVer 規範、每次合併前更新 `internal/version/version.go`）與 `byok version` 指令用法、GitHub Actions 自動發布流程說明（對應 Decision: 版本號預設值與手動更新）。行為：README 涵蓋版本指令用法、版本號更新流程、自動發布說明。驗證：人工審閱 README 含 `byok version` 範例、版本更新步驟、release workflow 說明。
