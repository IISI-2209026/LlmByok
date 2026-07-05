package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/IISI-2209026/LlmByok/cmd"
)

// Version 是 byok 二進位檔版本，可在 link 時覆寫。
var Version = "dev"

func main() {
	if err := cmd.NewRoot(Version).Execute(); err != nil {
		// ErrExit 表示 RunE 已自行將訊息寫入 stderr，不再重複印出。
		if !errors.Is(err, cmd.ErrExit) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}