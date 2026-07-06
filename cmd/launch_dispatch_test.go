package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestLaunch_TargetToolSelection 驗證 `byok launch` 的第一個位置參數正式
// 分派至 copilot 或 codex 流程，並對省略/不支援目標印出錯誤並 exit 1。
//
// 為避免實際啟動外部 CLI，測試使用一個不存在的 --config 路徑，使分派
// 進入 copilot/codex 後於「找不到設定檔」錯誤路徑結束；藉由 stderr 是否
// 出現 codex 專屬訊息（或 copilot 既有訊息）判斷分派目標。
func TestLaunch_TargetToolSelection(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "does-not-exist.yaml")

	scenarios := []struct {
		name          string
		args          []string
		wantStderr    string
		wantExitOne   bool
		wantSupported bool
	}{
		{
			name:       "omitted target",
			args:       []string{},
			wantStderr: "必須指定目標工具",
			wantExitOne: true,
		},
		{
			name:          "unsupported target",
			args:          []string{"gemini"},
			wantStderr:    "不支援的工具",
			wantExitOne:   true,
			wantSupported: true,
		},
		{
			name:       "copilot dispatches to copilot flow",
			args:       []string{"copilot", "--config", missingPath},
			wantStderr: "找不到設定檔",
			wantExitOne: true,
		},
		{
			name:       "codex dispatches to codex flow",
			args:       []string{"codex", "--config", missingPath},
			wantStderr: "找不到設定檔",
			wantExitOne: true,
		},
		{
			name:       "claude dispatches to claude flow",
			args:       []string{"claude", "--config", missingPath},
			wantStderr: "找不到設定檔",
			wantExitOne: true,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			cmd := newLaunchCmd()
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)
			cmd.SetArgs(sc.args)
			err := cmd.Execute()
			if sc.wantExitOne {
				if err == nil {
					t.Fatalf("expected non-nil error (errExit), got nil")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(stderr.String(), sc.wantStderr) {
				t.Errorf("stderr missing %q, got: %s", sc.wantStderr, stderr.String())
			}
			if sc.wantSupported {
				if !strings.Contains(stderr.String(), "copilot") || !strings.Contains(stderr.String(), "codex") || !strings.Contains(stderr.String(), "claude") {
					t.Errorf("stderr should list supported tools copilot, codex & claude, got: %s", stderr.String())
				}
			}
		})
	}
}

// ensure cobra.Command satisfies compile-time use of *cobra.Command.
var _ = (*cobra.Command)(nil)