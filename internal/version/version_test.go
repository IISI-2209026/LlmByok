package version

import (
	"regexp"
	"testing"
)

// semverRe 匹配不含前置 "v" 的 semver base 版號（MAJOR.MINOR.PATCH）。
var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// TestVersionDefault 確認未注入 ldflags 時 Version 為合法的 canonical base 版號。
// 測試不變量（semver 格式、無 v prefix、非空），而非具體版號值，
// 使版號晉升不需同步修改本測試。
func TestVersionDefault(t *testing.T) {
	if Version == "" {
		t.Fatal("Version 預設值為空字串")
	}
	if !semverRe.MatchString(Version) {
		t.Errorf("Version 預設值 %q 不符合 semver 格式（MAJOR.MINOR.PATCH，無 v prefix）", Version)
	}
}