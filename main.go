package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/IISI-2209026/LlmByok/cmd"
	"github.com/IISI-2209026/LlmByok/internal/version"
)

func main() {
	root := cmd.NewRoot(version.Version)
	if err := root.Execute(); err != nil {
		// ErrExit 表示 RunE 已自行將訊息寫入 stderr，不再重複印出。
		if !errors.Is(err, cmd.ErrExit) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}