//go:build !windows

package updater

import "os"

// renameReplace 在 Unix 上以 os.Rename 原子覆蓋 dst（同檔案系統為原子操作）。
func renameReplace(src, dst string) error {
	return os.Rename(src, dst)
}