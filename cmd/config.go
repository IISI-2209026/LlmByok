package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/IISI-2209026/LlmByok/internal/config"
	"github.com/IISI-2209026/LlmByok/internal/secret"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// keyStore 抽象 keychain 操作，便於測試注入 mock。
type keyStore interface {
	Store(profileName, apiKey string) error
	Delete(profileName string) error
}

// realKeyStore 呼叫 internal/secret 實作。
type realKeyStore struct{}

func (realKeyStore) Store(profileName, apiKey string) error { return secret.Store(profileName, apiKey) }
func (realKeyStore) Delete(profileName string) error        { return secret.Delete(profileName) }

// keys 是全域 keyStore，測試可替換。
var keys keyStore = realKeyStore{}

// isTerminal 偵測 stdin 是否為終端機，測試可替換。
var isTerminal = term.IsTerminal

// newConfigCmd 建置 `byok config` 父指令及其子指令。
func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "管理設定檔中的 BYOK profile",
	}
	c.AddCommand(newConfigAddCmd())
	c.AddCommand(newConfigUpdateCmd())
	c.AddCommand(newConfigListCmd())
	c.AddCommand(newConfigDeleteCmd())
	c.AddCommand(newConfigSetDefaultCmd())
	return c
}

// --config 旗標由所有 config 子指令共用。
func addConfigFlag(c *cobra.Command, p *string) {
	c.Flags().StringVar(p, "config", "", "設定檔路徑（預設為 ~/.byok/config.yaml）")
}

// stdinIsTerminal 判斷 cmd 的 stdin 是否為終端機。
func stdinIsTerminal(cmd *cobra.Command) bool {
	f, ok := cmd.InOrStdin().(*os.File)
	if !ok {
		return false
	}
	return isTerminal(int(f.Fd()))
}

