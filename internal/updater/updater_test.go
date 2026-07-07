package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// withAPIBase 暫時覆寫 apiBaseURL 指向 stub server，並於測試結束還原。
func withAPIBase(t *testing.T, url string) {
	t.Helper()
	prev := apiBaseURL
	apiBaseURL = url
	t.Cleanup(func() { apiBaseURL = prev })
}

// withExecutableFn 暫時覆寫 executableFn 回傳 targetPath。
func withExecutableFn(t *testing.T, targetPath string) {
	t.Helper()
	prev := executableFn
	executableFn = func() (string, error) { return targetPath, nil }
	t.Cleanup(func() { executableFn = prev })
}

func TestChannel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"0.1.0", "stable"},
		{"0.1.1-dev.42", "dev"},
		{"dev", "stable"},
	}
	for _, c := range cases {
		if got := Channel(c.in); got != c.want {
			t.Errorf("Channel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIsNewer_DevByRunNumber(t *testing.T) {
	newer, err := IsNewer("0.1.1-dev.42", "0.1.1-dev.50")
	if err != nil {
		t.Fatalf("IsNewer error: %v", err)
	}
	if !newer {
		t.Errorf("IsNewer(dev.42, dev.50) = false, want true")
	}
}

func TestIsNewer_StableSemver(t *testing.T) {
	newer, err := IsNewer("0.1.0", "0.1.1")
	if err != nil {
		t.Fatalf("IsNewer error: %v", err)
	}
	if !newer {
		t.Errorf("IsNewer(0.1.0, 0.1.1) = false, want true")
	}
}

func TestIsNewer_LocalDevNotNewer(t *testing.T) {
	newer, err := IsNewer("dev", "0.1.1")
	if err != nil {
		t.Fatalf("IsNewer error: %v", err)
	}
	if newer {
		t.Errorf("IsNewer(dev, 0.1.1) = true, want false")
	}
}

func TestIsNewer_DevParseError(t *testing.T) {
	_, err := IsNewer("0.1.1-dev.42", "0.1.1-dev.x")
	if err == nil {
		t.Fatalf("IsNewer(dev.42, dev.x) expected error, got nil")
	}
}

func TestIsNewer_CrossChannelError(t *testing.T) {
	_, err := IsNewer("0.1.0", "0.1.1-dev.50")
	if err == nil {
		t.Fatalf("IsNewer cross-channel expected error, got nil")
	}
}

func TestLatestRelease_DevChannel(t *testing.T) {
	body := []rawRelease{
		{TagName: "v0.1.1", Prerelease: false},
		{TagName: "v0.1.1-dev.42", Prerelease: true},
		{TagName: "v0.1.1-dev.50", Prerelease: true},
	}
	srv := stubReleasesServer(t, body)
	withAPIBase(t, srv.URL)
	rel, err := LatestRelease(context.Background(), "dev")
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.Tag != "v0.1.1-dev.50" {
		t.Errorf("LatestRelease(dev) tag = %q, want v0.1.1-dev.50", rel.Tag)
	}
	if !rel.Prerelease {
		t.Errorf("LatestRelease(dev) prerelease = false, want true")
	}
}

func TestLatestRelease_StableChannel(t *testing.T) {
	body := []rawRelease{
		{TagName: "v0.1.0", Prerelease: false},
		{TagName: "v0.1.1", Prerelease: false},
		{TagName: "v0.1.1-dev.50", Prerelease: true},
	}
	srv := stubReleasesServer(t, body)
	withAPIBase(t, srv.URL)
	rel, err := LatestRelease(context.Background(), "stable")
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.Tag != "v0.1.1" {
		t.Errorf("LatestRelease(stable) tag = %q, want v0.1.1", rel.Tag)
	}
}

func TestLatestRelease_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	withAPIBase(t, srv.URL)
	_, err := LatestRelease(context.Background(), "stable")
	if err == nil {
		t.Fatalf("LatestRelease on 500 expected error, got nil")
	}
}

func TestSelectAsset(t *testing.T) {
	rel := Release{
		Version: "0.1.1",
		Assets: []Asset{
			{Name: "byok-0.1.1-windows-amd64.zip", DownloadURL: "u1"},
			{Name: "byok-0.1.1-linux-amd64.tar.gz", DownloadURL: "u2"},
			{Name: "byok-0.1.1-darwin-arm64.tar.gz", DownloadURL: "u3"},
		},
	}
	cases := []struct {
		goos, goarch string
		wantName     string
		wantOK       bool
	}{
		{"windows", "amd64", "byok-0.1.1-windows-amd64.zip", true},
		{"linux", "amd64", "byok-0.1.1-linux-amd64.tar.gz", true},
		{"darwin", "arm64", "byok-0.1.1-darwin-arm64.tar.gz", true},
		{"freebsd", "amd64", "", false},
	}
	for _, c := range cases {
		a, ok := SelectAsset(rel, c.goos, c.goarch)
		if ok != c.wantOK {
			t.Errorf("SelectAsset(%s/%s) ok = %v, want %v", c.goos, c.goarch, ok, c.wantOK)
			continue
		}
		if ok && a.Name != c.wantName {
			t.Errorf("SelectAsset(%s/%s) name = %q, want %q", c.goos, c.goarch, a.Name, c.wantName)
		}
	}
}

// stubAssetServer 回傳預先組好的封存位元組。
func stubAssetServer(t *testing.T, data []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
}

// makeZip 組一個含 byok(.exe) 成員的 zip 封存。
func makeZip(t *testing.T, memberName string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(memberName)
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

// makeTarGz 組一個含 byok 成員的 tar.gz 封存。
func makeTarGz(t *testing.T, memberName string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: memberName, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("tar write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func TestDownloadAndExtract_Zip(t *testing.T) {
	content := []byte("fake-windows-binary")
	data := makeZip(t, "byok.exe", content)
	srv := stubAssetServer(t, data)
	defer srv.Close()
	got, err := downloadAndExtract(context.Background(), Asset{Name: "byok.exe", DownloadURL: srv.URL}, "windows")
	if err != nil {
		t.Fatalf("downloadAndExtract: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("downloadAndExtract zip = %q, want %q", got, content)
	}
}

func TestDownloadAndExtract_TarGz(t *testing.T) {
	content := []byte("fake-unix-binary")
	data := makeTarGz(t, "byok", content)
	srv := stubAssetServer(t, data)
	defer srv.Close()
	got, err := downloadAndExtract(context.Background(), Asset{Name: "byok", DownloadURL: srv.URL}, "linux")
	if err != nil {
		t.Fatalf("downloadAndExtract: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("downloadAndExtract tar.gz = %q, want %q", got, content)
	}
}

func TestDownloadAndExtract_NestedPath(t *testing.T) {
	content := []byte("nested-binary")
	data := makeTarGz(t, "bin/byok", content)
	srv := stubAssetServer(t, data)
	defer srv.Close()
	got, err := downloadAndExtract(context.Background(), Asset{Name: "byok", DownloadURL: srv.URL}, "linux")
	if err != nil {
		t.Fatalf("downloadAndExtract: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("downloadAndExtract nested = %q, want %q", got, content)
	}
}

func TestDownloadAndReplace_ReplaceTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "byok.exe")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatalf("write old: %v", err)
	}
	withExecutableFn(t, target)

	content := []byte("new-binary-bytes")
	var archive []byte
	if runtime.GOOS == "windows" {
		archive = makeZip(t, "byok.exe", content)
	} else {
		archive = makeTarGz(t, "byok", content)
	}
	srv := stubAssetServer(t, archive)
	defer srv.Close()

	rel := Release{
		Version: "0.1.1",
		Assets:  []Asset{{Name: assetName(t, "0.1.1"), DownloadURL: srv.URL}},
	}
	if err := DownloadAndReplace(context.Background(), rel, runtime.GOOS, runtime.GOARCH); err != nil {
		t.Fatalf("DownloadAndReplace: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("target content = %q, want %q", got, content)
	}
	// 確認檔案可執行（權限位）；Windows 以副檔名決定可執行性，跳過。
	if runtime.GOOS != "windows" {
		info, err := os.Stat(target)
		if err != nil {
			t.Fatalf("stat target: %v", err)
		}
		if info.Mode()&0o100 == 0 {
			t.Errorf("target not executable: mode %v", info.Mode())
		}
	}
}

func TestDownloadAndReplace_AssetNotFound(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "byok")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatalf("write old: %v", err)
	}
	withExecutableFn(t, target)
	rel := Release{Version: "0.1.1", Assets: nil}
	err := DownloadAndReplace(context.Background(), rel, runtime.GOOS, runtime.GOARCH)
	if err == nil {
		t.Fatalf("expected asset-not-found error, got nil")
	}
	got, _ := os.ReadFile(target)
	if string(got) != "old" {
		t.Errorf("target modified on asset-not-found: %q", got)
	}
}

func TestDownloadAndReplace_NetworkError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "byok")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatalf("write old: %v", err)
	}
	withExecutableFn(t, target)
	// 以關閉的 server 模擬網路失敗。
	srv := stubAssetServer(t, []byte("x"))
	srv.Close()
	rel := Release{
		Version: "0.1.1",
		Assets:  []Asset{{Name: assetName(t, "0.1.1"), DownloadURL: srv.URL}},
	}
	err := DownloadAndReplace(context.Background(), rel, runtime.GOOS, runtime.GOARCH)
	if err == nil {
		t.Fatalf("expected network error, got nil")
	}
	got, _ := os.ReadFile(target)
	if string(got) != "old" {
		t.Errorf("target modified on network error: %q", got)
	}
}

// assetName 依當前平台組出預期資產名稱。
func assetName(t *testing.T, version string) string {
	t.Helper()
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("byok-%s-%s-%s.%s", version, runtime.GOOS, runtime.GOARCH, ext)
}

// stubReleasesServer 回傳固定 releases JSON 的 stub server。
func stubReleasesServer(t *testing.T, body []rawRelease) *httptest.Server {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal stub releases: %v", err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); !strings.Contains(got, "vnd.github+json") {
			t.Errorf("Accept header = %q, want vnd.github+json", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, string(raw))
	}))
}