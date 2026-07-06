## Context

`byok` 目前每個 profile 的 `api_key` 以明碼存於 `~/.byok/config.yaml`（`config.Profile.APIKey`），金鑰毫無保護地躺在磁碟。同時裸 `go build .` 因目錄名為 `LlmByok` 產出 `LlmByok.exe`，與發布資產的 `byok` 不一致。本變更同時解決金鑰安全儲存與專案佈局一致性兩個問題。

現狀關鍵點：
- `internal/config/config.go` 的 `Profile.APIKey` 為必要明碼欄位；`internal/runner` 直接讀取 `profile.APIKey` 注入子程序環境變數。
- `main.go` 位於 repo 根；`go.mod` module 為 `github.com/IISI-2209026/LlmByok`。
- Makefile build target 為 `go build ... -o dist/byok .`；release.yml build step 為 `go build ... -o byok .`。

## Goals / Non-Goals

**Goals:**

- 金鑰改存於 OS keychain（Windows Credential Manager / macOS Keychain / Linux secret-service），設定檔不再保存明碼金鑰。
- 保留明碼 `api_key` 作為 fallback，確保 keychain 不可用環境仍可運作並平滑遷移。
- 裸 `go build ./cmd/byok` 預設輸出名為 `byok` 的執行檔，與發布資產一致。

**Non-Goals:**

- 不自建加密層；不自建 keychain backend；不支援 keychain 之外的遠端 vault。
- 不改變環境變數注入機制與子程序啟動方式。
- 不重命名 GitHub repo 或 Go module path。

## Decisions

### 採用 zalando/go-keyring 作為 keychain 抽象層

`zalando/go-keyring` 在 Windows/macOS 為純 Go（CGO-free），Linux 透過 D-Bus 呼叫 libsecret（不需 CGO，但需執行環境存在 secret-service daemon）。相較自行呼叫各平台 syscall，它能以單一 API 覆蓋三大平台且社群維護穩定。替代方案 `99designs/keyring` 引入較多可選 backend（檔案、KWallet、pass）與依賴，超出「OS 原生 keychain」需求，故不採。

### Keychain service 與 key 命名規則

service 固定為 `byok`，key 為 `profile:<profile.Name>`。以 profile 名為 key 後綴可讓多 profile 共存於同一 OS keychain 命名空間且互不衝突，亦便於 `byok config del-key <profile>` 精準刪除。替代方案「一個 key 存整份 JSON」會使更新單一 profile 需重寫全部金鑰且易競態，故不採。

### 金鑰解析順序：keychain 優先、明碼 fallback、皆無報錯

啟動與 `config list` 解析 profile 金鑰時順序為：(1) 查 keychain `profile:<name>`，命中即用；(2) 未命中或 keychain 不可用時，若設定檔 `api_key` 非空則使用明碼並在 `list` 標記 `plaintext`；(3) 兩者皆無則回傳錯誤「找不到 profile <name> 的金鑰（keychain 與設定檔皆無）」並 exit 1。此順序保證遷移期間舊設定檔仍可用，遷移完成後自動改用 keychain。

### set-key 採互動式密碼提示以避免 shell 歷史洩漏

`byok config set-key <profile>` 使用 `golang.org/x/term.ReadPassword` 自 stdin 讀取金鑰且不回顯，讀完即存入 keychain 並清除設定檔中的明碼 `api_key`（若存在）後回寫設定檔。不提供 `--api-key` 旗標以免金鑰進入 shell 歷史與 process listing。替代方案「從檔案讀」需使用者自行建立暫存檔且忘記刪除風險更高，故不採。

### import-keys 批次遷移並產生 zero 明碼設定檔

`byok config import-keys` 遍歷設定檔中所有 `api_key` 非空的 profile，逐一存入 keychain 並將該 profile 的 `api_key` 清空後回寫設定檔一次。單一 profile 失敗時印出該 profile 名稱與錯誤後繼續其餘 profile，最終若有任一失敗則 exit 1 並列出失敗清單；全部成功則印出匯入數量。

### 專案佈局：main package 移至 cmd/byok/

將 `main.go` 移至 `cmd/byok/main.go`（`git mv`），使 Go 以目錄名 `byok` 為預設輸出檔名。`go build ./cmd/byok` → `byok`；`go install github.com/IISI-2209026/LlmByok/cmd/byok@latest` → 安裝 `byok`。根目錄不再有 `main.go`。替代方案「根目錄加 `-o byok` 到 go.mod」不可行（go.mod 無 build 設定）；「rename repo 目錄」為環境特定不可控，故不採。

### 新增 internal/secret 套件封裝 keychain 操作

新增 `internal/secret` 套件定義 `Store(profileName, apiKey string) error`、`Load(profileName string) (string, error)`、`Delete(profileName string) error` 與 `Exists(profileName string) (bool, error)`，內部以 `zalando/go-keyring` 實作。`internal/config` 不直接呼叫 go-keyring，而是透過注入的 `KeyResolver` 介面（`Resolve(profile Profile) (string, Source, error)`）取得金鑰，便於測試以 fake resolver 替換真實 keychain。

