# byok

`byok` 是一支命令列工具，讓你在啟動 GitHub Copilot CLI、OpenAI Codex CLI 或 Claude Code 時，可以**暫時**使用自己的 OpenAI 相容 API 金鑰（BYOK = Bring Your Own Key），**不會**修改系統環境變數或 Shell 設定檔。它會從 `~/.byok/config.yaml` 這個 YAML 設定檔讀取金鑰相關設定，只把各目標工具的 BYOK 所需環境變數注入到子行程中；當子行程結束後，你原本的環境完全不受影響。

### 主要功能

- **以設定檔（profile）管理金鑰** — 每個 profile 各自儲存 Provider、API Base、API Key 與 Default Model 四個設定值。
- **一行指令啟動** — `byok launch copilot --model gemma4` 即可用選定 profile 的金鑰啟動 Copilot，並可選擇性地覆寫模型。同樣支援 `byok launch codex` 與 `byok launch claude`。
- **暫時性的環境注入** — 環境變數只注入到目標工具子行程，永遠不會寫入系統環境變數或 Shell 設定檔。
- **支援三個目標工具** — Copilot CLI、Codex CLI 與 Claude Code，皆使用同一套 BYOK profile 機制。
- **第一版** 僅支援 OpenAI 相容端點（provider 類型為 `openai`）。

### 解決什麼問題

Copilot CLI、Codex CLI 與 Claude Code 的 BYOK 功能每次使用時，都需要手動匯出環境變數：

- **Copilot**：`COPILOT_PROVIDER_BASE_URL`、`COPILOT_PROVIDER_TYPE`、`COPILOT_PROVIDER_API_KEY`、`COPILOT_MODEL`
- **Codex**：`BYOK_CODEX_API_KEY` 加上 `--config` 旗標覆寫
- **Claude**：`ANTHROPIC_BASE_URL`、`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL`

手動設定既繁瑣又會污染 Shell 環境。`byok` 從設定檔自動化這件工作，做到每次啟動時才臨時注入。

## 前置需求

- **Go** 1.26 以上（`go.mod` 中宣告的版本）。
- **Git**（用來 clone 本專案）。
- **Copilot CLI、Codex CLI 或 Claude Code** 已安裝並放在 `PATH` 上（僅 `launch` 指令需要，依你要使用的目標工具而定）。
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
    default_model: gpt-4o
  - name: local-ollama
    provider: openai
    api_base: http://localhost:11434
    api_key: ""
    default_model: llama3.2
default_profile: openai-official
```

### 欄位說明

| 欄位           | 說明                                                                          |
| -------------- | ---------------------------------------------------------------------------- |
| `name`         | profile 名稱，用於 `--profile` 選取。檔案內必須唯一。                          |
| `provider`     | provider 類型。第一版僅接受 `openai`。                                          |
| `api_base`     | OpenAI 相容端點的 Base URL（例如 `https://api.openai.com/v1`）。               |
| `api_key`      | API 金鑰字串。本機伺服器（如 Ollama）不需金鑰時用 `""`。                          |
| `default_model`| 模型名稱，依目標工具注入為對應環境變數（Copilot: `COPILOT_MODEL`、Codex: `--config model=`、Claude: `ANTHROPIC_MODEL`）；若 `--model` 有指定則以 `--model` 為準。 |

### 安全性提醒

設定檔以**明文**儲存 API 金鑰，請妥善保護該檔案：

- macOS/Linux 可將權限設為 `600`：`chmod 600 ~/.byok/config.yaml`。
- Windows 可透過檔案內容 > 安全性，將存取權限限制為你的使用者帳戶。
- 絕對不要把 `~/.byok/config.yaml` commit 到版本控制。
- **推薦**：`byok config add`/`update` 預設以 `--key-storage keychain` 將金鑰存入 OS keychain，設定檔中不再保留明碼 `api_key`。

## 使用說明

### `byok launch <target>`

以某個 BYOK profile 啟動指定的目標 CLI（`copilot`、`codex` 或 `claude`），將 BYOK 設定暫時注入子程序環境；你的 Shell 環境永不被改變。

- `copilot`：四個 `COPILOT_*` 環境變數只注入到 `copilot` 子行程。
- `codex`：API 金鑰以 `BYOK_CODEX_API_KEY` 環境變數注入 `codex` 子行程，並透過 `--config` 旗標覆寫模型與連線設定；`~/.codex/config.toml` 完全不受影響。
- `claude`：`ANTHROPIC_BASE_URL`、`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL` 三個環境變數只注入到 `claude` 子行程；`~/.claude/settings.json` 完全不受影響。

**Targets：**

