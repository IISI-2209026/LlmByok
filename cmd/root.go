package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/IISI-2209026/LlmByok/internal/updater"
	"github.com/spf13/cobra"
)

// NewRoot 建立根 `byok` 指令並掛載子指令。
func NewRoot(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "byok",
		Short: "Copilot CLI 的 BYOK（自帶金鑰）啟動器",
		Long: `byok 暫時將 BYOK（Bring Your Own Key）環境變數注入 Copilot CLI
子程序，設定來自 ~/.byok/config.yaml 的 YAML 設定檔。父程序
byok 與您的 shell 環境永不被修改，因此日常 Copilot 使用不受影響。

首版僅支援 Copilot CLI 與 OpenAI 相容端點
（provider 類型 "openai"）。`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			runStartupUpdateCheck(cmd, version)
			return nil
		},
	}
	root.AddCommand(newLaunchCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newUpdateCmd(version))
	return root
}

// runStartupUpdateCheck 於非 launch/update 子指令完成後，查詢當前 channel
// 最新 release；較新時在 stderr 印一行提示。launch 與 update 跳過；
// BYOK_NO_UPDATE_CHECK=1 跳過。任何錯誤靜默不影響 exit code/stdout。
func runStartupUpdateCheck(cmd *cobra.Command, version string) {
	if name := cmd.Name(); name == "launch" || name == "update" {
		return
	}
	if os.Getenv("BYOK_NO_UPDATE_CHECK") == "1" {
		return
	}
	channel := updater.Channel(version)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rel, err := defaultFetcher.LatestRelease(ctx, channel)
	if err != nil {
		return
	}
	newer, err := defaultFetcher.IsNewer(version, rel.Version)
	if err != nil || !newer {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "新版本可用：%s（目前：%s）；執行 `byok update` 更新\n", rel.Version, version)
}

// ErrExit 是一個 sentinel 錯誤，由 RunE 閉包回傳以強制 main 回傳非零
// 結束碼，而不需依賴 os.Exit（其會跳過 defer）。詳細錯誤訊息已由閉包
// 寫入 stderr，main 應對此 sentinel 略過再次印出。
var ErrExit = fmt.Errorf("byok: non-zero exit")

// errExit 為 ErrExit 的別名，供 cmd 套件內部使用。
var errExit = ErrExit