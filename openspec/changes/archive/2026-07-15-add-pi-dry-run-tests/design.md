## Context

`renderLaunchDryRun` 已針對 pi 產生含暫存目錄、遮罩 `models.json`、`PI_CODING_AGENT_DIR` 與清理程序的 shell 片段。現有測試只驗證其他 target，故 pi branch 缺少回歸保護。

## Goals / Non-Goals

**Goals:**

- 為 pi dry-run renderer 加入平台無關的單元測試。
- 驗證輸出遮罩 API key、建立並清理 pi 暫存目錄、寫入 `models.json`、設定 `PI_CODING_AGENT_DIR`，以及保留解析後的 model、effort 和 yolo 映射。

**Non-Goals:**

- 不修改 `renderLaunchDryRun`、pi runner 或 CLI 執行行為。
- 不測試實際 shell 執行、檔案建立或 pi 子程序啟動。
- 不新增或調整 API key resolver 的實作。

## Decisions

### Use renderer-level assertions for pi dry-run

直接呼叫 `renderLaunchDryRun`，以有意義的 profile 和 `launchOptions` 檢查輸出片段。這沿用現有 Codex 與 Claude renderer test 模式，且不需要平台 shell 或外部 pi binary。

### Assert semantic fragments instead of one platform-specific full string

測試會檢查 Windows 與 POSIX 變體共同的語意片段，並以目前 GOOS 對應的建立/清理語法檢查資源生命週期。如此可避免因無關的 quoting 格式變動造成脆弱測試，同時仍保障 pi dry-run contract。

## Implementation Contract

**Behavior:** `TestRenderLaunchDryRun_PiMasksKeyAndRendersTemporaryConfig` SHALL render pi dry-run output from profile API base `https://example.test/v1`、API key `real-secret`、model `gpt-5`、effort `high` 與 pi yolo args. The test SHALL reject output containing `real-secret` and require quoted `***`, `models.json`, `PI_CODING_AGENT_DIR`, `pi --model`, `gpt-5`, `--thinking`, `high`, and `--approve`.

**Platform contract:** On Windows the test SHALL require the PowerShell temporary-directory creation and `finally` cleanup tokens. On non-Windows it SHALL require the POSIX `mktemp` and `trap` cleanup tokens.

**Failure mode:** Any absent semantic token, leaked API key, or missing platform cleanup token SHALL fail the Go unit test.

**Acceptance criteria:** `go test ./cmd -run TestRenderLaunchDryRun_PiMasksKeyAndRendersTemporaryConfig -count=1` passes on the host platform.

**Scope:** Only `cmd/launch_dry_run_test.go` changes. Renderer and runtime behavior remain out of scope.

## Risks / Trade-offs

- [Renderer quoting changes while preserving semantics can require test updates] → Assertions target contract-level tokens rather than a complete generated shell script.
