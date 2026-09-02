# byok

`byok` 是一支命令列工具，讓你在啟動 GitHub Copilot CLI、OpenAI Codex CLI、Claude Code 或 pi CLI 時，可以**暫時**使用自己的 OpenAI 相容 API 金鑰（BYOK = Bring Your Own Key），**不會**修改系統環境變數或 Shell 設定檔。它會從 `~/.byok/config.yaml` 這個 YAML 設定檔讀取金鑰相關設定，只把各目標工具的 BYOK 所需環境變數注入到子行程中；當子行程結束後，你原本的環境完全不受影響。

### 主要功能

- **以設定檔（profile）管理金鑰** — 每個 profile 各自儲存 Provider、API Base、API Key 與 Default Model 四個設定值。
- **一行指令啟動** — `byok launch copilot --model gemma4` 即可用選定 profile 的金鑰啟動 Copilot，並可選擇性地覆寫模型。同樣支援 `byok launch codex`、`byok launch codex-app`（Codex 桌面版）、`byok launch claude` 與 `byok launch pi`。
- **暫時性的環境注入** — 環境變數只注入到目標工具子行程，永遠不會寫入系統環境變數或 Shell 設定檔。
- **模型 token 上限宣告與覆寫** — profile 可設定 `default_model_limits` 與個別模型的 `context_window_tokens` / `max_output_tokens`，啟動時依優先序注入各目標工具的對應環境變數或旗標，並可用 `--context-window-tokens` / `--max-output-tokens` 一次性覆寫（詳見下方「模型 token 上限」）。
- **支援五個目標工具** — Copilot CLI、Codex CLI、Claude Code 與 pi CLI，皆使用同一套 BYOK profile 機制。
- **第一版** 僅支援 OpenAI 相容端點（provider 類型為 `openai`）。

### 解決什麼問題

Copilot CLI、Codex CLI 與 Claude Code 的 BYOK 功能每次使用時，都需要手動匯出環境變數：

- **Copilot**：`COPILOT_PROVIDER_BASE_URL`、`COPILOT_PROVIDER_TYPE`、`COPILOT_PROVIDER_API_KEY`、`COPILOT_MODEL`
- **Codex**：`BYOK_CODEX_API_KEY` 加上 `--config` 旗標覆寫
- **Claude**：`ANTHROPIC_BASE_URL`、`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL`
- **Pi**：pi 沒有單一環境變數可覆寫 base URL，`byok` 建立臨時目錄放置 `models.json`（覆寫 `openai` provider 的 `baseUrl` 與 `apiKey`），再以 `PI_CODING_AGENT_DIR` 環境變數指向臨時目錄；模型透過 `--model` CLI 旗標傳遞。`~/.pi/agent/models.json` 完全不受影響，臨時目錄於 pi 結束後自動清理。

手動設定既繁瑣又會污染 Shell 環境。`byok` 從設定檔自動化這件工作，做到每次啟動時才臨時注入。

## 前置需求

- **Go** 1.26 以上（`go.mod` 中宣告的版本）。
- **Git**（用來 clone 本專案）。
- **Copilot CLI、Codex CLI、Claude Code 或 pi CLI** 已安裝並放在 `PATH` 上（僅 `launch` 指令需要，依你要使用的目標工具而定）。
- 一組 **OpenAI 相容的 API 金鑰**（若是 Ollama 這類本機伺服器則可用空字串）。

## 安裝 Go

如果你從沒寫過 Go，請依照以下方式安裝 Go 工具鏈。

### Windows

```powershell
winget install GoLang.Go
```

或者從 <https://go.dev/dl/> 下載 MSI 安裝程式並執行。安裝完成後驗證：

```powershell
go version
```

### macOS

```bash
brew install go
```

或從 <https://go.dev/dl/> 下載 `.pkg` 安裝程式並執行。驗證：

```bash
go version
```

### Linux

從 <https://go.dev/dl/> 下載壓縮檔，解壓縮並加入 PATH：

```bash
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

### 關於 GOPATH / GOBIN

Go 會把安裝的執行檔放在 `GOBIN` 目錄，預設為 `$HOME/go/bin`（Windows 為 `%USERPROFILE%\go\bin`）。`go install` 指令會把執行檔放到這裡。建議將此目錄加入 `PATH`，方便直接執行安裝的工具：

- **Windows**：Go 的 MSI 安裝程式會自動把 `%USERPROFILE%\go\bin` 加入 PATH。
- **macOS/Linux**：在 `~/.bashrc` 或 `~/.zshrc` 加入 `export PATH=$PATH:$(go env GOPATH)/bin`。

## Clone 專案

```bash
git clone https://github.com/IISI-2209026/LlmByok.git
cd LlmByok
```

## 安裝

### 方式一：自 GitHub Releases 下載預建二進位（推薦）

前往 [Releases 頁面](https://github.com/IISI-2209026/LlmByok/releases) 下載對應平台的資產，檔名格式為 `byok-<version>-<os>-<arch>.<ext>`：

| 平台           | 資產名稱範例                              |
| -------------- | ----------------------------------------- |
| Windows amd64  | `byok-0.1.0-windows-amd64.zip`            |
| Linux amd64    | `byok-0.1.0-linux-amd64.tar.gz`           |
| macOS amd64    | `byok-0.1.0-darwin-amd64.tar.gz`          |
| macOS arm64    | `byok-0.1.0-darwin-arm64.tar.gz`          |

下載後解壓縮，將 `byok`（或 `byok.exe`）放到 `PATH` 上的目錄，再驗證：

```bash
byok --version
# 輸出：byok version 0.1.0
```

> 以 Releases 預建二進位安裝為啟用 `byok update` 自我更新的建議路徑 — `byok update` 會自同一個 Releases 來源下載新版並替換執行檔。

### 方式二：以 Go 工具鏈安裝

若已安裝 Go 1.26 以上：

```bash
go install github.com/IISI-2209026/LlmByok/cmd/byok@latest
```

安裝後執行檔位於 `GOBIN`（預設 `~/go/bin`），確認已加入 `PATH` 後驗證：

```bash
byok --version
```

## 建置

有三種方式建置 `byok` 執行檔：

```bash
# 1. 建置到 ./dist（Windows 會產生 dist\byok.exe；macOS/Linux 產生 dist/byok）
go build -o dist/byok ./cmd/byok

