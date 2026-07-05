package cmd

import (
	"fmt"

	"github.com/IISI-2209026/LlmByok/internal/version"
	"github.com/spf13/cobra"
)

// NewVersionCmd 建立 `byok version` 子指令，輸出當前版本號。
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "顯示 byok 版本號",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "byok version %s\n", version.Version)
		},
	}
}