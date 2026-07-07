package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
	"github.com/spf13/cobra"
)

// newSetModelsCmd 建置 `byok config set-models <profile name>` 子指令（掛於 config 之下）。
// profile 名稱為第一位置參數；可重複的 --model 旗標整批覆寫該 profile
// 的候選模型清單。未提供任何 --model 且 stdin 為終端機時進入互動模式，
// 逐行輸入模型識別碼直至空行。
func newSetModelsCmd() *cobra.Command {
	var models []string
	var cfgPath string
	c := &cobra.Command{
		Use:   "set-models <profile name>",
		Short: "設定 profile 的候選模型清單",
		Long: `設定指定 profile 的候選模型清單，整批覆寫原有清單。
profile 名稱為第一位置參數。可重複使用 --model <model> 指定多個候選模型；
未提供 --model 且 stdin 為終端機時進入互動模式，逐行輸入模型直至空行結束。
profile 不存在時回傳錯誤且不修改檔案。

此指令為 ` + "`byok config`" + ` 的子指令，用法：byok config set-models <profile name>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("必須提供 profile 名稱作為位置參數")
			}
			name := args[0]
			fs := cmd.Flags()
			if !fs.Changed("model") {
				if !stdinIsTerminal(cmd) {
					return fmt.Errorf("互動模式需要終端機；請改用 --model 旗標指定模型")
				}
				collected, err := promptModels(cmd.InOrStdin(), cmd.OutOrStdout())
				if err != nil {
					return err
				}
				models = collected
			}
			return runSetModels(cfgPath, name, models, cmd.OutOrStdout())
		},
		SilenceUsage: false,
	}
	c.Flags().StringArrayVar(&models, "model", nil, "候選模型識別碼（可重複）")
	addConfigFlag(c, &cfgPath)
	return c
}

// promptModels 逐行提示輸入模型識別碼，直至空行結束，回傳收集到的
// 非空項目。供 set-models 互動模式使用。
func promptModels(in io.Reader, out io.Writer) ([]string, error) {
	fmt.Fprintln(out, "逐行輸入候選模型識別碼，輸入空行結束：")
	r := bufio.NewReader(in)
	var models []string
	for {
		fmt.Fprint(out, "模型: ")
		line, err := r.ReadString('\n')
		if err != nil && line == "" {
			if len(models) == 0 {
				// 未輸入任何模型即 EOF：視為空清單，交由 runSetModels 報錯。
				return nil, nil
			}
			break
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}
		models = append(models, trimmed)
	}
	return models, nil
}

// runSetModels 將 models 整批寫入為指定 profile 的候選模型清單（覆寫）。
// 空清單 → 錯誤退出 1 且不修改檔案；profile 不存在 → 列出可用 profile
// 並錯誤退出 1。
func runSetModels(cfgPath, name string, models []string, w io.Writer) error {
	if len(models) == 0 {
		return fmt.Errorf("至少需要一個候選模型")
	}
	path, err := configPath(cfgPath)
	if err != nil {
		return fmt.Errorf("解析設定檔路徑: %w", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		if isNotExistMsg(err) {
			return fmt.Errorf("找不到 profile %q", name)
		}
		return err
	}
	idx := -1
	for i := range cfg.Profiles {
		if cfg.Profiles[i].Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		names := availableProfileNames(cfg.Profiles)
		if len(names) > 0 {
			return fmt.Errorf("找不到 profile %q；可用 profile: %s", name, strings.Join(names, ", "))
		}
		return fmt.Errorf("找不到 profile %q", name)
	}
	cfg.Profiles[idx].Models = append([]string(nil), models...)
	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("儲存設定檔: %w", err)
	}
	fmt.Fprintf(w, "已將 profile %q 的候選模型設為：%s\n", name, strings.Join(models, ", "))
	return nil
}