## Context

byok 目前支援 `copilot`、`codex`、`codex-app`、`claude` 四個目標工具的 BYOK 啟動。每個目標的工具都有自己的 `runLaunchX` 函式（`cmd/launch_X.go`）與 runner（`internal/runner/X.go`），透過環境變數將 profile 的 `api_base`、`api_key`、`default_model` 注入子程序環境。

pi CLI（pi.dev, `@earendil-works/pi-coding-agent`）的 BYOK 注入機制與現有目標工具不同：pi 沒有單一環境變數可覆寫 OpenAI 相容端點的 base URL。自訂 base URL 的唯一方式是透過 pi 設定目錄中的 `models.json`，以 provider override 指定 `baseUrl`。pi 的設定目錄可透過 `PI_CODING_AGENT_DIR` 環境變數覆寫（預設 `~/.pi/agent`）。

## Goals / Non-Goals

**Goals:**

- 新增 `byok launch pi` 子指令，與現有目標工具並列。
- pi 的 BYOK 注入採用臨時目錄 + `models.json` + `PI_CODING_AGENT_DIR` 環境變數，不修改使用者 pi 設定檔。
- `--yolo` 旗標對 pi 映射為 `--approve`。
- 更新 README.md 與 AGENTS.md。

**Non-Goals:**

- 不複製使用者既有 `~/.pi/agent/` 內容到臨時目錄（v1 採用空白設定，不保留 sessions/settings）。
- 不支援 `openai` 以外的 provider（與現有目標工具一致）。
- 不透過 `--api-key` CLI 旗標傳遞金鑰（改用 models.json 的 `apiKey` 欄位，與 base URL 注入一致）。
- 不新增 pi 專屬的 `--thinking`、`--no-context-files` 等旗標（使用者可透過 `--` 透傳）。

## Decisions

### PI_CODING_AGENT_DIR + 臨時 models.json 注入方式

pi 沒有像 `ANTHROPIC_BASE_URL` 或 `BYOK_CODEX_API_KEY` 這樣的單一環境變數可覆寫 base URL。自訂 base URL 只能透過 `models.json` 的 provider override：

```json
{"providers": {"openai": {"baseUrl": "<api_base>", "apiKey": "<api_key>"}}}
```

`PI_CODING_AGENT_DIR` 環境變數可覆寫 pi 的整個設定目錄。byok 建立臨時目錄，在內放置 `models.json`，再以 `PI_CODING_AGENT_DIR` 指向該目錄。子程序結束後刪除臨時目錄。

**替代方案考量：** 可透過 `--api-key` CLI 旗標傳遞金鑰，但 base URL 仍需 models.json，因此金鑰也一併放入 models.json 以保持注入機制一致。複製使用者既有 `~/.pi/agent/` 到臨時目錄再覆蓋 models.json 可保留 sessions/settings，但增加複雜度且與 BYOK 暫時注入的定位不符，v1 不採用。

### --yolo 映射為 --approve

pi 沒有 `--dangerously-skip-permissions` 或 `--yolo` 旗標。最接近的 `--approve`（`-a`）會自動信任專案本機檔案，跳過信任提示。byok 將 `--yolo`/`-y` 對 pi 映射為 `--approve`。

### 使用 resolveProfileForLaunch 共用輔助函式

`runLaunchPi` 比照 codex/codex-app 呼叫 `resolveProfileForLaunch` 共用輔助函式（步驟 1–6：解析設定檔路徑、載入設定檔、選擇 profile、provider 驗證、LookPath、解析金鑰），再以 `runner.LaunchPi` 啟動子程序。

### --model 透過 CLI 旗標傳遞

pi 的 `--model` 旗標接受 `provider/id` 格式或單一名稱。byok 將 profile 的 `default_model`（或 `--model` 覆寫值）作為 `--model` CLI 旗標傳遞給 pi 子程序。models.json 中不設定 model，以 CLI 旗標為優先。

### 臨時目錄清理

`LaunchPi` 在 `cmd.Run()` 完成後（無論成功或失敗）以 `defer os.RemoveAll(tempDir)` 清理臨時目錄。

## Implementation Contract

