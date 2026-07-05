## 1. 專案初始化與模組結構

- [x] 1.1 初始化 Go module manifest：執行 `go mod init github.com/IISI-2209026/LlmByok` 產生 `go.mod`，宣告 Go 版本。驗證：`go.mod` 存在且 module path 正確。對應 spec「Go module manifest」。
- [x] 1.2 新增 Cobra 與 yaml.v3 依賴：執行 `go get github.com/spf13/cobra@latest` 與 `go get gopkg.in/yaml.v3@latest`，更新 `go.mod` 與 `go.sum`。驗證：`go mod tidy` 無錯誤且 `go build` 成功。對應 design 決策「使用 Go 語言與 Cobra CLI 框架」。
- [x] 1.3 建立專案目錄結構：建立 `cmd/`、`internal/config/`、`internal/runner/` 目錄。驗證：目錄存在且後續檔案可放置。

## 2. 設定檔資料模型與讀寫

- [x] 2.1 實作 `internal/config/config.go` 中的設定檔資料結構：定義 `Profile` struct（欄位 `Name`、`Provider`、`APIBase`、`APIKey`、`DefaultModel`，yaml tag 對應 `name`、`provider`、`api_base`、`api_key`、`default_model`）與 `Config` struct（欄位 `Profiles []Profile`、`DefaultProfile string`）。驗證：單元測試 `TestConfigStructYAMLTags` 以 YAML 字串 unmarshal 後欄位值正確。對應 design 決策「設定檔格式為 YAML 並存放於 ~/.byok/config.yaml」。
- [x] 2.2 實作 `Load(path string)` 函式：讀取指定路徑 YAML 檔並 unmarshal 為 `Config`。當檔案不存在回傳明確錯誤；當 YAML 解析失敗回傳包含檔案路徑與解析錯誤的訊息。驗證：單元測試 `TestLoad_MissingFile` 於不存在路徑回傳錯誤；`TestLoad_MalformedYAML` 於非法 YAML 回傳含路徑的錯誤。對應 spec「Config file parse error reporting」與「Config file location」。
- [x] 2.3 實作 `Save(path string, cfg *Config)` 函式：將 `Config` marshal 為 YAML 並寫入指定路徑（檔案權限 0600，目錄不存在則建立）。驗證：單元測試 `TestSave_CreatesFile` 寫入後以 `Load` 讀回內容一致。
- [x] 2.4 實作預設設定檔路徑解析函式 `DefaultConfigPath()`：回傳 `~/.byok/config.yaml`（使用 `os.UserHomeDir`）。驗證：單元測試 `TestDefaultConfigPath` 回傳路徑以 `.byok/config.yaml` 結尾。對應 spec「Config file location」。

## 3. 設定檔管理子指令

- [x] 3.1 實作 `cmd/config.go` 中的 `config add` 子指令：接受 `--name`、`--provider`、`--api-base`、`--api-key`、`--default-model`、`--config` 旗標，將新 profile 附加至設定檔；檔案不存在則建立；`default_profile` 未設定時將新 profile 設為預設；同名 profile 已存在時印出錯誤並 exit code 1，不修改檔案。驗證：`TestConfigAdd_NewProfileCreatesFile` 與 `TestConfigAdd_DuplicateNameErrors`。對應 spec「Add a profile」與「Set default profile」。
- [x] 3.2 實作 `config list` 子命令：列出所有 profile 的 `name`、`provider`、`api_base`、`default_model` 與遮罩後的 `api_key`。遮罩規則：顯示前 4 與後 4 字元以 `...` 連接，空字串顯示空字串。驗證：單元測試 `TestMaskAPIKey` 涵蓋長金鑰、短金鑰、空字串三種邊界；`TestConfigList_Output` 檢查輸出含 profile 名稱。對應 spec「List profiles with masked API key」與「Config file parse error reporting」。
- [x] 3.3 實作 `config remove` 子命令：以 `--name` 移除指定 profile；不存在時印出錯誤並 exit code 1 不修改檔案；被移除 profile 為 `default_profile` 時清空該欄位。驗證：`TestConfigRemove_Existing` 與 `TestConfigRemove_NotFound`。對應 spec「Remove a profile」。
- [x] 3.4 實作 `config set-default` 子命令：以 `--name` 更新 `default_profile`；profile 不存在時 exit code 1。驗證：`TestSetDefault_Existing` 與 `TestSetDefault_NotFound`。對應 spec「Set default profile」。

## 4. Launch 子命令與子行程環境變數注入

