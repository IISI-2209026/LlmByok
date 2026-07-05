package version

import "testing"

// TestVersionDefault 確認未注入 ldflags 時 Version 預設為 "dev"。
func TestVersionDefault(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version 預設值應為 %q，得到 %q", "dev", Version)
	}
}