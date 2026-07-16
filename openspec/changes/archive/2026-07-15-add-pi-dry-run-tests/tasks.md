## 1. Pi dry-run 測試覆蓋

- [x] 1.1 在 `cmd/launch_dry_run_test.go` 實作「Pi dry-run renderer regression coverage」：先新增 `TestRenderLaunchDryRun_PiMasksKeyAndRendersTemporaryConfig`，以 renderer-level assertions 呼叫 `renderLaunchDryRun`，並以語意片段而非完整平台指令字串驗證 `https://example.test/v1`、`real-secret`、`gpt-5`、`high` 與 pi yolo 映射所產生的遮罩金鑰、`models.json`、`PI_CODING_AGENT_DIR`、`pi --model`、`--thinking`、`--approve`，及依執行平台的暫存目錄建立與清理；以 `go test ./cmd -run TestRenderLaunchDryRun_PiMasksKeyAndRendersTemporaryConfig -count=1` 驗證通過且沒有修改 renderer 執行行為。