| Target    | 說明                                                |
| --------- | --------------------------------------------------- |
| `copilot` | 以 BYOK profile 啟動 GitHub Copilot CLI。           |
| `codex`   | 以 BYOK profile 啟動 OpenAI Codex CLI。             |
| `claude`  | 以 BYOK profile 啟動 Claude Code。                  |

**旗標：**

| 旗標        | 說明                                              |
| ----------- | ------------------------------------------------ |
| `--model`   | 此次啟動覆寫 profile 的 `default_model`。          |
| `--profile` | 依名稱選取 profile。未指定則使用 `default_profile`。 |
| `--config`  | 覆寫設定檔路徑（預設 `~/.byok/config.yaml`）。        |
| `-y`, `--yolo` | 啟用目標工具的 yolo 模式：copilot/codex 附加 `--yolo`，claude 附加 `--dangerously-skip-permissions`。 |
| `--`        | 之後的參數原樣透傳給目標工具（不解析、不驗證）。     |

**範例：**

```bash
# 使用預設 profile 與其 default_model 啟動 copilot
byok launch copilot

# 覆寫模型啟動
byok launch copilot --model gemma4
byok launch codex --model gpt-4o
byok launch claude --model claude-sonnet-4-5

# 指定特定 profile 啟動
byok launch copilot --profile local-ollama
byok launch codex --profile openai-official
byok launch claude --profile openai-official

# 使用自訂設定檔路徑
byok launch copilot --config /tmp/my-config.yaml --profile openai-official

# 啟用 yolo 模式（-y 為 --yolo 短形式）
byok launch copilot -y
byok launch codex -y
byok launch claude -y

# 透傳參數給目標工具（-- 之後原樣轉發）
byok launch copilot -- skills
byok launch copilot -- continue --model x
byok launch codex -- exec
byok launch claude -- --resume

# yolo + 透傳同時使用（yolo 旗標在前，透傳參數在後）
byok launch copilot -y -- skills
byok launch codex -y -- exec
byok launch claude -y -- review this
```

### `byok config add`

新增一個 profile 到設定檔。若檔案不存在會自動建立。若目前沒有設定 `default_profile`，新加入的 profile 會自動設為預設。若已有同名 profile 則會報錯且不修改檔案。

未提供任何欄位旗標（`--name`、`--provider`、`--api-base`、`--default-model`、`--api-key`）時進入**互動模式**，於終端依序提示各欄位與金鑰儲存選擇（需 TTY，非 TTY 印錯並 exit 1）。

**旗標：**

| 旗標             | 說明                                       |
| ---------------- | ----------------------------------------- |
| `--name`         | profile 名稱。                              |
| `--provider`     | provider 類型（目前僅支援 `openai`）。        |
| `--api-base`     | API base URL。                            |
| `--api-key`      | API 金鑰（無金鑰的本機伺服器用 `""`）。        |
| `--default-model`| 預設模型名稱。                               |
| `--key-storage`  | 金鑰儲存位置：`keychain`（預設）或 `plaintext`。|
| `--config`       | 覆寫設定檔路徑。                            |

**範例：**

```bash
byok config add \
  --name openai-official \
  --provider openai \
  --api-base https://api.openai.com/v1 \
  --api-key sk-xxxx \
  --default-model gpt-4o
# 金鑰預設存入 keychain；設定檔中不含明碼 api_key
```

互動模式：

```bash
byok config add
# 依序提示 profile 名稱、provider、API base URL、預設模型、API key、金鑰儲存
```

### `byok config update`

更新既有 profile 的欄位。未提供的欄位保留原值。僅提供 `--name` 而未提供其他欄位旗標時進入**互動模式**（需 TTY）。

提供 `--api-key` 時依 `--key-storage` 處理金鑰；`--api-key ""` 清除既有金鑰（同步刪除 keychain 條目）。

**旗標：**

| 旗標             | 說明                                       |
| ---------------- | ----------------------------------------- |
| `--name`         | 要更新的 profile 名稱（必填）。             |
| `--provider`     | provider 類型。                            |
| `--api-base`     | API base URL。                            |
| `--api-key`      | API 金鑰（設為空字串清除金鑰）。              |
| `--default-model`| 預設模型名稱。                               |
| `--key-storage`  | 金鑰儲存位置：`keychain`（預設）或 `plaintext`。|
| `--config`       | 覆寫設定檔路徑。                            |

**範例：**

```bash
byok config update --name openai-official --api-key sk-new-key
# 新金鑰存入 keychain，舊金鑰被覆寫
```

### `byok config list`

列出設定檔中所有 profile。API 金鑰會遮罩：只顯示前 4 與後 4 個字元，中間以 `...` 連接；空金鑰顯示為空。

