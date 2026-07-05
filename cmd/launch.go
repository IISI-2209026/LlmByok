package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
	"github.com/IISI-2209026/LlmByok/internal/runner"
	"github.com/spf13/cobra"
)

// Copilot 可執行檔名稱，於啟動時在 PATH 中解析。
const copilotBinary = "copilot"

func newLaunchCmd() *cobra.Command {
	var model, profileName, cfgPath string
	var yolo bool
	c := &cobra.Command{
		Use:   "launch <target>",
		Short: "以 BYOK profile 啟動 Copilot 或 Codex CLI（暫時注入環境變數）",
		Long: `以設定檔中的 profile 啟動指定的目標 CLI，並將 BYOK 設定暫時
注入子程序環境。父程序 byok 與您的 shell 環境永不被改變。

首版僅支援 openai provider 類型。`,
		Example: `  byok launch copilot
  byok launch copilot -y -- --continue
  byok launch codex
  byok launch codex -y -- exec
  byok launch codex --profile my-profile --model gpt-4o`,
		// 接受目標工具名稱（第一位置參數）與 -- 之後的透傳參數。
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "錯誤：必須指定目標工具（目前支援 copilot、codex）\n")
				return errExit
			}
			target := args[0]
			extraArgs := buildExtraArgs(yolo, args[1:])
			switch target {
			case "copilot":
				return runLaunchCopilot(cfgPath, profileName, model, extraArgs, cmd.OutOrStdout(), cmd.ErrOrStderr())
			case "codex":
				return runLaunchCodex(cfgPath, profileName, model, extraArgs, cmd.OutOrStdout(), cmd.ErrOrStderr())
			default:
				fmt.Fprintf(cmd.ErrOrStderr(), "錯誤：不支援的工具 %q（目前支援 copilot、codex）\n", target)
				return errExit
			}
		},
		SilenceUsage: true,
	}
	// 接受目標工具名稱作為位置參數，以便未來擴充
	// （此版本僅支援 "copilot"）。
	c.Flags().StringVar(&model, "model", "", "覆寫 profile 的預設模型")
	c.Flags().StringVar(&profileName, "profile", "", "要使用的 profile 名稱（預設使用 default_profile）")
	c.Flags().StringVar(&cfgPath, "config", "", "設定檔路徑（預設為 ~/.byok/config.yaml）")
	c.Flags().BoolVarP(&yolo, "yolo", "y", false, "啟用目標工具的 yolo 模式（等同附加 --yolo）")
	// 自訂 usage 模板：Usage → Targets → Flags → Examples。
	c.SetUsageTemplate(`Usage:
  {{.UseLine}}

Targets:
  copilot  以 BYOK profile 啟動 GitHub Copilot CLI
  codex    以 BYOK profile 啟動 OpenAI Codex CLI

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Examples:
{{.Example}}
`)
	return c
}

func runLaunchCopilot(cfgPath, profileName, model string, extraArgs []string, stdout, stderr io.Writer) error {
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

	// 3. 選擇設定檔（指定名稱或預設值）。
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

	// 5. 確認 copilot 可執行檔可在 PATH 中解析。
	resolved, err := exec.LookPath(copilotBinary)
	if err != nil {
		fmt.Fprintf(stderr, "錯誤：在 PATH 中找不到 %q 可執行檔\n", copilotBinary)
		fmt.Fprintf(stderr, "提示：請先安裝 GitHub Copilot CLI。參見 https://docs.github.com/copilot/copilot-cli\n")
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

	// 7. 以暫時的 BYOK 環境變數啟動 copilot（父程序環境不變）。
	if err := runner.Launch(profile, model, resolved, extraArgs, os.Stdin, os.Stdout, os.Stderr); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// copilot 以非零結束碼結束 — 靜默傳遞，不額外印出訊息。
			return errExit
		}
		fmt.Fprintf(stderr, "錯誤：執行 copilot 失敗: %v\n", err)
		return errExit
	}
	return nil
}

// availableProfileNames 回傳供錯誤訊息使用的設定檔名稱清單。
func availableProfileNames(profiles []config.Profile) []string {
	names := make([]string, 0, len(profiles))
	for _, p := range profiles {
		names = append(names, p.Name)
	}
	return names
}

// buildExtraArgs 組合 yolo 旗標與透傳參數為 extraArgs。
// yolo 旗標在前，透傳參數在後；兩者皆空時回傳 nil。
func buildExtraArgs(yolo bool, args []string) []string {
	var extraArgs []string
	if yolo {
		extraArgs = append(extraArgs, "--yolo")
	}
	extraArgs = append(extraArgs, args...)
	return extraArgs
}