## Implementation Contract

**Behavior（終端使用者可觀察）：**

- `byok config set-key <profile>`：提示「輸入金鑰:」，讀取後印「已將金鑰存入 keychain（profile: <name>）」並（若原設定檔有明碼）印「已清除設定檔中的明碼 api_key」。
- `byok config del-key <profile>`：印「已自 keychain 刪除金鑰（profile: <name>）」；keychain 中不存在時印「profile <name> 未在 keychain 中」並 exit 1。
- `byok config import-keys`：印「匯入 N 個金鑰至 keychain」或「匯入失敗: <清單>」並 exit 1。
- `byok config list`：每行 profile 顯示金鑰來源欄位，值為 `keychain` / `plaintext` / `missing`。
- `byok launch <target>`：解析金鑰成功即照原機制注入；解析失敗印「找不到 profile <name> 的金鑰（keychain 與設定檔皆無）」並 exit 1。

**Interface / data shape：**

- `internal/secret`：`Store(profileName, apiKey string) error`、`Load(profileName string) (string, error)`、`Delete(profileName string) error`、`Exists(profileName string) (bool, error)`；service 常數 `byok`，key 前綴 `profile:`。
- `internal/config`：`Profile.APIKey` 保留但改為可選（YAML omitempty）；新增 `KeyResolver` 介面 `Resolve(Profile) (apiKey string, source Source, err error)`，`Source` 為列舉 `SourceKeychain`/`SourcePlaintext`/`SourceMissing`；提供 `DefaultResolver` 以 `internal/secret` 實作。
- `cmd/config.go`：新增 `setKeyCmd`、`delKeyCmd`、`importKeysCmd` 三個 cobra.Command 並於 `configCmd.AddCommand` 註冊；`listCmd` 輸出新增 `Source` 欄。
- `cmd/byok/main.go`：自根 `main.go` `git mv` 而來，內容不變（package main、呼叫 `cmd.NewRoot`）。
- `Makefile` build target：`go build -ldflags "$(LDFLAGS)" -o dist/byok ./cmd/byok`；run target：`go run ./cmd/byok $(ARGS)`。
- `.github/workflows/release.yml` build step：`go build -ldflags "..." -o byok ./cmd/byok`。
- `go.mod` 新增 `github.com/zalando/go-keyring` 與 `golang.org/x/term`。

**Failure modes：**

- keychain 不可用（headless Linux 無 secret-service）：`secret.Load` 回傳錯誤 → `DefaultResolver` fallback 到明碼 `api_key`；若明碼亦空 → 回傳 `SourceMissing` 錯誤。
- `set-key` 讀取到空字串：印「金鑰不可為空」並 exit 1，不寫入 keychain。
- `import-keys` 單一 profile 失敗：記錄並繼續，最終彙整 exit 1。
- profile 不存在（set-key/del-key）：印「profile <name> 不存在」並 exit 1。

**Acceptance criteria：**

- `go build ./cmd/byok` 產出檔名為 `byok`（Windows 為 `byok.exe`）。
- `go test ./internal/secret/...` 通過（含 fake 覆蓋 Store/Load/Delete/Exists 與錯誤路徑）。
- `go test ./internal/config/...` 通過（含 KeyResolver fake 覆蓋三種 Source 與金鑰解析順序）。
- `go test ./cmd/...` 通過（含 set-key/del-key/import-keys/list 來源標記測試）。
- `go vet ./...` 無警告。
- 手動：`byok config set-key <profile>` 後 `~/.byok/config.yaml` 不再含明碼 `api_key`，`byok launch <target>` 仍可正常啟動。

**Scope boundaries：**

- In scope：`internal/secret` 新套件、`internal/config` 金鑰解析、`cmd/config.go` 三新子指令與 list 標記、`cmd/byok/main.go` 佈局、Makefile/release.yml/README/AGENTS/go.mod 同步。
- Out of scope：環境變數注入邏輯、self-update 機制、release changelog、copilot/codex 啟動參數。

## Risks / Trade-offs

- [Linux headless 無 secret-service] → fallback 到明碼 `api_key`；`list` 標記 `plaintext` 提醒使用者。文件需說明 Linux 需安裝 secret-service 或 gnome-keyring。
- [`zalando/go-keyring` 新增第三方依賴] → 該套件穩定且為 Go 生態常用 keychain 抽象；接受依賴以換取跨平台一致性。
- [`cmd/byok` 佈局改變影響現有開發者 muscle memory] → README 與 AGENTS.md 同步更新；`go run main.go` 改為 `go run ./cmd/byok`。
- [`x/term` 在無 TTY 環境（CI）讀取失敗] → `set-key` 僅供互動使用，CI 不需執行；測試以 fake stdin 覆蓋。
