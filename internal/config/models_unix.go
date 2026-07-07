//go:build !windows

package config

// platformEnableVT 於非 Windows 平台為 no-op：現代 Unix 終端機（含 macOS
// Terminal.app、Linux gnome-terminal 等）原生支援 ANSI 反白與游標控制序列，
// 無需額外啟用。回傳的 restore 為 no-op。
func platformEnableVT(fd int) (restore func() error, err error) {
	return func() error { return nil }, nil
}