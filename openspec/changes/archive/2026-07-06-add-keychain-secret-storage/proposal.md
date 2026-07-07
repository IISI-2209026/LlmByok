## Why

`byok` 目前將每個 profile 的 `api_key` 以明碼存於 `~/.byok/config.yaml`，金鑰暴露在磁碟上無任何保護，風險過高。應將金鑰改存於各 OS 專屬的密碼管理軟體（Windows Credential Manager、macOS Keychain、Linux secret-service/libsecret），設定檔不再保存明碼金鑰。同時，目前裸 `go build .` 產出的執行檔名為 `LlmByok`（取自目錄名），與發布資產內的 `byok` 不一致，造成使用混淆；應讓裸 `go build` 預設輸出 `byok`。

## What Changes

- 新增 `internal/secret` 套件：以 `zalando/go-keyring` 封裝跨平台金鑰存取（service 名 `byok`，key 為 `profile:<name>`），提供 `Store`/`Load`/`Delete` 操作。
- 修改 `internal/config`：profile 的 `api_key` 改為可選；啟動時解析金鑰的順序為「先查 keychain，找不到再 fallback 到設定檔明碼 `api_key`，兩者皆無則報錯」。
- 新增 `byok config set-key <profile>` 子指令：以互動提示讀取金鑰並存入 keychain，同時清除設定檔中的明碼 `api_key`。
- 新增 `byok config del-key <profile>` 子指令：自 keychain 刪除該 profile 金鑰。
- 新增 `byok config import-keys` 子指令：將設定檔中所有含明碼 `api_key` 的 profile 匯入 keychain 並清除明碼，產生 zero 明碼設定檔。
- 修改 `byok config list`：顯示金鑰來源標記（`keychain` 或 `plaintext` 或 `missing`）。
- 重構專案佈局：將 `main.go` 移至 `cmd/byok/main.go`，使裸 `go build ./cmd/byok` 與 `go install github.com/IISI-2209026/LlmByok/cmd/byok@latest` 預設產出/安裝名為 `byok` 的執行檔；同步更新 Makefile、release workflow build 步驟、README 安裝指令與 AGENTS.md 套件職責。

## Non-Goals

- 不實作 keychain 加密層（交由各 OS 原生機制）。
- 不支援非 `zalando/go-keyring` 覆蓋的 backend（如 KDE KWallet 直接介接）；headless Linux 無 secret-service 時 fallback 到明碼 `api_key`。
- 不改變 codex/copilot 啟動時的環境變數注入機制，僅改變金鑰「來源解析」。
- 不為 keychain 操作加入 `--config` 覆寫（keychain 為 OS 全域，不隨設定檔路徑變動）。
- 不重命名 GitHub repo 或 Go module path（僅移動 main package 位置）。

## Capabilities

### New Capabilities

- `byok-secret-storage`: 跨平台 OS keychain 金鑰儲存與解析能力——封裝 `zalando/go-keyring` 提供 profile 金鑰的 Store/Load/Delete，並定義金鑰解析順序（keychain 優先、明碼 fallback、皆無報錯）。

### Modified Capabilities

- `byok-config`: 新增 `set-key`/`del-key`/`import-keys` 子指令；`list` 顯示金鑰來源；profile `api_key` 改為可選並以 keychain 為主要來源。
- `byok-setup`: main package 移至 `cmd/byok/`；README 安裝指令改為 `go install github.com/IISI-2209026/LlmByok/cmd/byok@latest`；裸 `go build ./cmd/byok` 輸出 `byok`。
- `byok-release`: release workflow build 步驟改為 `go build ./cmd/byok`，產出二進位仍命名 `byok`。

## Impact

- Affected specs: `byok-secret-storage`（new）、`byok-config`（modified）、`byok-setup`（modified）、`byok-release`（modified）
- Affected code:
  - New: `internal/secret/secret.go`, `internal/secret/secret_test.go`, `cmd/byok/main.go`（自 `main.go` 移入）
  - Modified: `internal/config/config.go`（金鑰解析順序與 `APIKey` 可選）、`internal/config/config_test.go`、`cmd/config.go`（新增 set-key/del-key/import-keys、list 來源標記）、`cmd/config_test.go`、`cmd/root.go`（註冊新子指令）、`cmd/launch.go`（金鑰解析改走新順序）、`Makefile`（build target 改 `./cmd/byok`）、`.github/workflows/release.yml`（build step 改 `./cmd/byok`）、`README.md`（安裝指令與金鑰管理說明）、`AGENTS.md`（套件職責與 main 位置）、`go.mod`（新增 `zalando/go-keyring` 相依）
  - Removed: `main.go`（內容移至 `cmd/byok/main.go`）
