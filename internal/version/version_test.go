package version

import "testing"

// TestVersionDefault 確認未注入 ldflags 時 Version 為 canonical base 版號 "0.1.0"。
func TestVersionDefault(t *testing.T) {
	if Version != "0.1.0" {
		t.Errorf("Version 預設值應為 %q，得到 %q", "0.1.0", Version)
	}
}