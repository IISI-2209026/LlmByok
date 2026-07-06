package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/IISI-2209026/LlmByok/internal/runner"
)

// runLaunchCodexApp 啟動 Codex 桌面版（codex app 子命令）。
// 與 runLaunchCodex 相同的 profile 解析與金鑰注入邏輯，差異僅在於
// 透過 runner.LaunchCodexApp 將 app 子命令插入為命令列第一個參數。
// 父程序環境與 ~/.codex/config.toml 永不被修改。
func runLaunchCodexApp(cfgPath, profileName, model string, extraArgs []string, stdout, stderr io.Writer) error {
	profile, resolved, err := resolveProfileForLaunch(cfgPath, profileName, codexBinary, codexInstallHint, stderr)
	if err != nil {
		return err
	}

	// 以暫時的 BYOK 環境變數與 --config 覆寫啟動 codex app（父程序環境不變）。
	if err := runner.LaunchCodexApp(profile, model, resolved, extraArgs, os.Stdin, stdout, stderr); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// codex app 以非零結束碼結束 — 靜默傳遞，不額外印出訊息。
			return errExit
		}
		fmt.Fprintf(stderr, "錯誤：執行 codex app 失敗: %v\n", err)
		return errExit
	}
	return nil
}