# 2. 安裝到 GOBIN（之後可在 PATH 任何地方直接執行 `byok`）
go install ./cmd/byok

# 3. 使用 Makefile（輸出同方式 1）
make build
```

> 注意：Windows 上可能沒有安裝 `make`。可使用方式 1 或 2，或自行安裝 `make`（例如 `winget install GnuWin32.Make` 或以 Chocolatey 安裝）。

## 執行

不想建置也可直接執行：

```bash
go run ./cmd/byok <指令> [旗標]
```

或執行已建置的執行檔：

```bash
# macOS / Linux
./dist/byok <指令>

# Windows
dist\byok.exe <指令>
```

## 設定檔

`byok` 預設從 `~/.byok/config.yaml` 讀取設定。你可以用 `byok config add` 建立（推薦），也可以手動編輯。以下是一份可直接複製的範例，包含兩個 profile：

```yaml
profiles:
  - name: openai-official
    provider: openai
    api_base: https://api.openai.com/v1
    api_key: sk-your-openai-key-here
    models:
      - gpt-4o
      - gpt-4o-mini
  - name: local-ollama
    provider: openai
    api_base: http://localhost:11434
    api_key: ""
    models:
      - llama3.2
default_profile: openai-official
```

### 欄位說明

| 欄位       | 說明                                                                          |
| ---------- | ---------------------------------------------------------------------------- |
| `name`     | profile 名稱，用於 `--profile` 選取。檔案內必須唯一。                          |
| `provider` | provider 類型。第一版僅接受 `openai`。                                          |
| `api_base` | OpenAI 相容端點的 Base URL（例如 `https://api.openai.com/v1`）。               |
| `api_key`  | API 金鑰字串。本機伺服器（如 Ollama）不需金鑰時用 `""`。                          |
| `models`   | 候選模型清單。`byok launch <target>` 在未帶 `--model` 時依此清單決定模型：僅一個則直接使用；多個且 stdin 為終端機時顯示上下鍵互動選單；多個但非終端機時報錯（請改用 `--model`）；為空時報錯（請先以 `byok config set-models` 設定）。模型依目標工具注入為對應環境變數（Copilot: `COPILOT_MODEL`、Codex: `--config model=`、Claude: `ANTHROPIC_MODEL`；Pi: `--model` CLI 旗標）。若 `--model` 有指定則一律以 `--model` 為準。項目支援兩種寫法：純字串（如 `- gpt-4o`）或 mapping（如 `- name: gpt-5.4`，可附 `context_window_tokens:` / `max_output_tokens:`），詳見下方「模型 token 上限」。 |
| `default_model_limits` | 選填。profile 層級的模型 token 上限預設。詳見下方「模型 token 上限」。 |

> **舊設定檔相容性**：若你的設定檔仍使用舊版單一 `default_model` 欄位，`byok` 載入時會自動將其遷移為單元素 `models` 清單；下次寫入時舊欄位即不再出現。

### 模型 token 上限（`default_model_limits` 與模型層級設定）

設定檔可為模型宣告 token 上限，`byok launch` 會依優先序將生效值注入目標工具（見「使用說明 → `byok launch <target>`」的一次性 token 上限覆寫與支援矩陣）。

- **profile 層級 `default_model_limits`（選填）** — 該 profile 中所有模型的預設上限，包含兩個**選填**欄位：
  - `context_window_tokens`：context window token 上限（正的 64 位元整數）。
  - `max_output_tokens`：單次回應最大輸出 token 上限（正的 64 位元整數）。

  未設定的欄位即為 nil（不會注入對應覆寫）。
- **`models` 項目支援兩種寫法**：
  - **純字串（scalar，舊式）**：`- gpt-4o`，無個別上限。
  - **mapping（新式）**：`- name: gpt-5.4`，並可附上 `context_window_tokens:` / `max_output_tokens:`（同樣為正的 64 位元整數、皆選填）。

完整範例（單一 profile 同時包含 `default_model_limits`、完整設定、部分設定與舊式 scalar 三種模型項目）：

