## Context

`byok config` 目前以「子指令粒度」組織：`add`、`remove`、`set-default`、`list`、`set-key`、`del-key`、`import-keys`。金鑰管理是 profile 之外的獨立動作，使用者必須先 `add`（無金鑰）再 `set-key`，且 `add` 預設將金鑰寫成明碼於 config，與「keychain 為主要儲存」的開發規範不一致。本次變更將金鑰操作折疊回 profile 生命週期（add/update/delete），並引入互動式與金鑰儲存位置選擇。

`internal/secret`（`Store`/`Load`/`Delete`/`Exists`）與 `internal/config.KeyResolver`（keychain → 明碼 fallback）介面與解析順序維持不變；本次只調整 `cmd/config.go` 的命令結構與新增 `internal/config/interactive.go`。

## Goals / Non-Goals

**Goals:**

- 把金鑰管理整合進 profile 的 add/update/delete，消除獨立 set-key/del-key/import-keys。
- `add`/`update` 同時支援終端互動與純參數模式，且明確選擇金鑰儲存位置（預設 keychain）。
- `delete` 在移除 profile 時同步清理 keychain，保持儲存層與 config 一致。
- 維持參數模式可用於腳本/CI，互動模式僅在 TTY 啟用並在非 TTY 明確失敗。

**Non-Goals:**

- 不改 `internal/secret` 介面、service/key 命名，也不改 `KeyResolver` 解析順序。
- 不提供一次性批次遷移指令；遷移以 `update` 逐筆進行。
- 不引入第三方 prompt 函式庫；以 `golang.org/x/term` 與 `bufio` 實作。
- 不改變 `list`/`set-default` 行為與設定檔路徑/格式。

## Decisions

### Decision: 以 profile 為中心折疊金鑰操作

將 `set-key`/`del-key`/`import-keys` 移除，金鑰由 `add`/`update` 在建立或更新 profile 時一併處理，`delete` 在移除 profile 時同步清理 keychain。理由：使用者心智模型是「管理 profile」而非「先建 profile 再管金鑰」；獨立金鑰指令造成冗長操作鏈與 keychain/config 不一致風險。替代方案（保留獨立指令但加互動）被否決，因為仍存在兩條流程且容易產生金鑰懸置於 keychain 的孤兒條目。

### Decision: 預設金鑰儲存為 keychain，明確旗標切換明碼

`add`/`update` 提供 `--key-storage plaintext|keychain`，預設 `keychain`。選 `keychain` 時呼叫 `internal/secret.Store` 並清空 config 的 `api_key`；選 `plaintext` 時寫入 config 並刪除既有 keychain 條目。理由：對齊「金鑰以 keychain 為主要儲存」規範，同時保留明碼 fallback 給無 secret-service 的環境。替代方案（一律明碼、或一律強制 keychain）被否決：前者違反安全規範，後者在無 daemon 環境無可用路徑。

### Decision: 互動模式以「無任何 profile 欄位旗標」觸發

當 `add`/`update` 未傳 `--name`、`--provider`、`--api-base`、`--default-model`、`--api-key` 任一旗標時進入互動模式，依序提示各欄位與金鑰儲存選擇；`update` 仍需 `--name` 指定目標 profile，故 `update` 的互動模式定義為「提供 `--name` 但未提供其他欄位旗標」。理由：避免 `--name` 與互動提示語意衝突，並讓腳本可純參數使用。替代方案（新增 `--interactive` 旗標）被否決，因為「未傳欄位旗標即互動」更直覺且減少旗標數量。

### Decision: 非 TTY 時互動模式明確失敗

互動模式偵測 stdin 是否為 terminal（`term.IsTerminal`），非 TTY 時印出錯誤並 exit 1，提示改用參數模式。理由：避免管線輸入被誤當成互動回應造成靜默錯誤，並讓 CI 行為可預期。

### Decision: delete 同步清理 keychain 且容忍失敗