**旗標：**

| 旗標      | 說明                          |
| --------- | ----------------------------- |
| `--config`| 覆寫設定檔路徑。                |

**範例：**

```bash
byok config list
```

### `byok config delete`

依名稱刪除 profile，並同步清理 keychain 中的對應金鑰（盡力而為；keychain 刪除失敗僅印警告，profile 仍已移除）。找不到 profile 時報錯且不碰 keychain。若被刪除的 profile 正是 `default_profile`，則該欄位會被清空。

**旗標：**

| 旗標      | 說明                          |
| --------- | ----------------------------- |
| `--name`  | 要刪除的 profile 名稱（必填）。 |
| `--config`| 覆寫設定檔路徑。                |

**範例：**

```bash
byok config delete --name local-ollama
```

### `byok config set-default`

變更 `launch` 在未指定 `--profile` 時使用的 `default_profile`。

**旗標：**

| 旗標      | 說明                            |
| --------- | ------------------------------- |
| `--name`  | 要設為預設的 profile 名稱（必填）。 |
| `--config`| 覆寫設定檔路徑。                  |

**範例：**

```bash
byok config set-default --name local-ollama
```

### 金鑰管理（OS keychain）

`byok` 支援將 API 金鑰儲存於作業系統的 keychain（Windows Credential Manager、macOS Keychain、Linux Secret Service），避免明文寫入設定檔。金鑰以 `profile:<名稱>` 為 key 存入，service 名稱為 `byok`。

金鑰管理已整合至 profile 生命週期：

- **新增金鑰**：`byok config add`/`update` 時以 `--key-storage keychain`（預設）將金鑰存入 keychain，設定檔中不含明碼 `api_key`。可用 `--key-storage plaintext` 改存明碼。
- **刪除金鑰**：`byok config delete` 移除 profile 時同步清理 keychain。

`byok launch` 啟動時會自動依以下順序解析金鑰：**keychain 優先 → 設定檔明碼 fallback → 兩者皆無則報錯**。

> **遷移路徑**：舊版獨立指令 `set-key`/`del-key`/`import-keys` 已移除。請改用 `byok config update --name <profile> --api-key <key>` 更新金鑰，或 `byok config delete` 刪除。
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

### Claude BYOK 運作原理

執行 `byok launch claude` 時，`byok` 會複製當前行程的環境，**只**在這份副本中覆寫三個 `ANTHROPIC_*` 變數（`ANTHROPIC_BASE_URL`、`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL`），然後以這份修改後的環境啟動 `claude` 作為子行程。父行程（你的 Shell）的環境永遠不會被修改 — 一旦 `claude` 子行程結束，一切恢復原狀。

- **不寫入 `~/.claude/settings.json`** — 所有覆寫僅透過環境變數傳遞，`byok` 不會讀取或修改你既有的 Claude Code 設定檔。
- **`-y`/`--yolo` 映射** — `byok` 的 `--yolo` 旗標對 claude target 會附加 `--dangerously-skip-permissions`（Claude Code 的權限跳過旗標），而非 `--yolo`。

## 官方文件

- **Copilot CLI BYOK** — <https://docs.github.com/zh/copilot/how-tos/copilot-cli/customize-copilot/use-byok-models>
- **Codex CLI BYOK（自訂模型供應商）** — <https://developers.openai.com/codex/config-advanced#custom-model-providers>
- **Codex CLI BYOK（替代模型供應商驗證）** — <https://developers.openai.com/codex/auth#alternative-model-providers>
- **Claude Code 模型設定（第三方部署）** — <https://code.claude.com/docs/zh-TW/model-config#pin-models-for-third-party-deployments>

## 疑難排解

- **找不到設定檔** — 先執行 `byok config add ...` 建立 `~/.byok/config.yaml`。
- **`copilot` 不在 PATH 上** — 使用 `launch` 前請先安裝 Copilot CLI。
- **`codex` 不在 PATH 上** — 使用 `launch` 前請先安裝 Codex CLI。
- **`claude` 不在 PATH 上** — 使用 `launch` 前請先安裝 Claude Code。
- **非 `openai` 的 provider 被拒** — 第一版僅支援 `openai` provider 類型。
- **設定檔格式錯誤** — 檢查 `~/.byok/config.yaml` 的 YAML 語法（縮排、引號）。
- **Windows 上找不到 `make`** — 直接用 `go build` / `go install`，或透過 `winget install GnuWin32.Make` 安裝 `make`。

## 授權與貢獻

本專案以 MIT 授權（詳見 [LICENSE](LICENSE)）。歡迎貢獻 — 請至專案開 issue 或 pull request。