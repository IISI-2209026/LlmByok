//go:build windows

package updater

import (
	"syscall"
	"unsafe"
)

// Windows MoveFileEx 旗標與常數（避免引入 golang.org/x/sys 第三方相依）。
const movefileReplaceExisting = 0x00000001

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procMoveFileExW = kernel32.NewProc("MoveFileExW")
)

// renameReplace 在 Windows 上以 MoveFileExW + MOVEFILE_REPLACE_EXISTING 原子替換 dst。
// os.Rename 無法覆蓋使用中的執行檔，故直接呼叫 kernel32。
func renameReplace(src, dst string) error {
	srcPtr, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	dstPtr, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	r1, _, callErr := procMoveFileExW.Call(
		uintptr(unsafe.Pointer(srcPtr)),
		uintptr(unsafe.Pointer(dstPtr)),
		uintptr(movefileReplaceExisting),
	)
	if r1 == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return syscall.Errno(0)
	}
	return nil
}