- [x] 4.1 實作 `internal/runner/runner.go` 中的 `BuildEnv(profile *Profile, modelOverride string) []string` 函式：以 `os.Environ()` 為基礎，覆寫 `COPILOT_PROVIDER_BASE_URL`、`COPILOT_PROVIDER_TYPE`（取 profile.Provider，預設 `openai`）、`COPILOT_PROVIDER_API_KEY`、`COPILOT_MODEL`（modelOverride 非空則用之，否則用 profile.DefaultModel）。驗證：單元測試 `TestBuildEnv_OverridesByokVars` 確認四個變數值正確，且其他既有環境變數保留。對應 design 決策「透過子行程環境變數注入 BYOK 設定」與「--model 旗標覆寫預設模型」與 spec「Launch Copilot with BYOK profile」。
- [x] 4.2 實作 `cmd/launch.go` 中的 `launch copilot` 子命令：解析 `--model`、`--profile`、`--config` 旗標；`--config` 覆寫預設設定檔路徑，否則使用 `~/.byok/config.yaml`；以 `--config` 或預設路徑載入設定檔；以 `--profile` 或 `default_profile` 選取 profile；執行 provider 驗證（僅接受 `openai`，否則 exit 1）。驗證：`TestLaunch_NonOpenaiProviderRejected` 與 `TestLaunch_CustomConfigPath`。對應 spec「Provider validation」與「Config file path override」與 design 決策「預設 Provider 為 openai 且僅支援 OpenAI 相容格式」。
- [x] 4.3 實作缺失設定檔與缺失 profile 錯誤處理：設定檔不存在時印出建議執行 `byok config add` 並 exit 1；指定 profile 不存在時列出可用 profile 名稱並 exit 1。驗證：`TestLaunch_MissingConfigFile` 與 `TestLaunch_MissingProfile`。對應 spec「Missing config file error」與「Missing profile error」。
- [x] 4.4 實作 `copilot` 執行檔存在檢查與子行程啟動：以 `exec.LookPath("copilot")` 檢查，找不到時印出安裝提示並 exit 1；找到時以 `exec.Command` 啟動，`cmd.Env` 設為 `BuildEnv` 結果，`cmd.Stdin/Stdout/Stderr` 串接 `os.Stdin/os.Stdout/os.Stderr`，呼叫 `Run`。驗證：`TestLaunch_CopilotNotInstalled`（模擬 PATH 中無 copilot）與整合測試以 stub 執行檔驗證環境變數正確注入。對應 spec「Copilot executable presence check」與「Launch Copilot with BYOK profile」。
- [x] 4.5 驗證父行程環境不受影響：在啟動子行程前後比對 `os.Environ()` 內容不變。驗證：整合測試 `TestLaunch_ParentEnvUnchanged` 於 stub copilot 結束後確認父行程環境變數集合相同。對應 spec「Parent process environment unchanged」。

## 5. 根命令與 CLI 入口

- [x] 5.1 實作 `main.go` 與根命令：建立 root `cobra.Command`（`byok`），註冊 `launch` 與 `config` 父子命令，設定使用說明與版本資訊。驗證：執行 `go run main.go --help` 顯示 `launch` 與 `config` 子命令列表；`go run main.go config --help` 顯示 add/list/remove/set-default。對應 design 決策「使用 Go 語言與 Cobra CLI 框架」。

## 6. 建置腳本與 README

- [x] 6.1 新增 `Makefile`，提供 `build`（`go build -o dist/byok .`）、`run`（`go run main.go`，可傳args）、`clean`（移除 `dist/`）目標。驗證：`make build` 產生 `dist/byok`（Windows 為 `dist/byok.exe`）且 `make clean` 後該檔案不存在。對應 spec「Build via Makefile」。
- [x] 6.2 撰寫 `README.md`：面向未寫過 Go 的開發者。開頭以工具概覽章節說明 `byok` 的功能（命令列工具，暫時注入 BYOK 環境變數以啟動 Copilot CLI 並使用自己的 OpenAI 相容 API 金鑰，不修改系統環境）、解決的問題、主要特性（profile 管理、單指令啟動、暫時性環境注入不影響日常使用）。其後依序涵蓋前置需求、各平台 Go 安裝（Windows/macOS/Linux）、`go build`/`go install` 編譯、`go run main.go` 執行、建立設定檔範例（含兩個 profile：遠端 OpenAI 相容端點含 api_key、本機無驗證端點）。使用說明章節涵蓋每個指令（`launch copilot`、`config add`、`config list`、`config remove`、`config set-default`）的旗標、白話說明與至少一個具體範例指令。驗證：內容審查確認涵蓋工具概覽、Go 環境建置、所有指令的使用說明與範例，且含可複製的 YAML 範例與指令範例。對應 spec「README.md with tool overview and Go environment setup guide」與「Example config in README」與 design 決策「README.md 面向 Go 新手」。

## 7. 端對端驗證

- [x] 7.1 撰寫整合測試 `internal/runner/launch_integration_test.go`：以 stub `copilot` 腳本（印出環境變數後結束）驗證 `byok launch copilot --model glm-5.2` 啟動後子行程環境中 `COPILOT_MODEL=glm-5.2`、`COPILOT_PROVIDER_BASE_URL` 等於 profile api_base、父行程環境不變。驗證：`TestLaunchIntegration_ByokVarsInjected` 通過。對應 spec「Launch Copilot with BYOK profile」、「Parent process environment unchanged」。
- [x] 7.2 執行完整測試套件：`go test ./...` 全部通過。驗證：無失敗測試，coverage 報告涵蓋 config 與 runner 套件。