func newConfigAddCmd() *cobra.Command {
	var name, provider, apiBase, apiKey, defaultModel, keyStorage, cfgPath string
	c := &cobra.Command{
		Use:   "add",
		Short: "新增 BYOK profile 至設定檔",
		Long: `新增一個 profile 至設定檔。若檔案不存在則會建立。
當未設定預設 profile 時，新 profile 會成為預設值。
若同名 profile 已存在，則回傳錯誤且不修改檔案。

未提供任何欄位旗標時進入互動模式（需 TTY）。
金鑰預設存入 OS keychain，可用 --key-storage plaintext 改存明碼。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := cmd.Flags()
			interactive := !fs.Changed("name") &&
				!fs.Changed("provider") &&
				!fs.Changed("api-base") &&
				!fs.Changed("default-model") &&
				!fs.Changed("api-key")
			if interactive {
				if !stdinIsTerminal(cmd) {
					return fmt.Errorf("互動模式需要終端機；請改用參數模式（--name, --api-base, --default-model 等）")
				}
				p := &config.Prompter{In: cmd.InOrStdin(), Out: cmd.OutOrStdout(), IsTTY: term.IsTerminal}
				return runConfigAddInteractive(cfgPath, keyStorage, p, cmd.OutOrStdout())
			}
			if name == "" {
				return fmt.Errorf("--name 為必填")
			}
			if apiBase == "" {
				return fmt.Errorf("--api-base 為必填")
			}
			if defaultModel == "" {
				return fmt.Errorf("--default-model 為必填")
			}
			return runConfigAdd(cfgPath, name, provider, apiBase, apiKey, defaultModel, keyStorage, cmd.OutOrStdout())
		},
		SilenceUsage: false,
	}
	c.Flags().StringVar(&name, "name", "", "profile 名稱")
	c.Flags().StringVar(&provider, "provider", "openai", "provider 類型（首版僅支援 openai）")
	c.Flags().StringVar(&apiBase, "api-base", "", "API base URL")
	c.Flags().StringVar(&apiKey, "api-key", "", "API key（無驗證端點可留空）")
	c.Flags().StringVar(&defaultModel, "default-model", "", "預設模型識別碼")
	c.Flags().StringVar(&keyStorage, "key-storage", "keychain", "金鑰儲存位置（keychain|plaintext）")
	addConfigFlag(c, &cfgPath)
	return c
}

func newConfigUpdateCmd() *cobra.Command {
	var name, provider, apiBase, apiKey, defaultModel, keyStorage, cfgPath string
	c := &cobra.Command{
		Use:   "update",
		Short: "更新既有 BYOK profile",
		Long: `更新既有 profile 的欄位。未提供的欄位保留原值。
僅提供 --name 時進入互動模式（需 TTY）。
提供 --api-key 時依 --key-storage 處理金鑰。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := cmd.Flags()
			interactive := !fs.Changed("provider") &&
				!fs.Changed("api-base") &&
				!fs.Changed("default-model") &&
				!fs.Changed("api-key") &&
				!fs.Changed("key-storage")
			if interactive {
				if !stdinIsTerminal(cmd) {
					return fmt.Errorf("互動模式需要終端機；請改用參數模式（--provider, --api-base 等）")
				}
				p := &config.Prompter{In: cmd.InOrStdin(), Out: cmd.OutOrStdout(), IsTTY: term.IsTerminal}
				return runConfigUpdateInteractive(cfgPath, name, keyStorage, p, cmd.OutOrStdout())
			}
			var pProvider, pAPIBase, pDefaultModel *string
			if fs.Changed("provider") {
				pProvider = &provider
			}
			if fs.Changed("api-base") {
				pAPIBase = &apiBase
			}
			if fs.Changed("default-model") {
				pDefaultModel = &defaultModel
			}
			return runConfigUpdate(cfgPath, name, pProvider, pAPIBase, pDefaultModel, apiKey, fs.Changed("api-key"), keyStorage, cmd.OutOrStdout())
		},
		SilenceUsage: false,
	}
	c.Flags().StringVar(&name, "name", "", "要更新的 profile 名稱（必填）")
	c.Flags().StringVar(&provider, "provider", "openai", "provider 類型")
	c.Flags().StringVar(&apiBase, "api-base", "", "API base URL")
	c.Flags().StringVar(&apiKey, "api-key", "", "API key（設為空字串清除金鑰）")
	c.Flags().StringVar(&defaultModel, "default-model", "", "預設模型識別碼")
	c.Flags().StringVar(&keyStorage, "key-storage", "keychain", "金鑰儲存位置（keychain|plaintext）")
	addConfigFlag(c, &cfgPath)
	_ = c.MarkFlagRequired("name")
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

func newConfigDeleteCmd() *cobra.Command {
	var name, cfgPath string
	c := &cobra.Command{
		Use:   "delete",
		Short: "依名稱刪除 BYOK profile（同步清理 keychain）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigDelete(cfgPath, name, cmd.OutOrStdout())
		},
	}
	c.Flags().StringVar(&name, "name", "", "要刪除的 profile 名稱（必填）")
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

// persistKey 依 keyStorage 處理金鑰儲存。
// keychain: secret.Store，成功後清空 profile.APIKey。
// plaintext: best-effort 刪除 keychain 條目，設 profile.APIKey = apiKey。
// apiKey 為空時不做任何事（適用於 add 的「無金鑰」情境）。
func persistKey(p *config.Profile, apiKey, keyStorage string) error {
	if apiKey == "" {
		return nil
	}
	switch keyStorage {
	case "plaintext":
		_ = keys.Delete(p.Name)
		p.APIKey = apiKey
	default: // "keychain"
		if err := keys.Store(p.Name, apiKey); err != nil {
			return fmt.Errorf("儲存至 keychain: %w（可改用 --key-storage plaintext）", err)
		}
		p.APIKey = ""
	}
	return nil
}

