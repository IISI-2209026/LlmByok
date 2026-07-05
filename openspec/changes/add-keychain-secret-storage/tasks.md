## 1. 依賴與專案佈局

- [x] [P] 1.1 新增 `github.com/zalando/go-keyring` 與 `golang.org/x/term` 至 go.mod 並執行 `go mod tidy`，使專案可取得 keychain 與終端密碼讀取相依。驗證：`go build ./...` 成功且 go.sum 已更新。
- [x] [P] 1.2 專案佈局：main package 移至 cmd/byok/ — 以 `git mv main.go cmd/byok/main.go` 移動入口，內容不變（package main、呼叫 `cmd.NewRoot`），使裸 `go build ./cmd/byok` 預設輸出 `byok`（Windows 為 `byok.exe`）。驗證：`go build ./cmd/byok` 產出檔名為 `byok`，且 `go run ./cmd/byok config list` 可正常執行。

## 2. internal/secret 套件

- [ ] 2.1 新增 internal/secret 套件封裝 keychain 操作 — 採用 zalando/go-keyring 作為 keychain 抽象層，定義 service 常數 `byok` 與 Keychain service 與 key 命名規則 `profile:<profileName>`，實作 `Store(profileName, apiKey string) error`（覆寫已存在值）、`Load(profileName string) (string, error)`、`Delete(profileName string) error`、`Exists(profileName string) (bool, error)`，並將 go-keyring 錯誤分類為「not-found」與「backend-unavailable」兩種可區分錯誤。驗證：`go vet ./internal/secret/...` 無警告且套件可編譯。
- [ ] 2.2 撰寫 internal/secret 測試，覆蓋 OS keychain secret storage abstraction 之 Store→Load 往返、覆寫後 Load 回新值、Delete 後 Exists 為 false、Load 不存在 key 回 not-found 錯誤、backend 不可用回 backend-unavailable 錯誤（以 fake/wrap 驗證錯誤分類）。驗證：`go test ./internal/secret/...` 通過。

## 3. internal/config 金鑰解析

- [ ] 3.1 修改 internal/config：`Profile.APIKey` 改為 yaml omitempty（可選），新增 `Source` 列舉（`SourceKeychain`/`SourcePlaintext`/`SourceMissing`）與 `KeyResolver` 介面 `Resolve(Profile) (apiKey string, source Source, err error)`，並提供 `DefaultResolver` 以 internal/secret 實作金鑰解析順序：keychain 優先、明碼 fallback、皆無報錯（keychain 命中回 SourceKeychain；失敗且 APIKey 非空回 SourcePlaintext；兩者皆無回 SourceMissing 與含 profile 名稱的錯誤）。驗證：`go build ./internal/config/...` 成功。
- [ ] 3.2 撰寫 internal/config 測試，以 fake KeyResolver 覆蓋 API key resolution order 之四情境：keychain hit（SourceKeychain）、plaintext fallback（SourcePlaintext）、both empty（SourceMissing + 錯誤含 profile 名）、backend-unavailable 且 plaintext 存在（SourcePlaintext）。驗證：`go test ./internal/config/... -run Resolve` 通過。

## 4. cmd/config 子指令

- [ ] [P] 4.1 實作 Set API key in OS keychain — `byok config set-key <profile>`：set-key 採互動式密碼提示以避免 shell 歷史洩漏（golang.org/x/term.ReadPassword 不回顯讀取），存入 keychain 後清除設定檔明碼 api_key 並回寫；空金鑰印「金鑰不可為空」exit 1；profile 不存在印「profile <name> 不存在」exit 1。驗證：`go test ./cmd/... -run SetKey` 通過（含空金鑰與 profile 不存在路徑）。
- [ ] [P] 4.2 實作 Delete API key from OS keychain — `byok config del-key <profile>`：自 keychain 刪除 `profile:<name>`；keychain 無該項印「profile <name> 未在 keychain 中」exit 1；profile 不存在印「profile <name> 不存在」exit 1。驗證：`go test ./cmd/... -run DelKey` 通過。
- [ ] [P] 4.3 實作 Batch import plaintext keys into keychain — `byok config import-keys`：import-keys 批次遷移並產生 zero 明碼設定檔 — 遍歷 api_key 非空 profile 存入 keychain 後清除欄位並回寫設定檔一次；單一失敗記錄並繼續，最終印失敗清單 exit 1；全部成功印「匯入 N 個金鑰至 keychain」；無明碼金鑰印「設定檔中無明碼金鑰可匯入」exit 0 不回寫。驗證：`go test ./cmd/... -run ImportKeys` 通過（含全部成功、部分失敗、無可匯入三情境）。
- [ ] [P] 4.4 修改 `byok config list`：List profiles with key source indicator — 透過 KeyResolver 解析每個 profile 金鑰來源並顯示 `keychain`/`plaintext`/`missing` 欄位，遮罩金鑰以解析後的值計算。驗證：`go test ./cmd/... -run List` 通過（含混合來源情境）。
- [ ] 4.5 於 cmd/root.go（或 config.go）將 set-key、del-key、import-keys 三個 cobra.Command 註冊至 configCmd。驗證：`go run ./cmd/byok config --help` 列出 set-key、del-key、import-keys 三個子指令。
- [ ] 4.6 修改 `byok config add`：Add a profile 的 `api_key` 改為可選，省略 `--api-key` 時建立空 api_key profile 並成功退出。驗證：`go test ./cmd/... -run Add` 通過（含不帶 --api-key 情境）。

## 5. build/CI/文件同步

- [ ] [P] 5.1 更新建置入口（Build via Makefile 與 release workflow）：Makefile build target 改 `go build -ldflags "$(LDFLAGS)" -o dist/byok ./cmd/byok`、run target 改 `go run ./cmd/byok $(ARGS)`；.github/workflows/release.yml build step 改 `go build -ldflags "..." -o byok ./cmd/byok`，使 Multi-platform build via GitHub Actions matrix 仍產出 byok 二進位。驗證：`make build` 產出 `dist/byok`，且 release.yml build step 指向 `./cmd/byok`（內容審閱）。
- [ ] [P] 5.2 更新 README.md：安裝指令改 `go install github.com/IISI-2209026/LlmByok/cmd/byok@latest`、建置改 `go build ./cmd/byok`、執行改 `go run ./cmd/byok`，新增金鑰管理區塊說明 set-key/del-key/import-keys 與 Linux 需 secret-service daemon 及明碼 fallback 注意事項，並補齊 README.md with tool overview and Go environment setup guide 所列所有指令說明。驗證：內容審閱確認安裝指令、金鑰管理區塊與 Linux 注意事項齊備。
- [ ] [P] 5.3 更新 AGENTS.md：套件職責表新增 `internal/secret` 套件、main 位置改 `cmd/byok`、internal/config 職責補充 keychain 金鑰解析；開發規範補充「金鑰以 OS keychain 為主要儲存、明碼 api_key 為 fallback」。驗證：內容審閱確認套件表與開發規範已同步。

## 6. 驗證

- [ ] 6.1 執行 `go build ./cmd/byok` 確認產出檔名為 byok；`go vet ./...` 無警告；`go test ./...`（Windows 不加 -race）全部通過。驗證：三項指令皆 exit 0。
- [ ] 6.2 手動端對端：`byok config set-key <profile>` 後確認 `~/.byok/config.yaml` 不再含明碼 api_key，且 `byok launch <target>` 仍可正常啟動（金鑰來自 keychain）。驗證：手動斷言設定檔無明碼且 launch 成功。