```yaml
profiles:
  - name: gpt5-direct
    provider: openai
    api_base: https://api.openai.com/v1
    api_key: sk-your-openai-key-here
    default_model_limits:        # profile 層級預設（兩個欄位皆選填）
      context_window_tokens: 200000
      max_output_tokens: 32000
    models:
      - name: gpt-5.4            # mapping：完整設定（context + output）
        context_window_tokens: 1000000
        max_output_tokens: 128000
      - name: o4-mini            # mapping：部分設定（只給 max_output_tokens）
        max_output_tokens: 64000
      - gpt-4o                   # 舊式 scalar：未設個別上限，沿用 default_model_limits
default_profile: gpt5-direct
```

> **向下相容**：只使用純字串 `models` 的既有設定檔不需任何修改即可繼續使用。
>
> **降級提醒**：`default_model_limits` 與 mapping 形式的 `models` 項目只有較新版的 `byok` 才能解析。若要改回**較舊版本**的 `byok` 二進位，請先從設定檔移除 `default_model_limits`，並把 mapping 項目改寫回純字串模型名稱（舊版無法解析 mapping 項目）。API 金鑰與 keychain 內容不受影響，無需任何遷移。

**設定檔驗證**：載入設定檔時，`byok` 會拒絕以下情況並印出錯誤（訊息會標明所屬的 profile、模型與欄位名稱）後以非零結束碼退出：

- 同一 profile 內有空字串的模型名稱。
- 同一 profile 內出現重複的模型名稱（scalar 與 mapping 混用時同樣以名稱判斷重複）。
- 任一模型項目或 `default_model_limits` 的 `context_window_tokens` / `max_output_tokens` 為 0 或負數。

context 與 output 兩個欄位之間**沒有**大小先後規則（例如 `max_output_tokens` 可以大於 `context_window_tokens`），`byok` 不做跨欄位檢查。

### 安全性提醒

設定檔以**明文**儲存 API 金鑰，請妥善保護該檔案：

- macOS/Linux 可將權限設為 `600`：`chmod 600 ~/.byok/config.yaml`。
- Windows 可透過檔案內容 > 安全性，將存取權限限制為你的使用者帳戶。
- 絕對不要把 `~/.byok/config.yaml` commit 到版本控制。
- **推薦**：`byok config add`/`update` 預設以 `--key-storage keychain` 將金鑰存入 OS keychain，設定檔中不再保留明碼 `api_key`。

## 使用說明

### `byok launch <target>`

以某個 BYOK profile 啟動指定的目標 CLI（`copilot`、`codex`、`codex-app`、`claude` 或 `pi`），將 BYOK 設定暫時注入子程序環境；你的 Shell 環境永不被改變。

- `copilot`：四個 `COPILOT_*` 環境變數只注入到 `copilot` 子行程。
- `codex`：API 金鑰以 `BYOK_CODEX_API_KEY` 環境變數注入 `codex` 子行程，並透過 `--config` 旗標覆寫模型與連線設定；`~/.codex/config.toml` 完全不受影響。
- `codex-app`：與 `codex` 相同的 BYOK 機制，差異僅在於子程序命令列插入 `app` 子命令以啟動 Codex 桌面版（`codex app [--config ...] [...]`）。
- `claude`：`ANTHROPIC_BASE_URL`、`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL` 三個環境變數只注入到 `claude` 子行程；`~/.claude/settings.json` 完全不受影響。
- `pi`：建立臨時目錄放置 `models.json`（覆寫 `openai` provider 的 `baseUrl` 與 `apiKey`），以 `PI_CODING_AGENT_DIR` 環境變數指向臨時目錄，模型透過 `--model` CLI 旗標傳遞至 `pi` 子行程；`~/.pi/agent/models.json` 完全不受影響。

**Targets：**

| Target      | 說明                                                |
| ----------- | --------------------------------------------------- |
| `copilot`   | 以 BYOK profile 啟動 GitHub Copilot CLI。           |
| `codex`     | 以 BYOK profile 啟動 OpenAI Codex CLI。             |
| `codex-app` | 以 BYOK profile 啟動 OpenAI Codex 桌面版（`codex app`）。 |
| `claude`    | 以 BYOK profile 啟動 Claude Code。                  |
| `pi`        | 以 BYOK profile 啟動 pi CLI。                       |

**旗標：**

| 旗標        | 說明                                              |
| ----------- | ------------------------------------------------ |
| `--model`   | 此次啟動明確指定模型，覆寫 profile 的候選 `models` 清單（不顯示互動選單）。 |
| `--profile` | 依名稱選取 profile。未指定則使用 `default_profile`。 |
| `--config`  | 覆寫設定檔路徑（預設 `~/.byok/config.yaml`）。        |
| `-y`, `--yolo` | 啟用目標工具的 yolo 模式：copilot/codex/codex-app 附加 `--yolo`，claude 附加 `--dangerously-skip-permissions`，pi 附加 `--approve`。 |
| `--context-window-tokens` | 覆寫 context window token 上限（正整數；僅這次啟動生效，不寫入設定檔）。詳見下方「一次性 token 上限覆寫」。 |
| `--max-output-tokens`     | 覆寫單次回應最大輸出 token 上限（正整數；僅這次啟動生效，不寫入設定檔）。詳見下方「一次性 token 上限覆寫」。 |
| `--`        | 之後的參數原樣透傳給目標工具（不解析、不驗證）。     |