func runConfigAdd(cfgPath, name, provider, apiBase, apiKey, defaultModel, keyStorage string, w io.Writer) error {
	path, err := configPath(cfgPath)
	if err != nil {
		return fmt.Errorf("解析設定檔路徑: %w", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
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
	prof := config.Profile{
		Name:         name,
		Provider:     provider,
		APIBase:      apiBase,
		DefaultModel: defaultModel,
	}
	if err := persistKey(&prof, apiKey, keyStorage); err != nil {
		return err
	}
	cfg.Profiles = append(cfg.Profiles, prof)
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

func runConfigAddInteractive(cfgPath, keyStorageDefault string, p *config.Prompter, w io.Writer) error {
	name, err := p.PromptString("profile 名稱")
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("profile 名稱不可為空")
	}
	provider, err := p.PromptDefault("provider", "openai")
	if err != nil {
		return err
	}
	apiBase, err := p.PromptString("API base URL")
	if err != nil {
		return err
	}
	if apiBase == "" {
		return fmt.Errorf("API base URL 不可為空")
	}
	defaultModel, err := p.PromptString("預設模型")
	if err != nil {
		return err
	}
	if defaultModel == "" {
		return fmt.Errorf("預設模型不可為空")
	}
	apiKey, err := p.PromptSecret("API key（可留空）")
	if err != nil {
		return err
	}
	keyStorage, err := p.PromptChoice("金鑰儲存", []string{"keychain", "plaintext"}, keyStorageDefault)
	if err != nil {
		return err
	}
	return runConfigAdd(cfgPath, name, provider, apiBase, apiKey, defaultModel, keyStorage, w)
}

func runConfigUpdate(cfgPath, name string, provider, apiBase, defaultModel *string, apiKey string, apiKeyProvided bool, keyStorage string, w io.Writer) error {
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
		return fmt.Errorf("找不到 profile %q", name)
	}
	prof := &cfg.Profiles[idx]
	config.ApplyProfileUpdates(prof, provider, apiBase, defaultModel)
	if apiKeyProvided {
		if apiKey == "" {
			_ = keys.Delete(prof.Name)
			prof.APIKey = ""
		} else {
			if err := persistKey(prof, apiKey, keyStorage); err != nil {
				return err
			}
		}
	}
	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("儲存設定檔: %w", err)
	}
	fmt.Fprintf(w, "已更新 profile %q\n", name)
	return nil
}

func runConfigUpdateInteractive(cfgPath, name, keyStorageDefault string, p *config.Prompter, w io.Writer) error {
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
		return fmt.Errorf("找不到 profile %q", name)
	}
	prof := &cfg.Profiles[idx]
	provider, err := p.PromptDefault("provider", prof.Provider)
	if err != nil {
		return err
	}
	apiBase, err := p.PromptDefault("API base URL", prof.APIBase)
	if err != nil {
		return err
	}
	defaultModel, err := p.PromptDefault("預設模型", prof.DefaultModel)
	if err != nil {
		return err
	}
	apiKey, err := p.PromptSecret("API key（留空保留原值）")
	if err != nil {
		return err
	}
	keyStorage, err := p.PromptChoice("金鑰儲存", []string{"keychain", "plaintext"}, keyStorageDefault)
	if err != nil {
		return err
	}
	prof.Provider = provider
	prof.APIBase = apiBase
	prof.DefaultModel = defaultModel
	if apiKey != "" {
		if err := persistKey(prof, apiKey, keyStorage); err != nil {
			return err
		}
	}
	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("儲存設定檔: %w", err)
	}
	fmt.Fprintf(w, "已更新 profile %q\n", name)
	return nil
}

func runConfigList(cfgPath string, w io.Writer) error {
	path, err := configPath(cfgPath)
	if err != nil {
		return fmt.Errorf("解析設定檔路徑: %w", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
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

func runConfigDelete(cfgPath, name string, w io.Writer) error {
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
	fmt.Fprintf(w, "已從 %s 刪除 profile %q\n", path, name)
	if err := keys.Delete(name); err != nil {
		if !errors.Is(err, secret.ErrNotFound) {
			fmt.Fprintf(w, "警告：清理 keychain 失敗: %v\n", err)
		}
	} else {
		fmt.Fprintf(w, "已自 keychain 刪除金鑰（profile: %s）\n", name)
	}
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
func isNotExistMsg(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, os.ErrNotExist)
}