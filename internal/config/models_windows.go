//go:build windows

package config

import (
	"golang.org/x/sys/windows"
)

// platformEnableVT 於 Windows console 啟用虛擬終端機（VT）處理，使 ANSI
// 反白與游標控制序列（如 \x1b[7m、\x1b[<n>A）正確渲染。讀取 stdout fd
// 的原 console mode，加上 ENABLE_VIRTUAL_TERMINAL_PROCESSING 後設回；
// 回傳的 restore 還原為原 mode。fd 非 console handle 時回傳錯誤（呼叫端忽略）。
func platformEnableVT(fd int) (restore func() error, err error) {
	h := windows.Handle(fd)
	var old uint32
	if err := windows.GetConsoleMode(h, &old); err != nil {
		return nil, err
	}
	newMode := old | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if err := windows.SetConsoleMode(h, newMode); err != nil {
		return nil, err
	}
	return func() error { return windows.SetConsoleMode(h, old) }, nil
}