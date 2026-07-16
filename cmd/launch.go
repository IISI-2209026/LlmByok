package cmd

import (
	"errors"
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
	var model, profileName, cfgPath, effort, subModel string
	var yolo, dryRun bool
	c := &cobra.Command{
		Use:   "launch <target>",
		Short: "以 BYOK profile 啟動 Copilot、Codex、Codex App、Claude 或 pi CLI（暫時注入環境變數）",
		Long: `以設定檔中的 profile 啟動指定的目標 CLI，並將 BYOK 設定暫時
注入子程序環境。父程序 byok 與您的 shell 環境永不被改變。

首版僅支援 openai provider 類型。`,
		Example: `  byok launch copilot
  byok launch copilot -y -- --continue
  byok launch codex
  byok launch codex -y -- exec
  byok launch codex --profile my-profile --model gpt-4o
  byok launch codex-app
  byok launch codex-app -y -- exec
  byok launch claude
  byok launch claude -y
  byok launch claude --model claude-sonnet-4-5
  byok launch pi
  byok launch pi -y
  byok launch pi --model gpt-4o`,
		// 接受目標工具名稱（第一位置參數）與 -- 之後的透傳參數。
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if err := cmd.Help(); err != nil {
					return err
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "錯誤：必須指定目標工具（目前支援 copilot、codex、codex-app、claude、pi）\n")
				return errExit
			}
			target := args[0]
			if _, ok := launchEffortLevels[target]; !ok {
				fmt.Fprintf(cmd.ErrOrStderr(), "錯誤：不支援的工具 %q（目前支援 copilot、codex、codex-app、claude、pi）\n", target)
				return errExit
			}
			if err := validateLaunchEffort(target, effort); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "錯誤：", err)
				return errExit
			}
			extraArgs := buildExtraArgs(yolo, target, args[1:])
			options := launchOptions{effort: effort, subModel: subModel, dryRun: dryRun}
			if dryRun {
				return runLaunchDryRun(cfgPath, profileName, target, model, options, extraArgs, cmd.OutOrStdout(), cmd.ErrOrStderr())
			}
			switch target {
			case "copilot":
				return runLaunchCopilot(cfgPath, profileName, model, extraArgs, cmd.OutOrStdout(), cmd.ErrOrStderr(), options)
			case "codex":
				return runLaunchCodex(cfgPath, profileName, model, extraArgs, cmd.OutOrStdout(), cmd.ErrOrStderr(), options)
			case "codex-app":
				return runLaunchCodexApp(cfgPath, profileName, model, extraArgs, cmd.OutOrStdout(), cmd.ErrOrStderr(), options)
			case "claude":
				return runLaunchClaude(cfgPath, profileName, model, extraArgs, cmd.OutOrStdout(), cmd.ErrOrStderr(), options)
			case "pi":
				return runLaunchPi(cfgPath, profileName, model, extraArgs, cmd.OutOrStdout(), cmd.ErrOrStderr(), options)
			default:
				fmt.Fprintf(cmd.ErrOrStderr(), "錯誤：不支援的工具 %q（目前支援 copilot、codex、codex-app、claude、pi）\n", target)
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
	c.Flags().StringVar(&effort, "effort", "", "暫時指定目標工具的 reasoning effort")
	c.Flags().StringVar(&subModel, "sub-model", "", "暫時指定 Claude subagent model（其他 target 忽略）")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "只輸出遮罩金鑰的等效命令，不啟動目標工具")
	// 自訂 usage 模板：Usage → Targets → Flags → Examples。
	c.SetUsageTemplate(`Usage:
  {{.UseLine}}

Targets:
  copilot    以 BYOK profile 啟動 GitHub Copilot CLI
  codex      以 BYOK profile 啟動 OpenAI Codex CLI
  codex-app  以 BYOK profile 啟動 OpenAI Codex 桌面版（codex app）
  claude     以 BYOK profile 啟動 Claude Code CLI
  pi         以 BYOK profile 啟動 pi CLI

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Examples:
{{.Example}}
`)
	return c
}

type launchOptions struct {
	effort, subModel string
	dryRun           bool
}