**Behavior:**

- `byok launch pi` 啟動 pi 子程序，注入 BYOK 設定（base URL + API key + model）而不修改使用者 pi 設定檔。
- 父程序環境不被修改；`PI_CODING_AGENT_DIR` 僅出現在子程序環境中。
- `--yolo`/`-y` 對 pi 附加 `--approve` 旗標。
- `--model` 覆寫 profile 的 `default_model`。
- `--` 透傳參數原樣附加。

**Interface / data shape:**

- `runLaunchPi(cfgPath, profileName, model string, extraArgs []string, stdout, stderr io.Writer) error` — 在 `cmd/launch_pi.go` 新增，呼叫 `resolveProfileForLaunch(cfgPath, profileName, piBinary, piInstallHint, stderr)`，再呼叫 `runner.LaunchPi(profile, model, resolved, extraArgs, os.Stdin, stdout, stderr)`。
- `piBinary = "pi"` — PATH 中查找的可執行檔名稱。
- `piInstallHint = "請先安裝 pi CLI。參見 https://pi.dev/docs/latest"` — 找不到時的安裝提示。
- `BuildPiEnv(profile *config.Profile, tempDir string) []string` — 以 `os.Environ()` 為起點，覆寫 `PI_CODING_AGENT_DIR=<tempDir>`，其餘環境變數不變。
- `LaunchPi(profile *config.Profile, modelOverride, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer) error` — 建立臨時目錄、寫入 `models.json`（`{"providers":{"openai":{"baseUrl":"<api_base>","apiKey":"<api_key>"}}}`）、以 `BuildPiEnv` 組裝環境、附加 `--model <model>` 到 extraArgs 前端、啟動子程序、`defer` 清理臨時目錄。
- `models.json` 的 `apiKey` 欄位使用 profile 的明碼金鑰（已由 KeyResolver 解析）。

**Failure modes:**

- pi 不在 PATH → 印錯誤並 exit 1（與其他目標工具一致）。
- 設定檔不存在/profile 找不到/非 openai provider/金鑰找不到 → 印錯誤並 exit 1（與其他目標工具一致）。
- 臨時目錄建立失敗 → 印錯誤並 exit 1。
- models.json 寫入失敗 → 印錯誤並 exit 1。
- pi 以非零結束碼結束 → 靜默傳遞 exit 1（與其他目標工具一致）。
- 臨時目錄清理失敗 → 靜默忽略（不影響 exit code）。

**Acceptance criteria:**

- `byok launch pi` 以預設 profile 啟動 pi 子程序，子程序環境包含 `PI_CODING_AGENT_DIR` 指向臨時目錄，臨時目錄含 `models.json` 且 `providers.openai.baseUrl` = profile `api_base`、`providers.openai.apiKey` = profile `api_key`。
- `--model` 覆寫時，pi 子程序命令列包含 `--model <override>`。
- `--yolo` 時，pi 子程序命令列包含 `--approve`。
- 父程序環境不包含 `PI_CODING_AGENT_DIR`。
- `go test ./... -race` 全數通過。

**Scope boundaries:**

- In scope: `cmd/launch_pi.go`、`internal/runner/pi.go`、`cmd/launch.go` dispatch、`cmd/launch_pi_test.go`、`internal/runner/pi_test.go`、`cmd/launch_dispatch_test.go` pi case、`README.md`、`AGENTS.md`。
- Out of scope: 複製使用者 pi 設定、支援 openai 以外 provider、pi 專屬旗標（`--thinking` 等）。

## Risks / Trade-offs

- [PI_CODING_AGENT_DIR 覆寫整個設定目錄] → BYOK 啟動期間使用者無法存取既有 pi sessions/settings。此為 BYOK 暫時注入的可接受代價；使用者以不同 API 端點執行 pi 時，既有設定通常不適用。
- [臨時目錄殘留] → `defer os.RemoveAll` 確保正常/異常結束皆清理；清理失敗時靜默忽略，不影響功能。
- [models.json 格式變更] → pi 官方文件記載的 models.json provider override 格式穩定；若未來格式變更，需更新 `LaunchPi` 的 JSON 結構。
