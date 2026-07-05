package cmd

import (
	"fmt"

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
	}
	root.AddCommand(newLaunchCmd())
	root.AddCommand(newConfigCmd())
	return root
}

// ErrExit 是一個 sentinel 錯誤，由 RunE 閉包回傳以強制 main 回傳非零
// 結束碼，而不需依賴 os.Exit（其會跳過 defer）。詳細錯誤訊息已由閉包
// 寫入 stderr，main 應對此 sentinel 略過再次印出。
var ErrExit = fmt.Errorf("byok: non-zero exit")

// errExit 為 ErrExit 的別名，供 cmd 套件內部使用。
var errExit = ErrExit