**範例：**

```bash
# 使用預設 profile 啟動 copilot
# profile 僅一個候選模型時直接使用；多個時於終端顯示上下鍵選單
byok launch copilot

# 明確指定模型啟動（覆寫候選清單，不顯示選單）
byok launch copilot --model gemma4
byok launch codex --model gpt-4o
byok launch codex-app --model gpt-4o
byok launch claude --model claude-sonnet-4-5
byok launch pi --model claude-sonnet-4-5

# 指定特定 profile 啟動
byok launch copilot --profile local-ollama
byok launch codex --profile openai-official
byok launch codex-app --profile openai-official
byok launch claude --profile openai-official
byok launch pi --profile openai-official

# 使用自訂設定檔路徑
byok launch copilot --config /tmp/my-config.yaml --profile openai-official

# 啟用 yolo 模式（-y 為 --yolo 短形式）
byok launch copilot -y
byok launch codex -y
byok launch codex-app -y
byok launch claude -y
byok launch pi -y

# 透傳參數給目標工具（-- 之後原樣轉發）
byok launch copilot -- skills
byok launch copilot -- continue --model x
byok launch codex -- exec
byok launch codex-app -- exec
byok launch claude -- --resume
byok launch pi -- --continue

# yolo + 透傳同時使用（yolo 旗標在前，透傳參數在後）
byok launch copilot -y -- skills
byok launch codex -y -- exec
byok launch codex-app -y -- exec
byok launch claude -y -- review this
byok launch pi -y -- --continue
```

`--effort`、`--sub-model` 與 `--dry-run` 都是選填旗標。effort 會暫時映射為 Copilot 的
`--reasoning-effort`、Codex/Codex App 的 `--config model_reasoning_effort`、Claude 的
`CLAUDE_CODE_EFFORT_LEVEL`（並啟用 `CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1`），或 pi 的
`--thinking`。Claude 也會以 `CLAUDE_CODE_SUBAGENT_MODEL` 接收 opaque 的 `--sub-model`；
其他 target 接受但忽略 `--sub-model`。

```bash
byok launch codex --model gpt-5 --effort high
byok launch claude --effort high --sub-model claude-haiku-4-5
byok launch codex --model gpt-5 --effort high --dry-run
```

`--dry-run` 會解析 profile 與模型，但不解析 API key、不檢查 target 是否安裝，也不啟動子程序。
Windows 輸出 PowerShell，其他平台輸出 POSIX shell；所有金鑰位置固定輸出為已引用的 `***`，
複製命令前請先替換為實際金鑰。pi 的輸出會包含建立 masked `models.json`、執行與清理暫存目錄的完整片段。

#### 一次性 token 上限覆寫（`--context-window-tokens` / `--max-output-tokens`）

`byok launch <target>` 另支援這兩個選填旗標，為**這一次啟動**覆寫模型 token 上限（不寫入設定檔、不改變 profile）：

```bash
byok launch copilot --context-window-tokens 1000000 --max-output-tokens 128000
byok launch claude --context-window-tokens 200000
byok launch pi --model my-model --max-output-tokens 64000
```

兩個旗標的值都必須是**正整數**；傳入 0、負數或非整數時，`byok` 會在解析 API 金鑰與檢查目標工具是否安裝**之前**印出錯誤並以 exit 1 結束。

**生效優先序（每個欄位獨立解析）** — context 與 output 兩個欄位各自依以下順序取值，互不影響：

1. CLI 旗標（`--context-window-tokens` / `--max-output-tokens`）
2. 所選模型的對應欄位（`--model` 指定且命中 profile `models` 清單時，使用該模型的欄位；未帶 `--model` 時為從清單選出的模型）
3. profile 的 `default_model_limits` 對應欄位
4. 未設定

四層都未命中即為「未設定」，此時 `byok` 對該欄位**不注入任何 token 覆寫**（不加旗標、不設環境變數）。若 `--model` 指定的名稱不在 profile 的 `models` 清單中，則只會從 CLI 旗標與 profile 的 `default_model_limits` 推導上限，**不會**借用其他模型項目的設定。

**各 target 對應方式（token 上限支援矩陣）：**

| Target      | Context window 覆寫                                                           | Max output tokens 覆寫 |
| ----------- | ----------------------------------------------------------------------------- | ---------------------- |
| `copilot`   | 環境變數 `COPILOT_PROVIDER_MAX_PROMPT_TOKENS`                                 | 環境變數 `COPILOT_PROVIDER_MAX_OUTPUT_TOKENS` |
| `codex`     | `--config model_context_window=<值>`                                          | 不支援（印警告後繼續） |
| `codex-app` | 與 `codex` 相同（`--config model_context_window=<值>`；`app` 仍為第一個參數） | 不支援（印警告後繼續） |
| `claude`    | 環境變數 `CLAUDE_CODE_MAX_CONTEXT_TOKENS`                                     | 環境變數 `CLAUDE_CODE_MAX_OUTPUT_TOKENS` |
| `pi`        | 臨時 `models.json` 的 `providers.openai.modelOverrides.<model>.contextWindow` | 同一路徑的 `.maxTokens` |