func runLaunchCopilot(cfgPath, profileName, model string, extraArgs []string, stdout, stderr io.Writer, options ...launchOptions) error {
	opt := launchOptions{}
	if len(options) > 0 {
		opt = options[0]
	}
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

	// 7. 解析模型（--model 覆寫 / 單一候選直用 / 多候選互動選單 / 空則錯誤）。
	resolvedModel, err := resolveModelForLaunch(profile, model, os.Stdin, os.Stdout, stderr)
	if err != nil {
		return err
	}

	// 8. 以暫時的 BYOK 環境變數啟動 copilot（父程序環境不變）。
	if err := runner.Launch(profile, resolvedModel, resolved, extraArgs, os.Stdin, stdout, stderr, opt.effort); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// copilot 以非零結束碼結束 — 靜默傳遞，不額外印出訊息。
			return errExit
		}
		fmt.Fprintf(stderr, "錯誤：執行 copilot 失敗: %v\n", err)
		return errExit
	}
	return nil
}

// resolveProfileForLaunch 封裝 runLaunchCodex 步驟 1–6 的共用邏輯：
// 解析設定檔路徑、載入設定檔、選擇 profile、驗證 provider、以
// exec.LookPath 解析可執行檔、解析 API 金鑰。成功時回傳已注入金鑰的
// profile 與可執行檔路徑；失敗時將錯誤訊息寫入 stderr 並回傳 errExit。
// binaryName 為 PATH 中查找的可執行檔名稱；installHint 為找不到時的提示。
func resolveProfileForLaunch(cfgPath, profileName, binaryName, installHint string, stderr io.Writer) (*config.Profile, string, error) {
	// 1. 解析設定檔路徑。
	path, err := configPath(cfgPath)
	if err != nil {
		return nil, "", fmt.Errorf("解析設定檔路徑: %w", err)
	}

	// 2. 載入設定檔；檔案不存在為嚴重錯誤並附上提示。
	cfg, err := config.Load(path)
	if err != nil {
		if isNotExistMsg(err) {
			fmt.Fprintf(stderr, "錯誤：在 %q 找不到設定檔\n", path)
			fmt.Fprintf(stderr, "提示：請先以 `byok config add` 新增 profile\n")
			return nil, "", errExit
		}
		fmt.Fprintf(stderr, "錯誤：讀取設定檔 %q 失敗: %v\n", path, err)
		return nil, "", errExit
	}

	// 3. 選擇 profile（指定名稱或預設值）。
	selected := profileName
	if selected == "" {
		selected = cfg.DefaultProfile
	}
	if selected == "" {
		fmt.Fprintf(stderr, "錯誤：未指定 profile 且 %q 中未設定 default_profile\n", path)
		fmt.Fprintf(stderr, "提示：執行 `byok config set-default --name <profile>` 或傳入 --profile\n")
		return nil, "", errExit
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
		return nil, "", errExit
	}

	// 4. Provider 驗證：此版本僅支援 openai。
	provider := profile.Provider
	if provider == "" {
		provider = "openai"
	}
	if provider != "openai" {
		fmt.Fprintf(stderr, "錯誤：profile %q 使用 provider %q；byok 首版僅支援 openai provider\n", profile.Name, provider)
		return nil, "", errExit
	}

	// 5. 確認可執行檔可在 PATH 中解析。
	resolved, err := exec.LookPath(binaryName)
	if err != nil {
		fmt.Fprintf(stderr, "錯誤：在 PATH 中找不到 %q 可執行檔\n", binaryName)
		fmt.Fprintf(stderr, "提示：%s\n", installHint)
		return nil, "", errExit
	}

	// 6. 解析 API 金鑰（keychain 優先、明碼 fallback）。
	apiKey, _, err := config.Resolver.Resolve(*profile)
	if err != nil {
		fmt.Fprintf(stderr, "錯誤：找不到 profile %q 的金鑰（keychain 與設定檔皆無）\n", profile.Name)
		fmt.Fprintf(stderr, "提示：執行 `byok config set-key %s` 將金鑰存入 keychain\n", profile.Name)
		return nil, "", errExit
	}
	profile.APIKey = apiKey

	return profile, resolved, nil
}

