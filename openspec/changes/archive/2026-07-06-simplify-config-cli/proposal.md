## Why

目前 `byok config` 暴露過多獨立子指令（`set-key`、`del-key`、`import-keys`、`add`、`remove`、`set-default`、`list`），使用者需先 `add` 再 `set-key` 才能完成一個可用的 profile，操作鏈冗長且不直覺。金鑰儲存位置（明碼或 keychain）也散落在不同指令，造成認知負擔與誤用風險。現在需要整合成一條以 profile 為中心的操作流程。

## What Changes

- **BREAKING**：移除 `byok config set-key`、`byok config del-key`、`byok config import-keys` 三個獨立金鑰子指令。`set-default` 與 `list` 保留不變。
- **BREAKING**：將 `byok config remove` 重新命名為 `byok config delete`，並在刪除 profile 時同步刪除 keychain 中 `profile:<name>` 的條目（即使 keychain 查詢失敗仍完成 profile 刪除並印出警告）。
- 新增 `byok config update --name <name>` 子指令，可更新既有 profile 的 `provider`、`api_base`、`default_model` 與金鑰；未提供的欄位保持原值。
- `byok config add` 與 `byok config update` 皆同時支援「終端互動式」與「純參數」兩種模式：
  - 互動模式：未傳任何 `--name`/`--api-key` 等旗標時，依序提示輸入 name、provider、api_base、default_model、api_key。
  - 參數模式：透過 `--name`、`--provider`、`--api-base`、`--default-model`、`--api-key` 直接提供，適合腳本使用。
- `add`/`update` 在取得 api_key 後，透過 `--key-storage plaintext|keychain` 旗標（互動模式則提示選擇）決定儲存位置，預設為 `keychain`。選擇 `keychain` 時呼叫 `internal/secret.Store` 並將 config 中該 profile 的 `api_key` 欄位清空；選擇 `plaintext` 時將金鑰寫入 config 的 `api_key` 欄位並刪除 keychain 既有條目（如有）。
- keychain backend 不可用時（如無 secret-service 的 Linux），`--key-storage keychain` 須印出明確錯誤並 exit 1，引導使用者改用 `--key-storage plaintext`。
- 互動模式在非 TTY 環境下（stdin 非 terminal）須印出錯誤並 exit 1，提示改用參數模式。

## Non-Goals

- 不改變 `internal/secret` 的 `Store`/`Load`/`Delete`/`Exists` 介面與 service/key 命名規則。
- 不改變 `internal/config.KeyResolver` 的解析順序（keychain 優先 → 明碼 fallback）。
- 不改變設定檔路徑、格式或 `byok config list` 的輸出欄位。
- 不為現存明碼 profile 提供一次性遷移指令；使用者可透過 `update --key-storage keychain` 自行轉移。
- 不引入第三方互動式 prompt 函式庫；以 `golang.org/x/term` 與標準 `bufio` 實作提示。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `byok-config`: 移除 `set-key`/`del-key`/`import-keys` 需求；新增 `update` 子命令需求；修改 `add` 需求以涵蓋互動模式與金鑰儲存選擇；將 `remove` 需求改為 `delete` 並合併 keychain 同步刪除。

## Impact

- Affected specs:
  - Modified: `byok-config`（`byok-secret-storage` 介面不變，無 spec 層級變動）
- Affected code:
  - Modified: `cmd/config.go`（重構 add/remove 子命令、新增 update、移除 set-key/del-key/import-keys）、`internal/config/config.go`（新增 profile 欄位更新輔助函式）、`cmd/config_test.go`（改寫對應測試）、`internal/config/config_test.go`
  - New: `internal/config/interactive.go`（終端互動提示與 TTY 偵測共用邏輯）、`cmd/config_interactive_test.go`
  - Removed: `set-key`/`del-key`/`import-keys` 相關命令實作與測試片段
