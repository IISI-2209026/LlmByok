package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
	"github.com/IISI-2209026/LlmByok/internal/secret"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// newConfigCmd 建置 `byok config` 父指令及其子指令。
func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "管理設定檔中的 BYOK profile",
	}
	c.AddCommand(newConfigAddCmd())
	c.AddCommand(newConfigListCmd())
	c.AddCommand(newConfigRemoveCmd())
	c.AddCommand(newConfigSetDefaultCmd())
	c.AddCommand(newConfigSetKeyCmd())
	c.AddCommand(newConfigDelKeyCmd())
	c.AddCommand(newConfigImportKeysCmd())
	return c
}

// --config 旗標由所有 config 子指令共用。
func addConfigFlag(c *cobra.Command, p *string) {
	c.Flags().StringVar(p, "config", "", "設定檔路徑（預設為 ~/.byok/config.yaml）")
}

func newConfigAddCmd() *cobra.Command {
	var name, provider, apiBase, apiKey, defaultModel, cfgPath string
	c := &cobra.Command{
		Use:   "add",
		Short: "新增 BYOK profile 至設定檔",
		Long: `新增一個 profile 至設定檔。若檔案不存在則會建立。
當未設定預設 profile 時，新 profile 會成為預設值。
若同名 profile 已存在，則回傳錯誤且不修改檔案。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigAdd(cfgPath, name, provider, apiBase, apiKey, defaultModel, cmd.OutOrStdout())
		},
		SilenceUsage: false,
	}
	c.Flags().StringVar(&name, "name", "", "profile 名稱（必填）")
	c.Flags().StringVar(&provider, "provider", "openai", "provider 類型（首版僅支援 openai）")
	c.Flags().StringVar(&apiBase, "api-base", "", "API base URL（必填）")
	c.Flags().StringVar(&apiKey, "api-key", "", "API key（無驗證端點可留空）")
	c.Flags().StringVar(&defaultModel, "default-model", "", "預設模型識別碼（必填）")
	addConfigFlag(c, &cfgPath)
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("api-base")
	_ = c.MarkFlagRequired("default-model")
	return c
}

func newConfigListCmd() *cobra.Command {
	var cfgPath string
	c := &cobra.Command{
		Use:   "list",
		Short: "列出所有 BYOK profile（API key 已遮罩）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList(cfgPath, cmd.OutOrStdout())
		},
	}
	addConfigFlag(c, &cfgPath)
	return c
}

func newConfigRemoveCmd() *cobra.Command {
	var name, cfgPath string
	c := &cobra.Command{
		Use:   "remove",
		Short: "依名稱移除 BYOK profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigRemove(cfgPath, name, cmd.OutOrStdout())
		},
	}
	c.Flags().StringVar(&name, "name", "", "要移除的 profile 名稱（必填）")
	addConfigFlag(c, &cfgPath)
	_ = c.MarkFlagRequired("name")
	return c
}

func newConfigSetDefaultCmd() *cobra.Command {
	var name, cfgPath string
	c := &cobra.Command{
		Use:   "set-default",
		Short: "設定預設 BYOK profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSetDefault(cfgPath, name, cmd.OutOrStdout())
		},
	}
	c.Flags().StringVar(&name, "name", "", "要設為預設的 profile 名稱（必填）")
	addConfigFlag(c, &cfgPath)
	_ = c.MarkFlagRequired("name")
	return c
}

// --- 執行器 ---

func configPath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	return config.DefaultConfigPath()
}

func runConfigAdd(cfgPath, name, provider, apiBase, apiKey, defaultModel string, w io.Writer) error {
	path, err := configPath(cfgPath)
	if err != nil {
		return fmt.Errorf("解析設定檔路徑: %w", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		// 對 add 而言檔案不存在是允許的：從空設定檔開始。
		if os.IsNotExist(err) || isNotExistMsg(err) {
			cfg = &config.Config{}
		} else {
			return err
		}
	}
	for _, p := range cfg.Profiles {
		if p.Name == name {
			return fmt.Errorf("profile %q 已存在；未修改設定檔", name)
		}
	}
	cfg.Profiles = append(cfg.Profiles, config.Profile{
		Name:         name,
		Provider:     provider,
		APIBase:      apiBase,
		APIKey:       apiKey,
		DefaultModel: defaultModel,
	})
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = name
	}
	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("儲存設定檔: %w", err)
	}
	fmt.Fprintf(w, "已新增 profile %q 至 %s\n", name, path)
	if cfg.DefaultProfile == name {
		fmt.Fprintf(w, "已將 %q 設為預設 profile\n", name)
	}
	return nil
}

func runConfigList(cfgPath string, w io.Writer) error {
	path, err := configPath(cfgPath)
	if err != nil {
		return fmt.Errorf("解析設定檔路徑: %w", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		// 設定檔不存在視為尚無 profile，以友善訊息提示，不傾印原始 OS 錯誤。
		if isNotExistMsg(err) {
			fmt.Fprintf(w, "尚無任何 profile。請先執行 `byok config add`。\n")
			return nil
		}
		return err
	}
	if len(cfg.Profiles) == 0 {
		fmt.Fprintf(w, "尚無任何 profile。請先執行 `byok config add`。\n")
		return nil
	}
	fmt.Fprintf(w, "%-20s %-10s %-35s %-20s %-10s %s\n", "名稱", "Provider", "API Base", "預設模型", "來源", "API Key")
	for _, p := range cfg.Profiles {
		marker := ""
		if p.Name == cfg.DefaultProfile {
			marker = " (預設)"
		}
		key, source, _ := config.Resolver.Resolve(p)
		fmt.Fprintf(w, "%-20s %-10s %-35s %-20s %-10s %s%s\n",
			p.Name, p.Provider, p.APIBase, p.DefaultModel, source.String(), maskAPIKey(key), marker)
	}
	return nil
}

func runConfigRemove(cfgPath, name string, w io.Writer) error {
	path, err := configPath(cfgPath)
	if err != nil {
		return fmt.Errorf("解析設定檔路徑: %w", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		if isNotExistMsg(err) {
			return fmt.Errorf("設定檔不存在: %s", path)
		}
		return err
	}
	idx := -1
	for i, p := range cfg.Profiles {
		if p.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("找不到 profile %q；未修改設定檔", name)
	}
	cfg.Profiles = append(cfg.Profiles[:idx], cfg.Profiles[idx+1:]...)
	if cfg.DefaultProfile == name {
		cfg.DefaultProfile = ""
	}
	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("儲存設定檔: %w", err)
	}
	fmt.Fprintf(w, "已從 %s 移除 profile %q\n", path, name)
	return nil
}

func runConfigSetDefault(cfgPath, name string, w io.Writer) error {
	path, err := configPath(cfgPath)
	if err != nil {
		return fmt.Errorf("解析設定檔路徑: %w", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		if isNotExistMsg(err) {
			return fmt.Errorf("設定檔不存在: %s", path)
		}
		return err
	}
	found := false
	for _, p := range cfg.Profiles {
		if p.Name == name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("找不到 profile %q", name)
	}
	cfg.DefaultProfile = name
	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("儲存設定檔: %w", err)
	}
	fmt.Fprintf(w, "已將預設 profile 設為 %q\n", name)
	return nil
}

// maskAPIKey 顯示前 4 個與後 4 個字元，中間以 "..." 連接。
// 空字串回傳空字串。少於 4 個字元的金鑰原樣回傳（無需遮罩）。
// 否則回傳前 4 + "..." + 後 4，即使兩個視窗重疊亦然
// （依規格範例："sk-1234" -> "sk-1...1234"）。
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	n := len(key)
	if n < 4 {
		return key
	}
	return key[:4] + "..." + key[n-4:]
}

// isNotExistMsg 偵測 config.Load 回傳的「設定檔不存在」錯誤。
// 我們使用 errors.Is(err, os.ErrNotExist) 而非 os.IsNotExist，因為
// os.IsNotExist 在 Windows 上不會遍歷被包裝的錯誤（它以 == 比對
// ERROR_FILE_NOT_FOUND / ERROR_PATH_NOT_FOUND，而非使用 errors.Is），
// 而 config.Load 會以 fmt.Errorf("...: %w") 包裝底層 os.PathError。
func isNotExistMsg(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, os.ErrNotExist)
}

// --- 金鑰管理子指令 ---

func newConfigSetKeyCmd() *cobra.Command {
	var cfgPath string
	c := &cobra.Command{
		Use:   "set-key <profile>",
		Short: "將 API 金鑰存入 OS keychain（互動式輸入）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := args[0]
			fmt.Fprint(cmd.ErrOrStderr(), "輸入金鑰: ")
			apiKey, err := readPassword(cmd.InOrStdin())
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "錯誤：讀取金鑰失敗: %v\n", err)
				return errExit
			}
			if apiKey == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "金鑰不可為空")
				return errExit
			}
			if err := runConfigSetKey(cfgPath, profileName, apiKey, cmd.OutOrStdout()); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "錯誤：%v\n", err)
				return errExit
			}
			return nil
		},
		SilenceUsage: true,
	}
	addConfigFlag(c, &cfgPath)
	return c
}

func newConfigDelKeyCmd() *cobra.Command {
	var cfgPath string
	c := &cobra.Command{
		Use:   "del-key <profile>",
		Short: "自 OS keychain 刪除 API 金鑰",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := args[0]
			if err := runConfigDelKey(cfgPath, profileName, cmd.OutOrStdout()); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "錯誤：%v\n", err)
				return errExit
			}
			return nil
		},
		SilenceUsage: true,
	}
	addConfigFlag(c, &cfgPath)
	return c
}

func newConfigImportKeysCmd() *cobra.Command {
	var cfgPath string
	c := &cobra.Command{
		Use:   "import-keys",
		Short: "將設定檔中所有明碼金鑰批次匯入 OS keychain",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runConfigImportKeys(cfgPath, cmd.OutOrStdout()); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "錯誤：%v\n", err)
				return errExit
			}
			return nil
		},
		SilenceUsage: true,
	}
	addConfigFlag(c, &cfgPath)
	return c
}

// readPassword 自 stdin 讀取一行密碼。若 stdin 為終端機則使用
// term.ReadPassword 不回顯讀取；否則以 bufio 讀取一行（支援管線輸入）。
func readPassword(stdin io.Reader) (string, error) {
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		bytes, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}
	reader := bufio.NewReader(stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// runConfigSetKey 將 apiKey 存入 keychain 並清除設定檔中的明碼 api_key。
func runConfigSetKey(cfgPath, profileName, apiKey string, w io.Writer) error {
	if apiKey == "" {
		return fmt.Errorf("金鑰不可為空")
	}
	path, err := configPath(cfgPath)
	if err != nil {
		return fmt.Errorf("解析設定檔路徑: %w", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		if isNotExistMsg(err) {
			return fmt.Errorf("profile %s 不存在", profileName)
		}
		return err
	}
	found := false
	for i := range cfg.Profiles {
		if cfg.Profiles[i].Name == profileName {
			found = true
			if err := secret.Store(profileName, apiKey); err != nil {
				return fmt.Errorf("儲存至 keychain: %w", err)
			}
			hadPlaintext := cfg.Profiles[i].APIKey != ""
			cfg.Profiles[i].APIKey = ""
			if err := config.Save(path, cfg); err != nil {
				return fmt.Errorf("儲存設定檔: %w", err)
			}
			fmt.Fprintf(w, "已將金鑰存入 keychain（profile: %s）\n", profileName)
			if hadPlaintext {
				fmt.Fprintf(w, "已清除設定檔中的明碼 api_key\n")
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("profile %s 不存在", profileName)
	}
	return nil
}

// runConfigDelKey 自 keychain 刪除指定 profile 的金鑰。
func runConfigDelKey(cfgPath, profileName string, w io.Writer) error {
	path, err := configPath(cfgPath)
	if err != nil {
		return fmt.Errorf("解析設定檔路徑: %w", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		if isNotExistMsg(err) {
			return fmt.Errorf("profile %s 不存在", profileName)
		}
		return err
	}
	found := false
	for _, p := range cfg.Profiles {
		if p.Name == profileName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("profile %s 不存在", profileName)
	}
	if err := secret.Delete(profileName); err != nil {
		if errors.Is(err, secret.ErrNotFound) {
			return fmt.Errorf("profile %s 未在 keychain 中", profileName)
		}
		return fmt.Errorf("自 keychain 刪除: %w", err)
	}
	fmt.Fprintf(w, "已自 keychain 刪除金鑰（profile: %s）\n", profileName)
	return nil
}

// runConfigImportKeys 批次將設定檔中的明碼金鑰匯入 keychain，成功後清除明碼。
func runConfigImportKeys(cfgPath string, w io.Writer) error {
	path, err := configPath(cfgPath)
	if err != nil {
		return fmt.Errorf("解析設定檔路徑: %w", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		if isNotExistMsg(err) {
			return fmt.Errorf("設定檔不存在: %s", path)
		}
		return err
	}
	var toImport []int
	for i := range cfg.Profiles {
		if cfg.Profiles[i].APIKey != "" {
			toImport = append(toImport, i)
		}
	}
	if len(toImport) == 0 {
		fmt.Fprintln(w, "設定檔中無明碼金鑰可匯入")
		return nil
	}
	var failures []string
	imported := 0
	for _, i := range toImport {
		if err := secret.Store(cfg.Profiles[i].Name, cfg.Profiles[i].APIKey); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", cfg.Profiles[i].Name, err))
			continue
		}
		cfg.Profiles[i].APIKey = ""
		imported++
	}
	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("儲存設定檔: %w", err)
	}
	if len(failures) > 0 {
		fmt.Fprintln(w, "匯入失敗:")
		for _, f := range failures {
			fmt.Fprintf(w, "  %s\n", f)
		}
		return fmt.Errorf("匯入部分失敗（%d 成功，%d 失敗）", imported, len(failures))
	}
	fmt.Fprintf(w, "匯入 %d 個金鑰至 keychain\n", imported)
	return nil
}