**語義注意事項：**

- **Copilot** — `COPILOT_PROVIDER_MAX_PROMPT_TOKENS` 的官方語義是「maximum prompt tokens」；`byok` 將通用的 context 值**直接**映射進去，**不會**再自行扣除 max output tokens。
- **Codex / Codex App** — 不支援 max output tokens。若該欄位有生效值，`byok` 會向 stderr 印出警告（標明 target、參數、值與其來源）後照常繼續啟動。
- **Claude** — 模型名稱**原樣傳遞**；`byok` 不會自動附加 `[1m]` extended-context 後綴，需要 extended context 的使用者請自行在模型名稱中包含官方後綴。
- **Pi** — 除 `models.json` 的 `modelOverrides` 外，`byok` 會另寫一份臨時 `settings.json`，將 `compaction.reserveTokens` 設為生效的 max output tokens（作為回應餘裕的推導值），這**不是**可供使用者調整的 compaction 門檻設定。

**限制：**

- 這些值只作用於「向下游 target 呈現／宣告的容量」，**不會**放大後端模型真實的能力（gateway 端可能夾住數值或直接回錯）。
- 這**不是**跨 target 統一的 compaction 門檻控制。
- pi 的臨時 `settings.json` 與暫存目錄會於啟動結束後一併移除；`byok` 一律不修改任何使用者設定檔。

### `byok config add <profile name>`

新增一個 profile 到設定檔。若檔案不存在會自動建立。若目前沒有設定 `default_profile`，新加入的 profile 會自動設為預設。若已有同名 profile 則會報錯且不修改檔案。profile 名稱為第一位置參數；候選模型由 `byok config set-models` 維護，`add` 不設定模型。

未提供任何欄位旗標（`--provider`、`--api-base`、`--api-key`）時進入**互動模式**，於終端依序提示各欄位與金鑰儲存選擇（需 TTY，非 TTY 印錯並 exit 1）。

**旗標：**

| 旗標             | 說明                                       |
| ---------------- | ----------------------------------------- |
| `<profile name>` | profile 名稱（第一位置參數，必填）。           |
| `--provider`     | provider 類型（目前僅支援 `openai`）。        |
| `--api-base`     | API base URL。                            |
| `--api-key`      | API 金鑰（無金鑰的本機伺服器用 `""`）。        |
| `--key-storage`  | 金鑰儲存位置：`keychain`（預設）或 `plaintext`。|
| `--config`       | 覆寫設定檔路徑。                            |

**範例：**

```bash
byok config add openai-official \
  --provider openai \
  --api-base https://api.openai.com/v1 \
  --api-key sk-xxxx
# 金鑰預設存入 keychain；設定檔中不含明碼 api_key
# 接著以 config set-models 設定候選模型：
byok config set-models openai-official --model gpt-4o --model gpt-4o-mini
```

互動模式：

```bash
byok config add openai-official
# 依序提示 provider、API base URL、API key、金鑰儲存（不再提示模型）
```

### `byok config update <profile name>`

更新既有 profile 的欄位。未提供的欄位保留原值。profile 名稱為第一位置參數（必填）；未提供任何欄位旗標時進入**互動模式**（需 TTY）。候選模型由 `byok config set-models` 維護，`update` 不修改模型清單。

提供 `--api-key` 時依 `--key-storage` 處理金鑰；`--api-key ""` 清除既有金鑰（同步刪除 keychain 條目）。

**旗標：**

| 旗標             | 說明                                       |
| ---------------- | ----------------------------------------- |
| `<profile name>` | 要更新的 profile 名稱（第一位置參數，必填）。   |
| `--provider`     | provider 類型。                            |
| `--api-base`     | API base URL。                            |
| `--api-key`      | API 金鑰（設為空字串清除金鑰）。              |
| `--key-storage`  | 金鑰儲存位置：`keychain`（預設）或 `plaintext`。|
| `--config`       | 覆寫設定檔路徑。                            |

**範例：**

```bash
byok config update openai-official --api-key sk-new-key
# 新金鑰存入 keychain，舊金鑰被覆寫
```

### `byok config set-models <profile name>`

設定指定 profile 的候選模型清單，**整批覆寫**原有清單。profile 名稱為第一位置參數。可重複使用 `--model` 指定多個候選模型；未提供 `--model` 且 stdin 為終端機時進入互動模式，逐行輸入模型識別碼直至空行結束。profile 不存在時報錯且不修改檔案；結果為空清單時報錯。此指令為 `byok config` 的子指令。

**token 上限語義**：`set-models` 以模型**名稱**整批覆寫清單，各模型的 `context_window_tokens` / `max_output_tokens` 遵循以下規則：

- 新舊清單皆存在的同名模型：**保留**各自原有的 token 上限。
- 新出現的名稱：從「無上限」開始。
- 被移除的名稱：連同其 token 上限一併消失。
- profile 層級的 `default_model_limits`：不受 `set-models` 影響。

`byok config list` 與 `launch` 的互動模型選單皆只顯示模型名稱，不顯示各模型的 token 上限。

**旗標：**

