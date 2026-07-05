// Package updater 實作 byok 的自我更新能力：channel 判定、GitHub Releases
// 查詢、平台資產選擇、下載與跨平台執行檔原子替換。
//
// 不引入第三方 Go 相依，僅使用標準庫。
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
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Channel 依版本字串判定更新 channel。
// 含子字串 "-dev." 視為 "dev" channel，否則為 "stable"。
// 字面 "dev"（本地開發建置）不含 "-dev."，故回傳 "stable"。
func Channel(v string) string {
	if strings.Contains(v, "-dev.") {
		return "dev"
	}
	return "stable"
}

// devRunNumber 從版本字串中取出 -dev.<N> 的數字 N。
// 找不到或無法解析為整數時回傳錯誤。
func devRunNumber(v string) (int, error) {
	re := regexp.MustCompile(`-dev\.(\d+)`)
	m := re.FindStringSubmatch(v)
	if m == nil {
		return 0, fmt.Errorf("updater: version %q lacks -dev.<N> run number", v)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, fmt.Errorf("updater: parse run number in %q: %w", v, err)
	}
	return n, nil
}

// IsNewer 比較同 channel 內兩版本的新舊。
// dev channel 以 -dev.<N> 數字比較；stable 以 semver (major.minor.patch) 比較。
// 當前版本為字面 "dev" 時一律回傳 false（避免開發中誤升）。
// 跨 channel 比較回傳錯誤。
func IsNewer(current, latest string) (bool, error) {
	if current == "dev" {
		return false, nil
	}
	cc := Channel(current)
	lc := Channel(latest)
	if cc != lc {
		return false, fmt.Errorf("updater: cross-channel comparison %q (%s) vs %q (%s)", current, cc, latest, lc)
	}
	if cc == "dev" {
		cn, err := devRunNumber(current)
		if err != nil {
			return false, err
		}
		ln, err := devRunNumber(latest)
		if err != nil {
			return false, err
		}
		return ln > cn, nil
	}
	return semverNewer(current, latest)
}

// semverNewer 比較兩 stable 版本（major.minor.patch），latest > current 回傳 true。
func semverNewer(current, latest string) (bool, error) {
	c, err := parseSemver(current)
	if err != nil {
		return false, fmt.Errorf("updater: parse current %q: %w", current, err)
	}
	l, err := parseSemver(latest)
	if err != nil {
		return false, fmt.Errorf("updater: parse latest %q: %w", latest, err)
	}
	if l[0] != c[0] {
		return l[0] > c[0], nil
	}
	if l[1] != c[1] {
		return l[1] > c[1], nil
	}
	return l[2] > c[2], nil
}

// parseSemver 解析 "major.minor.patch"（可含 v prefix 與後綴，僅取前三段）。
func parseSemver(v string) ([3]int, error) {
	var out [3]int
	s := strings.TrimPrefix(v, "v")
	if idx := strings.IndexAny(s, "-+"); idx >= 0 {
		s = s[:idx]
	}
	parts := strings.Split(s, ".")
	if len(parts) < 3 {
		return out, fmt.Errorf("updater: %q is not major.minor.patch", v)
	}
	for i, p := range parts[:3] {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, fmt.Errorf("updater: segment %q in %q not numeric", p, v)
		}
		out[i] = n
	}
	return out, nil
}

// Release 代表一個 GitHub Release 的更新相關資訊。
type Release struct {
	Tag        string  // 如 "v0.1.1" 或 "v0.1.1-dev.42"
	Version    string  // 去 v prefix
	Prerelease bool
	Assets     []Asset
}

// Asset 為 Release 中可下載的平台資產。
type Asset struct {
	Name        string
	DownloadURL string
}

type rawRelease struct {
	TagName    string     `json:"tag_name"`
	Prerelease bool       `json:"prerelease"`
	Assets     []rawAsset `json:"assets"`
}

type rawAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// 可注入的依賴，供測試覆寫。
var (
	apiBaseURL   = "https://api.github.com/repos/IISI-2209026/LlmByok/releases"
	httpClient   = &http.Client{Timeout: 10 * time.Second}
	executableFn = os.Executable
)

// LatestRelease 查詢指定 channel 的最新 GitHub Release。
// channel 為 "dev"（只看 prerelease=true）或 "stable"（只看 prerelease=false）。
func LatestRelease(ctx context.Context, channel string) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseURL, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return Release{}, fmt.Errorf("updater: github api status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Release{}, err
	}
	var raws []rawRelease
	if err := json.Unmarshal(body, &raws); err != nil {
		return Release{}, fmt.Errorf("updater: decode releases: %w", err)
	}
	wantPre := channel == "dev"
	var best *rawRelease
	for i := range raws {
		r := &raws[i]
		if r.Prerelease != wantPre {
			continue
		}
		if best == nil || releaseNewer(r, best, channel) {
			best = r
		}
	}
	if best == nil {
		return Release{}, fmt.Errorf("updater: no releases in channel %q", channel)
	}
	return toRelease(best), nil
}

