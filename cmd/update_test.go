package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/updater"
)

// stubFetcher 為 fetcher 介面的測試樁，可控制每個方法的回傳值並記錄呼叫。
type stubFetcher struct {
	queriedChannel string
	queriedVersion string
	release        updater.Release
	releaseErr     error
	newer          bool
	newerErr       error
	replaceErr     error
	replaced       bool
	replaceFn      func(rel updater.Release, goos, goarch string) error
}

func (s *stubFetcher) Channel(v string) string { return updater.Channel(v) }

func (s *stubFetcher) LatestRelease(ctx context.Context, channel string) (updater.Release, error) {
	s.queriedChannel = channel
	return s.release, s.releaseErr
}

func (s *stubFetcher) IsNewer(current, latest string) (bool, error) {
	s.queriedVersion = current + "->" + latest
	return s.newer, s.newerErr
}

func (s *stubFetcher) DownloadAndReplace(ctx context.Context, rel updater.Release, goos, goarch string) error {
	s.replaced = true
	if s.replaceErr != nil {
		return s.replaceErr
	}
	if s.replaceFn != nil {
		return s.replaceFn(rel, goos, goarch)
	}
	return nil
}

// withFetcher 暫時覆寫 defaultFetcher，並於測試結束還原。
func withFetcher(t *testing.T, f fetcher) {
	t.Helper()
	prev := defaultFetcher
	defaultFetcher = f
	t.Cleanup(func() { defaultFetcher = prev })
}

func TestUpdateCmd_NoUpdate(t *testing.T) {
	stub := &stubFetcher{release: updater.Release{Version: "0.1.0"}, newer: false}
	withFetcher(t, stub)
	var out, errOut bytes.Buffer
	if e := runUpdate(stub, "0.1.0", false, "", "linux", "amd64", &out, &errOut); e != nil {
		t.Fatalf("runUpdate returned error: %v", e)
	}
	if !strings.Contains(out.String(), "已是最新版本") {
		t.Errorf("stdout = %q, want contains 已是最新版本", out.String())
	}
	if stub.replaced {
		t.Errorf("DownloadAndReplace should not be called when up to date")
	}
}

func TestUpdateCmd_CheckOnly(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "byok")
	if err := os.WriteFile(target, []byte("original"), 0o755); err != nil {
		t.Fatalf("write target: %v", err)
	}
	stub := &stubFetcher{
		release: updater.Release{Version: "0.1.1"},
		newer:   true,
		replaceFn: func(rel updater.Release, goos, goarch string) error {
			t.Errorf("DownloadAndReplace should not be called in --check mode")
			return nil
		},
	}
	withFetcher(t, stub)
	var out, errOut bytes.Buffer
	if e := runUpdate(stub, "0.1.0", true, "", "linux", "amd64", &out, &errOut); e != nil {
		t.Fatalf("runUpdate returned error: %v", e)
	}
	if !strings.Contains(out.String(), "0.1.1") || !strings.Contains(out.String(), "0.1.0") {
		t.Errorf("stdout = %q, want latest and current versions", out.String())
	}
	got, _ := os.ReadFile(target)
	if string(got) != "original" {
		t.Errorf("target modified in --check mode: %q", got)
	}
}

func TestUpdateCmd_ChannelOverride_Prerelease(t *testing.T) {
	stub := &stubFetcher{release: updater.Release{Version: "0.1.1-dev.50"}, newer: true}
	withFetcher(t, stub)
	var out, errOut bytes.Buffer
	_ = runUpdate(stub, "0.1.0", true, "prerelease", "linux", "amd64", &out, &errOut)
	if stub.queriedChannel != "dev" {
		t.Errorf("queried channel = %q, want dev", stub.queriedChannel)
	}
}

func TestUpdateCmd_ChannelOverride_Release(t *testing.T) {
	stub := &stubFetcher{release: updater.Release{Version: "0.1.1"}, newer: true}
	withFetcher(t, stub)
	var out, errOut bytes.Buffer
	_ = runUpdate(stub, "0.1.1-dev.42", true, "release", "linux", "amd64", &out, &errOut)
	if stub.queriedChannel != "stable" {
		t.Errorf("queried channel = %q, want stable", stub.queriedChannel)
	}
}

func TestUpdateCmd_InvalidChannel(t *testing.T) {
	stub := &stubFetcher{release: updater.Release{Version: "0.1.1"}, newer: true}
	withFetcher(t, stub)
	var out, errOut bytes.Buffer
	e := runUpdate(stub, "0.1.0", false, "beta", "linux", "amd64", &out, &errOut)
	if e == nil {
		t.Fatalf("expected errExit, got nil")
	}
	if stub.queriedChannel != "" {
		t.Errorf("channel queried on invalid value: %q", stub.queriedChannel)
	}
	if !strings.Contains(errOut.String(), "不合法的 channel") {
		t.Errorf("stderr = %q, want contains 不合法的 channel", errOut.String())
	}
}

func TestUpdateCmd_AssetNotFound(t *testing.T) {
	stub := &stubFetcher{
		release:    updater.Release{Version: "0.1.1"},
		newer:      true,
		replaceErr: errors.New("updater: 找不到 linux/amd64 資產"),
	}
	withFetcher(t, stub)
	var out, errOut bytes.Buffer
	e := runUpdate(stub, "0.1.0", false, "", "linux", "amd64", &out, &errOut)
	if e == nil {
		t.Fatalf("expected errExit on asset not found, got nil")
	}
	if !strings.Contains(errOut.String(), "更新失敗") {
		t.Errorf("stderr = %q, want contains 更新失敗", errOut.String())
	}
}

func TestUpdateCmd_NetworkError(t *testing.T) {
	stub := &stubFetcher{
		releaseErr: errors.New("network timeout"),
	}
	withFetcher(t, stub)
	var out, errOut bytes.Buffer
	e := runUpdate(stub, "0.1.0", false, "", "linux", "amd64", &out, &errOut)
	if e == nil {
		t.Fatalf("expected errExit on network error, got nil")
	}
	if !strings.Contains(errOut.String(), "查詢最新版本失敗") {
		t.Errorf("stderr = %q, want contains 查詢最新版本失敗", errOut.String())
	}
}

func TestUpdateCmd_UpdatesBinary(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "byok")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatalf("write target: %v", err)
	}
	stub := &stubFetcher{
		release: updater.Release{Version: "0.1.1"},
		newer:   true,
		replaceFn: func(rel updater.Release, goos, goarch string) error {
			return os.WriteFile(target, []byte("new-bytes"), 0o755)
		},
	}
	withFetcher(t, stub)
	var out, errOut bytes.Buffer
	if e := runUpdate(stub, "0.1.0", false, "", "linux", "amd64", &out, &errOut); e != nil {
		t.Fatalf("runUpdate returned error: %v", e)
	}
	if !strings.Contains(out.String(), "已更新至 0.1.1") {
		t.Errorf("stdout = %q, want contains 已更新至 0.1.1", out.String())
	}
	got, _ := os.ReadFile(target)
	if string(got) != "new-bytes" {
		t.Errorf("target content = %q, want new-bytes", got)
	}
}

func TestNewUpdateCmd_HelpFlags(t *testing.T) {
	c := newUpdateCmd("0.1.0")
	c.SetOut(new(bytes.Buffer))
	help := c.UsageString()
	for _, want := range []string{"--check", "--channel"} {
		if !strings.Contains(help, want) {
			t.Errorf("update help missing %q:\n%s", want, help)
		}
	}
}