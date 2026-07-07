//go:build windows

package updater

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestRenameReplaceWindowsRunningExe 驗證 rename-then-move 能替換執行中的執行檔。
func TestRenameReplaceWindowsRunningExe(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "byok.exe")

	// 編譯一個會持續執行的 helper .exe 作為「執行中」的目標。
	helperSrc := filepath.Join(dir, "helper.go")
	if err := os.WriteFile(helperSrc, []byte(`package main
import "time"
func main(){time.Sleep(60*time.Second)}
`), 0o644); err != nil {
		t.Fatalf("write helper src: %v", err)
	}
	build := exec.Command("go", "build", "-o", target, helperSrc)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build helper: %v\n%s", err, out)
	}

	// 啟動 helper — target 現在被執行中進程鎖定。
	helper := exec.Command(target)
	if err := helper.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	defer func() {
		_ = helper.Process.Kill()
		_, _ = helper.Process.Wait()
	}()
	time.Sleep(200 * time.Millisecond) // 確保進程已啟動

	// 準備新二進位作為來源。
	src := filepath.Join(dir, "new.exe")
	newContent := []byte("new-version-binary")
	if err := os.WriteFile(src, newContent, 0o755); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// 執行 renameReplace — 舊實作（MoveFileExW + REPLACE_EXISTING）在此會回傳 ACCESS_DENIED。
	if err := renameReplace(src, target); err != nil {
		t.Fatalf("renameReplace on running exe: %v", err)
	}

	// 驗證目標路徑內容為新二進位。
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !bytes.Equal(got, newContent) {
		t.Errorf("target content = %q, want %q", got, newContent)
	}

	// 備份檔應存在（仍被執行中進程鎖定）或已被排程刪除。
	backup := target + ".old"
	if _, err := os.Stat(backup); err == nil {
		// 備份檔存在 — 確認它不是新內容（應為舊 helper 二進位）。
		gotBackup, _ := os.ReadFile(backup)
		if bytes.Equal(gotBackup, newContent) {
			t.Errorf("backup contains new content, should contain old binary")
		}
	}
	// 備份檔不存在表示已被立即刪除（進程已釋放鎖定的罕見情況）或已排程開機刪除，皆可接受。
}

// TestRenameReplaceWindowsExistingOld 驗證既存 .old 檔案被正確處理。
func TestRenameReplaceWindowsExistingOld(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "byok.exe")
	backup := target + ".old"

	// 建立目標檔案。
	oldContent := []byte("old-version")
	if err := os.WriteFile(target, oldContent, 0o755); err != nil {
		t.Fatalf("write target: %v", err)
	}

	// 預先放置既存 .old（模擬上一輪更新殘留）。
	staleContent := []byte("stale-backup")
	if err := os.WriteFile(backup, staleContent, 0o755); err != nil {
		t.Fatalf("write stale backup: %v", err)
	}

	// 準備新二進位。
	src := filepath.Join(dir, "new.exe")
	newContent := []byte("new-version")
	if err := os.WriteFile(src, newContent, 0o755); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := renameReplace(src, target); err != nil {
		t.Fatalf("renameReplace with existing .old: %v", err)
	}

	// 驗證目標內容為新二進位。
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !bytes.Equal(got, newContent) {
		t.Errorf("target content = %q, want %q", got, newContent)
	}

	// 既存 .old 已被處理：若 .old 仍存在應為舊目標內容（old-version），而非殘留的 stale-backup。
	if gotBackup, err := os.ReadFile(backup); err == nil {
		if bytes.Equal(gotBackup, staleContent) {
			t.Errorf("backup still contains stale content, should have been overwritten")
		}
	}
	// .old 不存在表示已被成功刪除（非鎖定），亦可接受。
}

// TestRenameReplaceWindowsFailureRestore 驗證 step 3 失敗時還原備份。
func TestRenameReplaceWindowsFailureRestore(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "byok.exe")
	backup := target + ".old"

	oldContent := []byte("old-version")
	if err := os.WriteFile(target, oldContent, 0o755); err != nil {
		t.Fatalf("write target: %v", err)
	}

	// src 不存在，模擬 step 3（os.Rename(src, dst)）失敗。
	src := filepath.Join(dir, "nonexistent.exe")

	err := renameReplace(src, target)
	if err == nil {
		t.Fatalf("expected error for nonexistent src, got nil")
	}

	// 驗證備份已還原：target 應仍為舊內容。
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !bytes.Equal(got, oldContent) {
		t.Errorf("target content = %q, want %q (restored)", got, oldContent)
	}

	// 備份檔不應存在（已還原為 target）。
	if _, err := os.Stat(backup); err == nil {
		t.Errorf("backup should not exist after restore")
	}
}

// TestDownloadAndReplace_WindowsRunningExe 整合測試：驗證完整 DownloadAndReplace
// 流程搭配執行中執行檔，模擬 byok update 在 Windows 上的端對端行為。
//
// 只覆寫 executableFn（目標路徑）與 asset DownloadURL（stub HTTP server），
// 其餘全程使用生產程式碼路徑：SelectAsset → downloadAndExtract → 暫存檔 → renameReplace。
func TestDownloadAndReplace_WindowsRunningExe(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "byok.exe")

	// 編譯 helper .exe 並啟動，模擬執行中的 byok.exe。
	helperSrc := filepath.Join(dir, "helper.go")
	if err := os.WriteFile(helperSrc, []byte(`package main
import "time"
func main(){time.Sleep(60*time.Second)}
`), 0o644); err != nil {
		t.Fatalf("write helper src: %v", err)
	}
	if out, err := exec.Command("go", "build", "-o", target, helperSrc).CombinedOutput(); err != nil {
		t.Fatalf("go build helper: %v\n%s", err, out)
	}

	helper := exec.Command(target)
	if err := helper.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	defer func() {
		_ = helper.Process.Kill()
		_, _ = helper.Process.Wait()
	}()
	time.Sleep(200 * time.Millisecond) // 等待進程就緒

	// 覆寫目標路徑為執行中的 helper。
	withExecutableFn(t, target)

	// 建立含新二進位的 zip 封存，由 stub HTTP server 回傳。
	newContent := []byte("new-version-binary")
	archive := makeZip(t, "byok.exe", newContent)
	srv := stubAssetServer(t, archive)
	defer srv.Close()

	rel := Release{
		Version: "0.2.1",
		Assets: []Asset{{
			Name:         assetName(t, "0.2.1"),
			DownloadURL:  srv.URL,
		}},
	}

	// 執行完整 DownloadAndReplace 流程（生產程式碼路徑）。
	if err := DownloadAndReplace(context.Background(), rel, "windows", runtime.GOARCH); err != nil {
		t.Fatalf("DownloadAndReplace on running exe: %v", err)
	}

	// 驗證目標路徑內容為新二進位。
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !bytes.Equal(got, newContent) {
		t.Errorf("target content = %q, want %q", got, newContent)
	}

	// 備份檔（.old）可能因執行中進程鎖定而存在，或已排程於重啟時刪除。
	// 若存在，內容應為舊二進位（helper），而非新內容。
	backup := target + ".old"
	if gotBackup, err := os.ReadFile(backup); err == nil {
		if bytes.Equal(gotBackup, newContent) {
			t.Errorf("backup contains new content, should contain old binary")
		}
	}
	// .old 不存在表示已被成功刪除，亦可接受。
}