| 旗標             | 說明                                       |
| ---------------- | ----------------------------------------- |
| `<profile name>` | profile 名稱（第一位置參數，必填）。           |
| `--model`        | 候選模型識別碼（可重複，如 `--model a --model b`）。 |
| `--config`       | 覆寫設定檔路徑。                            |

**範例：**

```bash
byok config set-models openai-official --model gpt-4o --model gpt-4o-mini
# 覆寫為 [gpt-4o, gpt-4o-mini]

# 互動模式（終端，逐行輸入至空行）：
byok config set-models openai-official
```

### `byok config list`

列出設定檔中所有 profile，包含每個 profile 的候選 `models` 清單（逗號分隔；僅顯示模型名稱，不含各模型的 token 上限）。API 金鑰會遮罩：只顯示前 4 與後 4 個字元，中間以 `...` 連接；空金鑰顯示為空。

**旗標：**

| 旗標      | 說明                          |
| --------- | ----------------------------- |
| `--config`| 覆寫設定檔路徑。                |

**範例：**

```bash
byok config list
```

### `byok config delete <profile name>`

依名稱刪除 profile，並同步清理 keychain 中的對應金鑰（盡力而為；keychain 刪除失敗僅印警告，profile 仍已移除）。找不到 profile 時報錯且不碰 keychain。若被刪除的 profile 正是 `default_profile`，則該欄位會被清空。profile 名稱為第一位置參數。

**旗標：**

| 旗標             | 說明                          |
| ---------------- | ----------------------------- |
| `<profile name>` | 要刪除的 profile 名稱（第一位置參數，必填）。 |
| `--config`       | 覆寫設定檔路徑。                |

**範例：**

```bash
byok config delete local-ollama
```

### `byok config set-default <profile name>`

變更 `launch` 在未指定 `--profile` 時使用的 `default_profile`。profile 名稱為第一位置參數。

**旗標：**

| 旗標             | 說明                            |
| ---------------- | ------------------------------- |
| `<profile name>` | 要設為預設的 profile 名稱（第一位置參數，必填）。 |
| `--config`       | 覆寫設定檔路徑。                  |

**範例：**

```bash
byok config set-default local-ollama
```

### 金鑰管理（OS keychain）

`byok` 支援將 API 金鑰儲存於作業系統的 keychain（Windows Credential Manager、macOS Keychain、Linux Secret Service），避免明文寫入設定檔。金鑰以 `profile:<名稱>` 為 key 存入，service 名稱為 `byok`。

金鑰管理已整合至 profile 生命週期：

- **新增金鑰**：`byok config add`/`update` 時以 `--key-storage keychain`（預設）將金鑰存入 keychain，設定檔中不含明碼 `api_key`。可用 `--key-storage plaintext` 改存明碼。
- **刪除金鑰**：`byok config delete` 移除 profile 時同步清理 keychain。

`byok launch` 啟動時會自動依以下順序解析金鑰：**keychain 優先 → 設定檔明碼 fallback → 兩者皆無則報錯**。

> **遷移路徑**：舊版獨立指令 `set-key`/`del-key`/`import-keys` 已移除。請改用 `byok config update <profile> --api-key <key>` 更新金鑰，或 `byok config delete <profile>` 刪除。
>
> **Linux 注意事項**：keychain 功能依賴 Secret Service D-Bus API（如 `gnome-keyring` 或 `KWallet`）。若環境中無 secret-service daemon，keychain 操作會回傳 backend-unavailable 錯誤；此時可改用 `--key-storage plaintext` 將金鑰以明碼寫入設定檔。

### `byok update`

檢查並自我更新 `byok` 至最新 GitHub Release。依當前版本所屬 channel 自動判定查詢範圍（含 `-dev.` 為 dev channel，否則 stable channel），下載對應平台資產並替換當前執行檔。

- 不加旗標時，查到新版會下載並替換執行檔，完成後提示重新執行。
- 已是最新版本時印出 `已是最新版本 (<version>)`。
- `launch` 與 `update` 以外的子指令完成後，若有新版會在 stderr 印一行提示（可用 `BYOK_NO_UPDATE_CHECK=1` 停用）。

**旗標：**

| 旗標        | 說明                                                         |
| ----------- | ----------------------------------------------------------- |
| `--check`   | 只查詢最新版本，不下載或替換執行檔。                            |
| `--channel` | 覆寫自動 channel 判定（`prerelease` 或 `release`），可跨 channel 更新。 |

**範例：**

```bash
# 檢查並更新到當前 channel 最新版
byok update

# 只查詢不替換
byok update --check

# 覆寫 channel 查預發布版本
byok update --channel prerelease --check

# 覆寫 channel 更新到正式版本
byok update --channel release
```

## 版本管理