// releaseNewer 判定 r 是否比 best 更新（同 channel）。
func releaseNewer(r, best *rawRelease, channel string) bool {
	rv := strings.TrimPrefix(r.TagName, "v")
	bv := strings.TrimPrefix(best.TagName, "v")
	if channel == "dev" {
		rn, err := devRunNumber(rv)
		if err != nil {
			return false
		}
		bn, err := devRunNumber(bv)
		if err != nil {
			return false
		}
		return rn > bn
	}
	newer, err := semverNewer(bv, rv)
	if err != nil {
		return false
	}
	return newer
}

func toRelease(r *rawRelease) Release {
	assets := make([]Asset, 0, len(r.Assets))
	for _, a := range r.Assets {
		assets = append(assets, Asset{Name: a.Name, DownloadURL: a.BrowserDownloadURL})
	}
	return Release{
		Tag:        r.TagName,
		Version:    strings.TrimPrefix(r.TagName, "v"),
		Prerelease: r.Prerelease,
		Assets:     assets,
	}
}

// SelectAsset 依 GOOS/GOARCH 從 Release 中選出對應平台資產。
// 資產命名為 byok-<version>-<goos>-<goarch>.<ext>（windows→zip、其他→tar.gz）。
func SelectAsset(rel Release, goos, goarch string) (Asset, bool) {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	want := fmt.Sprintf("byok-%s-%s-%s.%s", rel.Version, goos, goarch, ext)
	for _, a := range rel.Assets {
		if a.Name == want {
			return a, true
		}
	}
	return Asset{}, false
}

// downloadAndExtract 下載資產並解壓出 byok（或 byok.exe）成員的位元組內容。
func downloadAndExtract(ctx context.Context, asset Asset, goos string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.DownloadURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("updater: download status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if goos == "windows" {
		return extractZip(body)
	}
	return extractTarGz(body)
}

// extractZip 從 zip 內找出 byok.exe 或 byok 成員並回傳其內容。
func extractZip(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("updater: read zip: %w", err)
	}
	for _, f := range zr.File {
		if baseName(f.Name) == "byok.exe" || baseName(f.Name) == "byok" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("updater: zip has no byok(byok.exe) member")
}

// extractTarGz 從 tar.gz 內找出 byok 或 byok.exe 成員並回傳其內容。
func extractTarGz(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("updater: read gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("updater: read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if baseName(hdr.Name) == "byok" || baseName(hdr.Name) == "byok.exe" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("updater: tar.gz has no byok(byok.exe) member")
}

// baseName 回傳路徑的檔名部分（跨平台路徑分隔）。
func baseName(p string) string {
	p = filepath.ToSlash(p)
	if idx := strings.LastIndex(p, "/"); idx >= 0 {
		return p[idx+1:]
	}
	return p
}

// DownloadAndReplace 下載所選 release 的對應平台資產並原子替換當前執行檔。
func DownloadAndReplace(ctx context.Context, rel Release, goos, goarch string) error {
	target, err := executableFn()
	if err != nil {
		return err
	}
	return replaceAt(ctx, rel, goos, goarch, target)
}

// replaceAt 為可測試的核心：將新二進位寫入同目錄暫存檔，再原子替換 targetPath。
func replaceAt(ctx context.Context, rel Release, goos, goarch, targetPath string) error {
	asset, ok := SelectAsset(rel, goos, goarch)
	if !ok {
		return fmt.Errorf("updater: 找不到 %s/%s 資產", goos, goarch)
	}
	data, err := downloadAndExtract(ctx, asset, goos)
	if err != nil {
		return err
	}
	dir := filepath.Dir(targetPath)
	tmp, err := os.CreateTemp(dir, ".byok-update-*")
	if err != nil {
		return fmt.Errorf("updater: 建立暫存檔: %w", err)
	}
	tmpPath := tmp.Name()
	// 若後續替換失敗，清理暫存檔；替換成功後 tmpPath 已不存在，Remove 為 no-op。
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("updater: 寫入暫存檔: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("updater: 關閉暫存檔: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("updater: chmod 暫存檔: %w", err)
	}
	return renameReplace(tmpPath, targetPath)
}