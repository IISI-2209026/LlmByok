package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/IISI-2209026/LlmByok/internal/runner"
)

// piBinary 為 pi 啟動時在 PATH 中解析的可執行檔名稱。
const piBinary = "pi"

// piInstallHint 為 pi 可執行檔找不到時的安裝提示。
const piInstallHint = "請先安裝 pi CLI。參見 https://pi.dev/docs/latest"

// runLaunchPi 與 runLaunchCodex 對等：解析設定檔、選擇 profile、
// 驗證 provider、以 exec.LookPath 解析 pi 可執行檔，再以
// runner.LaunchPi 暫時注入 PI_CODING_AGENT_DIR（指向臨時目錄含
// models.json）啟動 pi 子程序。父程序環境與 ~/.pi/agent/models.json
// 永不被修改。
func runLaunchPi(cfgPath, profileName, model string, extraArgs []string, stdout, stderr io.Writer) error {
	profile, resolved, err := resolveProfileForLaunch(cfgPath, profileName, piBinary, piInstallHint, stderr)
	if err != nil {
		return err
	}

	// 以臨時目錄 + models.json + PI_CODING_AGENT_DIR 啟動 pi（父程序環境不變）。
	if err := runner.LaunchPi(profile, model, resolved, extraArgs, os.Stdin, stdout, stderr); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// pi 以非零結束碼結束 — 靜默傳遞，不額外印出訊息。
			return errExit
		}
		fmt.Fprintf(stderr, "錯誤：執行 pi 失敗: %v\n", err)
		return errExit
	}
	return nil
}