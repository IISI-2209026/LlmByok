package cmd

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/IISI-2209026/LlmByok/internal/updater"
	"github.com/spf13/cobra"
)

// fetcher 為 update 指令與啟動檢查所需的可注入介面，便於測試以 stub 替換。
type fetcher interface {
	Channel(v string) string
	LatestRelease(ctx context.Context, channel string) (updater.Release, error)
	IsNewer(current, latest string) (bool, error)
	DownloadAndReplace(ctx context.Context, rel updater.Release, goos, goarch string) error
}

// realFetcher 以 internal/updater 套件函式實作 fetcher。
type realFetcher struct{}

func (realFetcher) Channel(v string) string { return updater.Channel(v) }

func (realFetcher) LatestRelease(ctx context.Context, channel string) (updater.Release, error) {
	return updater.LatestRelease(ctx, channel)
}

func (realFetcher) IsNewer(current, latest string) (bool, error) {
	return updater.IsNewer(current, latest)
}

func (realFetcher) DownloadAndReplace(ctx context.Context, rel updater.Release, goos, goarch string) error {
	return updater.DownloadAndReplace(ctx, rel, goos, goarch)
}

// defaultFetcher 為 update 指令與啟動檢查使用的預設 fetcher；測試可覆寫。
var defaultFetcher fetcher = realFetcher{}

// newUpdateCmd 建置 `byok update` 子指令。
func newUpdateCmd(version string) *cobra.Command {
	var check bool
	var channelFlag string
	c := &cobra.Command{
		Use:   "update",
		Short: "檢查並自我更新 byok 至最新 GitHub Release",
		Long: `依當前版本所屬 channel（含 -dev. 為 dev channel，否則 stable channel）
查詢 GitHub Releases，下載對應平台資產並替換當前執行檔。

使用 --channel 覆寫自動 channel 判定，可跨 channel 更新；
使用 --check 只查詢不替換。`,
		Example: `  byok update
  byok update --check
  byok update --channel prerelease
  byok update --channel release --check`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(defaultFetcher, version, check, channelFlag,
				runtime.GOOS, runtime.GOARCH,
				cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
		SilenceUsage: true,
	}
	c.Flags().BoolVar(&check, "check", false, "只檢查最新版本，不下載或替換執行檔")
	c.Flags().StringVar(&channelFlag, "channel", "", "覆寫 channel 判定（prerelease|release）")
	return c
}

// resolveChannel 依版本與 --channel 旗標決定查詢的 channel。
// channelFlag 為空時自動判定；不合法值回傳錯誤。
func resolveChannel(version, channelFlag string) (string, error) {
	if channelFlag == "" {
		return updater.Channel(version), nil
	}
	switch channelFlag {
	case "prerelease":
		return "dev", nil
	case "release":
		return "stable", nil
	default:
		return "", fmt.Errorf("不合法的 channel 值 %q（接受 prerelease 或 release）", channelFlag)
	}
}

// runUpdate 實作 `byok update` 的核心流程。
func runUpdate(f fetcher, version string, check bool, channelFlag, goos, goarch string, stdout, stderr io.Writer) error {
	channel, err := resolveChannel(version, channelFlag)
	if err != nil {
		fmt.Fprintf(stderr, "錯誤：%v\n", err)
		return errExit
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rel, err := f.LatestRelease(ctx, channel)
	if err != nil {
		fmt.Fprintf(stderr, "錯誤：查詢最新版本失敗: %v\n", err)
		return errExit
	}
	newer, err := f.IsNewer(version, rel.Version)
	if err != nil {
		fmt.Fprintf(stderr, "錯誤：版本比較失敗: %v\n", err)
		return errExit
	}
	if !newer {
		fmt.Fprintf(stdout, "已是最新版本 (%s)\n", version)
		return nil
	}
	if check {
		fmt.Fprintf(stdout, "最新版本：%s（目前：%s）\n", rel.Version, version)
		return nil
	}
	fmt.Fprintf(stdout, "更新中：%s → %s\n", version, rel.Version)
	if err := f.DownloadAndReplace(ctx, rel, goos, goarch); err != nil {
		fmt.Fprintf(stderr, "錯誤：更新失敗: %v\n", err)
		return errExit
	}
	fmt.Fprintf(stdout, "已更新至 %s，請重新執行 byok\n", rel.Version)
	return nil
}