`delete` 移除 profile 後呼叫 `internal/secret.Delete`；not-found 視為成功，其他錯誤印警告但 exit 0。理由：profile 已從 config 移除為主要成功條件，keychain 清理為盡力而為；對使用者而言 profile 不存在即等同金鑰無法被 `KeyResolver` 取得（因 profile 已不存在）。

## Implementation Contract

**Behavior（終端使用者觀察）:**

- `byok config add`（無旗標）於 TTY 互動詢問 name/provider/api_base/default_model/api_key/storage，完成後 profile 已建立且金鑰依選擇儲存。
- `byok config add --name N --provider openai --api-base U --default-model M --api-key K` 等同參數模式，金鑰預設存 keychain。
- `byok config update --name N [--provider ...] [--api-base ...] [--default-model ...] [--api-key ...] [--key-storage ...]` 更新既有 profile，未提供欄位保留原值；提供 `--api-key` 時依 `--key-storage` 處理。
- `byok config delete --name N` 移除 profile 並清理 keychain（盡力）。
- `set-key`/`del-key`/`import-keys` 不再存在，執行時 cobra 回報未知子指令。
- `list`/`set-default` 行為不變。

**Interface / data shape:**

- `cmd/config.go` 定義 `addCmd`、`updateCmd`（新）、`deleteCmd`（由 `removeCmd` 改名）、`setKeyCmd`/`delKeyCmd`/`importKeysCmd`（移除）。
- 新增旗標：`--key-storage`（enum `plaintext|keychain`，預設 `keychain`），於 `addCmd`/`updateCmd`。
- 新增 `internal/config/interactive.go`：匯出可測試的提示函式，簽章接收 `io.Reader`/`io.Writer` 與一個 `IsTerminal` 判定注入點，避免測試需真實 TTY。
- `internal/secret` 與 `KeyResolver` 介面不變。

**Failure modes:**

- profile 重複 → exit 1，config 未改。
- profile 不存在（update/delete）→ exit 1。
- keychain backend 不可用且選 `keychain` → exit 1，config 未寫，提示改用 `--key-storage plaintext`。
- 互動模式但非 TTY → exit 1，提示改用參數模式。
- `delete` 的 keychain 刪除失敗（非 not-found）→ 印警告，profile 仍移除，exit 0。

**Acceptance criteria:**

- `cmd/config_test.go` 覆蓋：add 參數/互動、update 各欄位、delete 含/不含 keychain、重複/不存在錯誤、keychain backend 不可用、非 TTY 互動失敗。
- `go test ./... -race` 通過。
- `byok config set-key` 執行回傳 unknown command（由 cobra 預設行為）。
- `internal/secret` 既有測試維持綠燈。

**Scope boundaries:**

- In scope：`cmd/config.go`、`internal/config/interactive.go`、`cmd/config_test.go`、`internal/config/config.go`（profile 欄位更新輔助）、`internal/config/config_test.go`、AGENTS.md「CLI 介面」與「金鑰」段落同步。
- Out of scope：`internal/secret`、`KeyResolver`、`list`/`set-default`、設定檔路徑/格式、launch/update 子命令的啟動版本檢查。

## Risks / Trade-offs

- **破壞性變更影響既有腳本**：使用 `set-key`/`del-key`/`import-keys` 的腳本會中斷 → 於 AGENTS.md 與 README 記錄遷移路徑，並在 release changelog 標記 BREAKING。
- **互動模式測試困難**：需注入 `io.Reader`/`io.Writer` 與 terminal 判定 → 以可注入介面設計 `interactive.go`，單元測試以 bytes buffer 驅動。
- **無 secret-service 環境互動預設為 keychain 會失敗**：使用者第一次新增即遇到 backend-unavailable → 錯誤訊息明確提示 `--key-storage plaintext`，互動模式在 storage 選擇步驟亦列出明碼選項。
- **delete 對孤兒 keychain 條目**：若 config 中 profile 已不存在但 keychain 殘留，`delete` 不會處理（因 profile 不存在直接 exit 1） → 視為非目標，使用者可手動或重新 add 同名 profile 後 delete。
