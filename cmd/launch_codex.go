package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/IISI-2209026/LlmByok/internal/runner"
)

// codexBinary 為 codex 啟動時在 PATH 中解析的可執行檔名稱。
const codexBinary = "codex"

// codexInstallHint 為 codex 可執行檔找不到時的安裝提示。
const codexInstallHint = "請先安裝 Codex CLI。參見 https://developers.openai.com/codex"

// runLaunchCodex 與 runLaunchCopilot 對等：解析設定檔、選擇 profile、
// 驗證 provider、以 exec.LookPath 解析 codex 可執行檔，再以
// runner.LaunchCodex 暫時注入 BYOK_CODEX_API_KEY 與 --config 覆寫啟動
// codex 子程序。父程序環境與 ~/.codex/config.toml 永不被修改。
func runLaunchCodex(cfgPath, profileName, model string, extraArgs []string, stdout, stderr io.Writer, options ...launchOptions) error {
	opt := launchOptions{}
	if len(options) > 0 {
		opt = options[0]
	}
	profile, resolved, err := resolveProfileForLaunch(cfgPath, profileName, codexBinary, codexInstallHint, stderr)
	if err != nil {
		return err
	}

	// 解析模型（--model 覆寫 / 單一候選直用 / 多候選互動選單 / 空則錯誤）。
	resolvedModel, err := resolveModelForLaunch(profile, model, os.Stdin, stdout, stderr)
	if err != nil {
		return err
	}

	// 以暫時的 BYOK 環境變數與 --config 覆寫啟動 codex（父程序環境不變）。
	if err := runner.LaunchCodex(profile, resolvedModel, resolved, extraArgs, os.Stdin, stdout, stderr, opt.effort); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// codex 以非零結束碼結束 — 靜默傳遞，不額外印出訊息。
			return errExit
		}
		fmt.Fprintf(stderr, "錯誤：執行 codex 失敗: %v\n", err)
		return errExit
	}
	return nil
}
