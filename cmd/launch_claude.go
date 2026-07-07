package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
	"github.com/IISI-2209026/LlmByok/internal/runner"
)

// claudeBinary 為 claude 啟動時在 PATH 中解析的可執行檔名稱。
const claudeBinary = "claude"

// runLaunchClaude 與 runLaunchCopilot/runLaunchCodex 對等：解析設定檔、
// 選擇 profile、驗證 provider、以 exec.LookPath 解析 claude 可執行檔，
// 再以 runner.LaunchClaude 暫時注入 ANTHROPIC_BASE_URL /
// ANTHROPIC_API_KEY / ANTHROPIC_MODEL 啟動 claude 子程序。
// 父程序環境與 ~/.claude/settings.json 永不被修改。
func runLaunchClaude(cfgPath, profileName, model string, extraArgs []string, stdout, stderr io.Writer) error {
	// 1. 解析設定檔路徑。
	path, err := configPath(cfgPath)
	if err != nil {
		return fmt.Errorf("解析設定檔路徑: %w", err)
	}

	// 2. 載入設定檔；檔案不存在為嚴重錯誤並附上提示。
	cfg, err := config.Load(path)
	if err != nil {
		if isNotExistMsg(err) {
			fmt.Fprintf(stderr, "錯誤：在 %q 找不到設定檔\n", path)
			fmt.Fprintf(stderr, "提示：請先以 `byok config add` 新增 profile\n")
			return errExit
		}
		fmt.Fprintf(stderr, "錯誤：讀取設定檔 %q 失敗: %v\n", path, err)
		return errExit
	}

	// 3. 選擇 profile（指定名稱或預設值）。
	selected := profileName
	if selected == "" {
		selected = cfg.DefaultProfile
	}
	if selected == "" {
		fmt.Fprintf(stderr, "錯誤：未指定 profile 且 %q 中未設定 default_profile\n", path)
		fmt.Fprintf(stderr, "提示：執行 `byok config set-default --name <profile>` 或傳入 --profile\n")
		return errExit
	}
	var profile *config.Profile
	for i := range cfg.Profiles {
		if cfg.Profiles[i].Name == selected {
			profile = &cfg.Profiles[i]
			break
		}
	}
	if profile == nil {
		fmt.Fprintf(stderr, "錯誤：在 %q 找不到 profile %q\n", path, selected)
		names := availableProfileNames(cfg.Profiles)
		if len(names) > 0 {
			fmt.Fprintf(stderr, "可用 profile: %s\n", strings.Join(names, ", "))
		} else {
			fmt.Fprintf(stderr, "尚無任何 profile。請先執行 `byok config add`。\n")
		}
		return errExit
	}

	// 4. Provider 驗證：此版本僅支援 openai。
	provider := profile.Provider
	if provider == "" {
		provider = "openai"
	}
	if provider != "openai" {
		fmt.Fprintf(stderr, "錯誤：profile %q 使用 provider %q；byok 首版僅支援 openai provider\n", profile.Name, provider)
		return errExit
	}

	// 5. 確認 claude 可執行檔可在 PATH 中解析。
	resolved, err := exec.LookPath(claudeBinary)
	if err != nil {
		fmt.Fprintf(stderr, "錯誤：在 PATH 中找不到 %q 可執行檔\n", claudeBinary)
		fmt.Fprintf(stderr, "提示：請先安裝 Claude Code。參見 https://docs.anthropic.com/en/docs/claude-code/overview\n")
		return errExit
	}

	// 6. 解析 API 金鑰（keychain 優先、明碼 fallback）。
	apiKey, _, err := config.Resolver.Resolve(*profile)
	if err != nil {
		fmt.Fprintf(stderr, "錯誤：找不到 profile %q 的金鑰（keychain 與設定檔皆無）\n", profile.Name)
		fmt.Fprintf(stderr, "提示：執行 `byok config set-key %s` 將金鑰存入 keychain\n", profile.Name)
		return errExit
	}
	profile.APIKey = apiKey

	// 7. 解析模型（--model 覆寫 / 單一候選直用 / 多候選互動選單 / 空則錯誤）。
	resolvedModel, err := resolveModelForLaunch(profile, model, os.Stdin, stdout, stderr)
	if err != nil {
		return err
	}

	// 8. 以暫時的 BYOK 環境變數啟動 claude（父程序環境不變）。
	if err := runner.LaunchClaude(profile, resolvedModel, resolved, extraArgs, os.Stdin, stdout, stderr); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// claude 以非零結束碼結束 — 靜默傳遞，不額外印出訊息。
			return errExit
		}
		fmt.Fprintf(stderr, "錯誤：執行 claude 失敗: %v\n", err)
		return errExit
	}
	return nil
}