byok 使用 [Semantic Versioning](https://semver.org/)（`MAJOR.MINOR.PATCH`）管理版本號。

### `byok --version`

顯示當前版本號（cobra 內建 `--version` flag，輸出格式 `byok version <Version>`）。

```bash
byok --version
# 輸出：byok version 0.1.0
```

### Canonical base 版號

版號的唯一來源（canonical base）為 `internal/version/version.go` 的 `Version` 字面值（semver、無 `v` prefix、無後綴），目前為 `0.1.0`。Makefile 與 Release workflow 皆以 `sed` 讀取此字面值，不引入額外 VERSION 檔或以 Git tag 為來源。

### 版本號與發布流程

- **develop 預發布**：推送 develop → Release workflow 產生預發布，二進位版號 `<base>-dev.<run_number>`、tag `v<base>-dev.<run_number>`（如 `0.1.0-dev.42` / `v0.1.0-dev.42`）、標記為 prerelease。`run_number` 取自 GitHub Actions `github.run_number`，確保每次推送產生唯一 tag、不再撞 tag。
- **main 穩定發布**：推送 main → Release workflow 產生穩定發布，二進位版號 `<base>`、tag `v<base>`（如 `0.1.0` / `v0.1.0`）。
- **晉升流程**：
  1. develop 累積預發布至可發布狀態。
  2. merge develop → main 並推送 main → Release workflow 自動產生穩定發布 `v<base>`。
  3. 於 develop 將 `internal/version/version.go` 的 base 晉升到下一個 patch（或其他 semver 遞增）並 commit。
  4. push 到 develop，使下一輪預發布使用更高的 base（如 `0.1.1-dev.N`），下一輪 main 發布即為 `0.1.1`。

### 自動發布

push 至 `main` 或 `develop` 分支時，`.github/workflows/release.yml` 會：

1. 讀取 `internal/version/version.go` 中的 canonical base 版號
2. 依分支推導完整版號與 tag：
   - `main`：`<base>` / `v<base>`（穩定發布）
   - `develop`：`<base>-dev.<run_number>` / `v<base>-dev.<run_number>`（預發布）
3. 以 matrix 策略平行建置四個平台執行檔：
   - `windows/amd64`（zip）
   - `linux/amd64`（tar.gz）
   - `darwin/amd64`（tar.gz）
   - `darwin/arm64`（tar.gz）
4. 使用 `softprops/action-gh-release` 建立 GitHub Release，以版號為 git tag，並附加所有平台壓縮檔

建置時透過 Go ldflags 注入完整版號：

```bash
go build -ldflags "-X github.com/IISI-2209026/LlmByok/internal/version.Version=0.1.0" -o byok ./cmd/byok
```

## OpenTelemetry

`byok` 支援透過設定檔統一注入 OpenTelemetry（OTEL）遙測設定至各 target 子程序。各 target 原生 OTEL 支援方式各異，`byok` 會自動轉譯為對應格式。

### 設定格式

在 `~/.byok/config.yaml` 加入頂層 `telemetry` 區段：

```yaml
telemetry:
  enabled: true
  service_name: "my-team"        # 選填，組合為 <service_name>-<agent-name>
  headers:                        # 選填，認證用 headers
    Authorization: "Bearer token"
  grpc:                           # 選填，gRPC OTLP endpoint
    endpoint: "http://localhost:4317"
  http:                           # 選填，HTTP OTLP endpoint
    endpoint: "http://localhost:4318"
    protocol: "protobuf"          # protobuf（預設）或 json
```

### 各 Target 對應方式

| Target | 選用 Endpoint | 注入方式 |
|--------|--------------|----------|
| Copilot CLI | HTTP only | 環境變數：`COPILOT_OTEL_ENABLED`、`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_EXPORTER_OTLP_PROTOCOL`、`OTEL_EXPORTER_OTLP_HEADERS`、`OTEL_SERVICE_NAME` |
| Codex / Codex App | gRPC 優先，HTTP fallback | `--config otel.*` 旗標 + `OTEL_SERVICE_NAME` 環境變數 |
| Claude | gRPC 優先，HTTP fallback | 環境變數：`CLAUDE_CODE_ENABLE_TELEMETRY`、`OTEL_METRICS_EXPORTER`、`OTEL_LOGS_EXPORTER`、`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_EXPORTER_OTLP_PROTOCOL`、`OTEL_EXPORTER_OTLP_HEADERS`、`OTEL_SERVICE_NAME` |
| Pi | HTTP only | 環境變數：`PI_OTEL_ENABLED`、`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_SERVICE_NAME`（不注入 headers） |

### Service Name 組合

設定 `service_name: "x"` 時，各 target 注入的 `OTEL_SERVICE_NAME` 為：
- Copilot → `x-github-copilot`
- Codex / Codex App → `x-codex-cli`
- Claude → `x-claude-code`
- Pi → `x-pi-coding-agent`

未設定 `service_name` 時不注入，各 target 使用原生預設。

### 注意事項

- 建議同時設定 `grpc` 與 `http` 兩組 endpoint，確保所有 target 皆可注入
- 僅設 `grpc` 時，Copilot 與 Pi 不會注入 telemetry（靜默跳過，不報錯）
- Pi 需使用者自行安裝 `pi-otel-telemetry` extension
- `--dry-run` 輸出會包含 telemetry 環境變數/旗標，headers 值以 `***` mask

### 官方文件來源

- **Copilot CLI**：`copilot help monitoring`
- **Codex**：https://github.com/openai/codex — `codex-rs/otel/README.md`
- **Claude Code**：https://docs.anthropic.com/en/docs/claude-code/monitoring-usage
- **Pi**：https://github.com/mprokopov/pi-otel-telemetry

## 運作原理（暫時性注入）

### Copilot BYOK

執行 `byok launch copilot` 時，`byok` 會複製當前行程的環境，**只**在這份副本中覆寫四個 `COPILOT_*` 變數（`COPILOT_PROVIDER_BASE_URL`、`COPILOT_PROVIDER_TYPE`、`COPILOT_PROVIDER_API_KEY`、`COPILOT_MODEL`），然後以這份修改後的環境啟動 `copilot` 作為子行程。父行程（你的 Shell）的環境永遠不會被修改 — 一旦 `copilot` 子行程結束，一切恢復原狀，因此平常使用 GitHub 託管模型的 Copilot 體驗完全不受影響。

### Codex BYOK 運作原理

執行 `byok launch codex` 時，`byok` 會以類似但不同的機制啟動 `codex`：

1. **環境變數承載 API 金鑰** — `byok` 將 profile 的 `api_key` 以 `BYOK_CODEX_API_KEY` 注入 `codex` 子行程環境（覆寫既存值），父程序環境不變。
2. **`--config` 旗標覆寫連線設定** — `byok` 透過多組 `--config` 旗標向 `codex` 指定：
   - `model="<預設模型或 --model 覆寫>"`
   - `model_provider="byok"`
   - `model_providers.byok.base_url="<profile.api_base>"`
   - `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`
3. **不寫入 `~/.codex/config.toml`** — 所有覆寫僅透過命令列 `--config` 旗標傳遞，`byok` 不會讀取或修改你既有的 Codex 設定檔。

命令列順序為 `codex [<--config ...>] [<--yolo>] [<透傳參數...]`，與 copilot 路徑一致（`--yolo` 在前、透傳在後）。

### Codex App BYOK 運作原理

執行 `byok launch codex-app` 時，機制與 `codex` 完全相同（同樣的 `BYOK_CODEX_API_KEY` 環境變數與 `--config` 旗標覆寫），唯一差異是命令列插入 `app` 子命令以啟動 Codex 桌面版：

命令列順序為 `codex app [--config ...] [--yolo] [透傳參數...]`。同樣不寫入 `~/.codex/config.toml`。

### Claude BYOK 運作原理

執行 `byok launch claude` 時，`byok` 會複製當前行程的環境，**只**在這份副本中覆寫三個 `ANTHROPIC_*` 變數（`ANTHROPIC_BASE_URL`、`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL`），然後以這份修改後的環境啟動 `claude` 作為子行程。父行程（你的 Shell）的環境永遠不會被修改 — 一旦 `claude` 子行程結束，一切恢復原狀。

- **不寫入 `~/.claude/settings.json`** — 所有覆寫僅透過環境變數傳遞，`byok` 不會讀取或修改你既有的 Claude Code 設定檔。
- **`-y`/`--yolo` 映射** — `byok` 的 `--yolo` 旗標對 claude target 會附加 `--dangerously-skip-permissions`（Claude Code 的權限跳過旗標），而非 `--yolo`。

### Pi BYOK 運作原理

執行 `byok launch pi` 時，`byok` 建立臨時目錄放置 `models.json`（覆寫 `openai` provider 的 `baseUrl` 與 `apiKey`），再以 `PI_CODING_AGENT_DIR` 環境變數指向臨時目錄；模型透過 `--model` CLI 旗標傳遞。`~/.pi/agent/models.json` 完全不受影響，臨時目錄於 pi 結束後自動清理。

## 官方文件

- **Copilot CLI BYOK** — <https://docs.github.com/zh/copilot/how-tos/copilot-cli/customize-copilot/use-byok-models>
- **Codex CLI BYOK（自訂模型供應商）** — <https://developers.openai.com/codex/config-advanced#custom-model-providers>
- **Codex CLI BYOK（替代模型供應商驗證）** — <https://developers.openai.com/codex/auth#alternative-model-providers>
- **Claude Code 模型設定（第三方部署）** — <https://code.claude.com/docs/zh-TW/model-config#pin-models-for-third-party-deployments>
- **pi CLI**：https://pi.dev/docs/latest/providers

## 疑難排解

- **找不到設定檔** — 先執行 `byok config add ...` 建立 `~/.byok/config.yaml`。
- **`copilot` 不在 PATH 上** — 使用 `launch` 前請先安裝 Copilot CLI。
- **`codex` 不在 PATH 上** — 使用 `launch` 前請先安裝 Codex CLI。
- **`claude` 不在 PATH 上** — 使用 `launch` 前請先安裝 Claude Code。
- **pi CLI**：以 `npm install -g --ignore-scripts @earendil-works/pi-coding-agent` 安裝。參見 https://pi.dev/docs/latest
- **非 `openai` 的 provider 被拒** — 第一版僅支援 `openai` provider 類型。
- **設定檔格式錯誤** — 檢查 `~/.byok/config.yaml` 的 YAML 語法（縮排、引號）。
- **Windows 上找不到 `make`** — 直接用 `go build` / `go install`，或透過 `winget install GnuWin32.Make` 安裝 `make`。

## 授權與貢獻

本專案以 MIT 授權（詳見 [LICENSE](LICENSE)）。歡迎貢獻 — 請至專案開 issue 或 pull request。
