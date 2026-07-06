## 1. 互動式提示基礎建設

- [x] [P] 1.1 建立 `internal/config/interactive.go`，匯出可測試的提示函式（如 `PromptString(reader io.Reader, writer io.Writer, label string) (string, error)` 與 `PromptSecret` 使用 `golang.org/x/term.ReadPassword`），並匯出可注入的 terminal 判定介面（例如 `type IsTerminalFunc func(fd int) bool`，預設為 `term.IsTerminal`）。契約：呼叫端可注入非 TTY 的 `io.Reader`/`io.Writer` 與 `IsTerminalFunc` 以驅動單元測試，無需真實 TTY。驗收：`internal/config/interactive_test.go` 以 `bytes.Buffer` 驅動，斷言各欄位提示順序與空白 api_key 回傳空字串。
- [x] [P] 1.2 於 `internal/config/config.go` 新增 profile 欄位更新輔助函式（如 `ApplyProfileUpdates(p *Profile, name, provider, apiBase, defaultModel *string)`），將非 nil 指標的欄位套用至既有 profile，nil 指標保留原值。契約：呼叫端以指標表達「未提供該欄位」，避免零值與「清空」混淆。驗收：`internal/config/config_test.go` 新增 `TestApplyProfileUpdates_KeepsUnspecified` 與 `TestApplyProfileUpdates_OverwritesSpecified`。

## 2. add 命令整合金鑰儲存

- [x] 2.1 重構 `cmd/config.go` 的 `addCmd` 為參數模式：新增 `--key-storage plaintext|keychain` 旗標（預設 `keychain`），當提供 `--api-key` 時依儲存選擇呼叫 `internal/secret.Store`（清空 config 的 `api_key`）或寫入明碼並刪除既有 keychain 條目（`internal/secret.Delete`，容忍 not-found）。實作 Decision: 預設金鑰儲存為 keychain，明確旗標切換明碼。當 keychain backend 不可用且選 `keychain` 時印出 backend-unavailable 錯誤並 exit 1，config 不寫入。涵蓋需求 `Add a profile`。驗收：`cmd/config_test.go` 新增 `TestAdd_KeyStorageKeychain`、`TestAdd_KeyStoragePlaintextClearsKeychain`、`TestAdd_KeychainBackendUnavailableFails`、`TestAdd_DuplicateNameRejected`、`TestAdd_FirstProfileBecomesDefault`。
- [x] 2.2 為 `addCmd` 加入互動模式觸發（未提供 `--name`/`--provider`/`--api-base`/`--default-model`/`--api-key` 任一旗標時），依序提示 name、provider（預設 `openai`）、api_base、default_model、api_key（可空）、storage（預設 `keychain`），api_key 使用 `PromptSecret` 非回顯輸入。實作 Decision: 互動模式以「無任何 profile 欄位旗標」觸發 與 Decision: 非 TTY 時互動模式明確失敗：當 `IsTerminalFunc` 回傳 false 時印出錯誤提示改用參數模式並 exit 1。驗收：`cmd/config_test.go` 新增 `TestAdd_InteractiveNonTTYFails`（注入非 TTY 判定，斷言 exit 1 且 config 未建立）與 `TestAdd_InteractivePromptsAllFields`（注入 buffer 輸入，斷言 profile 與金鑰依選擇儲存）。

## 3. update 命令

- [x] 3.1 新增 `cmd/config.go` 的 `updateCmd`，接受 `--name`（必填）與選填 `--provider`/`--api-base`/`--default-model`/`--api-key`/`--key-storage`。未提供的欄位保留原值（透過任務 1.2 的指標輔助）；提供 `--api-key` 時套用與 `add` 相同的儲存處理。profile 不存在時印錯並 exit 1。涵蓋需求 `Update an existing profile`。驗收：`cmd/config_test.go` 新增 `TestUpdate_KeepsUnspecifiedFields`、`TestUpdate_ApiKeyIntoKeychain`、`TestUpdate_NonExistentRejected`。
- [x] 3.2 為 `updateCmd` 加入互動模式：當只提供 `--name` 而未提供其他欄位旗標時進入互動模式提示其餘欄位與儲存選擇，沿用 `add` 的非 TTY 失敗邏輯。驗收：`cmd/config_test.go` 新增 `TestUpdate_InteractiveNonTTYFails` 與 `TestUpdate_InteractiveUpdatesFields`。

## 4. delete 命令與 keychain 同步

- [x] 4.1 將 `cmd/config.go` 的 `removeCmd` 改名為 `deleteCmd`（命令名 `delete`，檔案內對應 `remove` 的處理改為 `delete` 語意），移除 profile 後呼叫 `internal/secret.Delete` 同步清理 keychain。實作 Decision: delete 同步清理 keychain 且容忍失敗：not-found 視為成功；其他錯誤印警告命名 profile 與失敗原因但 exit 0；profile 不存在時 exit 1 且不碰 keychain；被刪 profile 為 `default_profile` 時清空該欄位。涵蓋需求 `Remove a profile`（RENAMED to `Delete a profile`）。驗收：`cmd/config_test.go` 新增 `TestDelete_SyncsKeychain`、`TestDelete_NoKeychainEntrySucceeds`、`TestDelete_NonExistentRejected`、`TestDelete_KeychainFailureWarnsButExits0`、`TestDelete_ClearsDefaultProfile`。

## 5. 移除舊金鑰子指令

- [x] 5.1 自 `cmd/config.go` 移除 `setKeyCmd`/`delKeyCmd`/`importKeysCmd` 及其 `RunE` 處理與對應 cobra 註冊，並移除 `cmd/config_test.go` 中專屬這些指令的測試案例。實作 Decision: 以 profile 為中心折疊金鑰操作。涵蓋被移除的需求 `Set API key in OS keychain`、`Delete API key from OS keychain`、`Batch import plaintext keys into keychain`。驗收：`go test ./cmd/... -race` 不再參照這些指令；手動執行 `byok config set-key foo`（透過 stub main 或 `go run`）回傳 cobra unknown command 訊息且非零結束碼。

## 6. 測試收斂與文件同步

- [x] 6.1 執行 `go test ./... -race`，確認 `cmd/config_test.go`、`internal/config/config_test.go`、`internal/config/interactive_test.go` 全數通過且無資料競爭。驗收：`go test ./... -race` 回傳 exit 0；`go vet ./...` 無警告。
- [x] 6.2 更新 `AGENTS.md`「CLI 介面」與「金鑰」段落，反映 `remove`→`delete` 改名、`set-key`/`del-key`/`import-keys` 移除、`update` 新增、`--key-storage` 旗標與預設 keychain、互動模式與非 TTY 行為。同步 `README.md` 中 `byok config` 子命令列表與金鑰操作說明。驗收：`AGENTS.md` 條列的 `byok config` 子命令集合為 `add`/`update`/`delete`/`list`/`set-default`，且明確記載遷移路徑；README 的命令範例與 `--help` 一致（以 `go run ./cmd/byok config --help` 比對）。