// availableProfileNames 回傳供錯誤訊息使用的設定檔名稱清單。
func availableProfileNames(profiles []config.Profile) []string {
	names := make([]string, 0, len(profiles))
	for _, p := range profiles {
		names = append(names, p.Name)
	}
	return names
}

// stdinTerminalCheck 可被測試替換，判斷給定 reader 是否為終端機。
// 預設為 isStdinTerminal（僅 *os.File 以 isTerminal 判定 fd）。
var stdinTerminalCheck = isStdinTerminal

// resolveModelForLaunch 依 spec「Launch Copilot with BYOK profile」的五條模型
// 解析規則，從 profile 的候選 models 清單與 --model 旗標決定要注入子程序
// 的單一模型字串。規則：
//  1. modelFlag 非空 → 一律使用 modelFlag，不顯示互動選單。
//  2. models 恰一個 → 直接使用該模型。
//  3. models 多個且 stdin 為終端機 → 顯示上下鍵互動選單，回傳使用者選取項。
//  4. models 多個但 stdin 非終端機 → 印出錯誤並回傳 errExit。
//  5. models 為空 → 印出錯誤（提示 byok config set-models）並回傳 errExit。
//
// stdin/stdout 用於互動選單；stderr 用於錯誤訊息。回傳的 model 字串將
// 傳入 runner 注入各 target 的模型環境變數，runner 不再自行回退 default_model。
func resolveModelForLaunch(profile *config.Profile, modelFlag string, stdin io.Reader, stdout, stderr io.Writer) (string, error) {
	if modelFlag != "" {
		return modelFlag, nil
	}
	switch len(profile.Models) {
	case 0:
		fmt.Fprintf(stderr, "錯誤：profile %q 未設定任何候選模型\n", profile.Name)
		fmt.Fprintf(stderr, "提示：執行 `byok config set-models %s --model <model>` 設定候選模型\n", profile.Name)
		return "", errExit
	case 1:
		return profile.Models[0], nil
	default:
		// 多個候選：需互動選單，故 stdin 必須為終端機。
		if !stdinTerminalCheck(stdin) {
			fmt.Fprintf(stderr, "錯誤：profile %q 有多個候選模型但 stdin 非終端機\n", profile.Name)
			fmt.Fprintf(stderr, "提示：請以 --model 旗標明確指定模型，或執行 `byok config set-models %s --model <model>` 縮減為單一候選\n", profile.Name)
			return "", errExit
		}
		selected, err := config.SelectModel(profile.Models, stdin, stdout, func(int) bool { return true })
		if err != nil {
			if errors.Is(err, config.ErrSelectionCancelled) {
				// 使用者主動取消（Ctrl-C/Esc）：簡潔提示後以非零結束碼退出。
				fmt.Fprintf(stderr, "已取消模型選擇。\n")
				return "", errExit
			}
			fmt.Fprintf(stderr, "錯誤：模型選擇失敗: %v\n", err)
			return "", errExit
		}
		return selected, nil
	}
}

// isStdinTerminal 判斷 r 是否為終端機檔案。非 *os.File 一律視為非終端機。
func isStdinTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return isTerminal(int(f.Fd()))
}

// buildExtraArgs 組合 yolo 旗標與透傳參數為 extraArgs。
// yoloLiteral 為目標工具特定的 yolo 旗標字串（copilot/codex 傳 "--yolo"，
// claude 傳 "--dangerously-skip-permissions"）。yolo 旗標在前，透傳參數
// 在後；兩者皆空時回傳 nil。
func buildExtraArgs(yolo bool, target string, args []string) []string {
	var extraArgs []string
	if yolo {
		yoloLiteral := "--yolo"
		if target == "claude" {
			yoloLiteral = "--dangerously-skip-permissions"
		}
		if target == "pi" {
			yoloLiteral = "--approve"
		}
		extraArgs = append(extraArgs, yoloLiteral)
	}
	extraArgs = append(extraArgs, args...)
	return extraArgs
}
