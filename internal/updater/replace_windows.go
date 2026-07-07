//go:build windows

package updater

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Windows MoveFileEx 旗標與常數（避免引入 golang.org/x/sys 第三方相依）。
const (
	movefileReplaceExisting  = 0x00000001
	movefileDelayUntilReboot = 0x00000004
)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procMoveFileExW = kernel32.NewProc("MoveFileExW")
)

// moveFileExW 呼叫 Windows MoveFileExW API。src 或 dst 為空字串時傳入 nil 指標。
func moveFileExW(src, dst string, flags uint32) error {
	var srcPtr, dstPtr *uint16
	if src != "" {
		var err error
		srcPtr, err = syscall.UTF16PtrFromString(src)
		if err != nil {
			return err
		}
	}
	if dst != "" {
		var err error
		dstPtr, err = syscall.UTF16PtrFromString(dst)
		if err != nil {
			return err
		}
	}
	r1, _, callErr := procMoveFileExW.Call(
		uintptr(unsafe.Pointer(srcPtr)),
		uintptr(unsafe.Pointer(dstPtr)),
		uintptr(flags),
	)
	if r1 == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return syscall.Errno(0)
	}
	return nil
}

// renameReplace 在 Windows 上以 rename-then-move 模式替換 dst。
// Windows 不允許覆寫執行中的執行檔，但允許重新命名，故先將目標重新命名為備份再移入新檔案。
func renameReplace(src, dst string) error {
	backup := dst + ".old"

	// Step 1: 處理既存 .old 檔案（上一輪更新殘留）。
	// .old 非執行中檔案，os.Remove 通常即可刪除；若失敗嘗試以 MoveFileExW + REPLACE_EXISTING 覆寫。
	if _, err := os.Stat(backup); err == nil {
		if err := os.Remove(backup); err != nil {
			tmp, cerr := os.CreateTemp(filepath.Dir(dst), ".byok-old-*")
			if cerr != nil {
				return err
			}
			tmpPath := tmp.Name()
			tmp.Close()
			if merr := moveFileExW(tmpPath, backup, movefileReplaceExisting); merr != nil {
				os.Remove(tmpPath)
				return err
			}
		}
	}

	// Step 2: 將執行中執行檔重新命名為備份。Windows 允許重新命名執行中的 .exe。
	if err := os.Rename(dst, backup); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Step 3: 將新二進位移至原始路徑。
	if err := os.Rename(src, dst); err != nil {
		// 還原備份，避免目標路徑空無一物。
		if _, statErr := os.Stat(backup); statErr == nil {
			_ = os.Rename(backup, dst)
		}
		return err
	}

	// Step 4: 清理備份檔。若檔案仍被執行中進程鎖定而無法刪除，排程於下次開機刪除。
	if err := os.Remove(backup); err != nil {
		_ = moveFileExW(backup, "", movefileDelayUntilReboot)
	}

	return nil
}