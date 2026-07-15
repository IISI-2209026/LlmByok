## Problem

`--dry-run` 已支援 pi 的遮罩命令輸出，但現有單元測試未覆蓋 pi renderer，因此暫存 `models.json`、遮罩 API key、`PI_CODING_AGENT_DIR`、`--thinking`、yolo 映射與清理片段的契約可能在未被偵測下退化。

## Root Cause

先前 dry-run 測試僅針對 Codex 與 Claude 建立 renderer assertions；pi 的分支未納入相同的單元測試範圍。

## Proposed Solution

在 `cmd/launch_dry_run_test.go` 加入 pi 專屬測試，以固定 profile、model、effort 與 yolo 輸入驗證輸出包含遮罩 `models.json`、唯一暫存目錄建立與清理、`PI_CODING_AGENT_DIR`、`--model`、`--thinking` 和 pi 的 `--approve` 映射，並斷言實際 API key 不會出現在輸出中。

## Success Criteria

pi dry-run renderer 的單元測試在 Windows 與非 Windows 平台皆可執行，且可偵測金鑰遮罩、暫存資源清理、effort 與 yolo 映射任一項的回歸。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `byok-launch`: 補強 pi dry-run 遮罩命令輸出的單元測試保障。

## Impact

- Affected code:
  - Modified: `cmd/launch_dry_run_test.go`
  - New: none
  - Removed: none
