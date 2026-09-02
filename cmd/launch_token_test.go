package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
	"github.com/zalando/go-keyring"
)

// ptrTokInt64 供測試建構可選 token 值指標。
func ptrToken(v int64) *int64 { return &v }

// TestResolveTokenLimits_PerFieldPrecedence 驗證逐欄位獨立解析，優先序為
// CLI → 模型 → profile 預設 → 未設定。
func TestResolveTokenLimits_PerFieldPrecedence(t *testing.T) {
	profile := &config.Profile{
		Name:               "p",
		Models:             []config.Model{{Name: "gpt-5.4", ContextWindowTokens: ptrToken(1000000), MaxOutputTokens: ptrToken(128000)}},
		DefaultModelLimits: &config.ModelLimits{ContextWindowTokens: ptrToken(272000), MaxOutputTokens: ptrToken(16384)},
	}

	t.Run("CLI overrides model and profile independently", func(t *testing.T) {
		got := resolveTokenLimits(profile, "gpt-5.4", ptrToken(500000), nil)
		if got.ContextWindow == nil || *got.ContextWindow != 500000 {
			t.Errorf("ContextWindow = %v, want 500000 (CLI)", got.ContextWindow)
		}
		if got.ContextWindowSource != tokenSourceCLI {
			t.Errorf("ContextWindowSource = %q, want %q", got.ContextWindowSource, tokenSourceCLI)
		}
		if got.MaxOutput == nil || *got.MaxOutput != 128000 {
			t.Errorf("MaxOutput = %v, want 128000 (model value)", got.MaxOutput)
		}
		if got.MaxOutputSource != tokenSourceModel {
			t.Errorf("MaxOutputSource = %q, want %q", got.MaxOutputSource, tokenSourceModel)
		}
	})

	t.Run("missing model field falls back to profile default", func(t *testing.T) {
		p := &config.Profile{
			Name: "p",
			Models: []config.Model{
				{Name: "gpt-5.4-mini", MaxOutputTokens: ptrToken(32768)},
			},
			DefaultModelLimits: &config.ModelLimits{ContextWindowTokens: ptrToken(272000)},
		}
		got := resolveTokenLimits(p, "gpt-5.4-mini", nil, nil)
		if got.ContextWindow == nil || *got.ContextWindow != 272000 {
			t.Errorf("ContextWindow = %v, want 272000 (profile default)", got.ContextWindow)
		}
		if got.ContextWindowSource != tokenSourceProfileDefault {
			t.Errorf("ContextWindowSource = %q, want %q", got.ContextWindowSource, tokenSourceProfileDefault)
		}
		if got.MaxOutput == nil || *got.MaxOutput != 32768 {
			t.Errorf("MaxOutput = %v, want 32768 (model)", got.MaxOutput)
		}
		if got.MaxOutputSource != tokenSourceModel {
			t.Errorf("MaxOutputSource = %q, want %q", got.MaxOutputSource, tokenSourceModel)
		}
	})

	t.Run("unknown model override uses profile defaults only", func(t *testing.T) {
		got := resolveTokenLimits(profile, "experimental-model", nil, nil)
		if got.ContextWindow == nil || *got.ContextWindow != 272000 {
			t.Errorf("ContextWindow = %v, want 272000 (profile default, not borrowed)", got.ContextWindow)
		}
		if got.MaxOutput == nil || *got.MaxOutput != 16384 {
			t.Errorf("MaxOutput = %v, want 16384 (profile default, not borrowed)", got.MaxOutput)
		}
		if got.ContextWindowSource != tokenSourceProfileDefault || got.MaxOutputSource != tokenSourceProfileDefault {
			t.Errorf("sources = %q,%q, want both profile default", got.ContextWindowSource, got.MaxOutputSource)
		}
	})

	t.Run("fully unset limits remain unset", func(t *testing.T) {
		p := &config.Profile{Name: "p", Models: []config.Model{{Name: "solo"}}}
		got := resolveTokenLimits(p, "solo", nil, nil)
		if got.ContextWindow != nil || got.MaxOutput != nil {
			t.Errorf("expected fully unset limits, got ctx=%v out=%v", got.ContextWindow, got.MaxOutput)
		}
	})
}

// TestValidatePositiveTokenFlag 驗證 token 旗標僅接受正整數；未提供時不通過驗證。
func TestValidatePositiveTokenFlag(t *testing.T) {
	cases := []struct {
		name    string
		changed bool
		value   int64
		wantErr bool
	}{
		{"unset flag skipped", false, 0, false},
		{"positive accepted", true, 1000000, false},
		{"zero rejected", true, 0, true},
		{"negative rejected", true, -1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePositiveTokenFlag("--context-window-tokens", tc.changed, tc.value)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr && !strings.Contains(err.Error(), "--context-window-tokens") {
				t.Errorf("error must name the flag, got: %v", err)
			}
		})
	}
}

// TestLaunch_TokenFlagValidationBeforeSetup 驗證非正 token 旗標錯誤發生於
// 讀取 API key、檢查可執行檔或啟動子程序之前：config 檔案存在且 copilot
// 不在 PATH 上，錯誤訊息應指向旗標而非「找不到設定檔」或可執行檔錯誤。
func TestLaunch_TokenFlagValidationBeforeSetup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    models: [gpt-4o]\ndefault_profile: openai-official\n")
	t.Setenv("PATH", "")

	cases := []struct {
		name string
		args []string
		flag string
	}{
		{"zero context rejected", []string{"claude", "--config", path, "--context-window-tokens", "0"}, "--context-window-tokens"},
		{"negative output rejected", []string{"pi", "--config", path, "--max-output-tokens=-1"}, "--max-output-tokens"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newLaunchCmd()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error (errExit), got nil")
			}
			if !strings.Contains(stderr.String(), tc.flag) {
				t.Errorf("stderr must name %s, got: %s", tc.flag, stderr.String())
			}
			if strings.Contains(stderr.String(), "找不到設定檔") || strings.Contains(stderr.String(), "找不到") && strings.Contains(stderr.String(), "可執行檔") {
				t.Errorf("validation must happen before config/executable setup, stderr: %s", stderr.String())
			}
		})
	}
}

// TestLaunch_PositiveTokenFlagsContinueToTargetFlow 驗證正整數旗標通過驗證
// 並繼續正常 target 流程。
func TestLaunch_PositiveTokenFlagsContinueToTargetFlow(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	cmd := newLaunchCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"copilot", "--config", missingPath, "--context-window-tokens", "1000000", "--max-output-tokens", "128000"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error (missing config file), got nil")
	}
	// 正整數旗標有效 → 應抵達設定檔載入階段（找不到設定檔的錯誤）。
	if !strings.Contains(stderr.String(), "找不到設定檔") {
		t.Errorf("positive flags should pass validation and reach config load, stderr: %s", stderr.String())
	}
}

// TestWarnUnsupportedTokenLimits 驗證 warning helper 的客體分流（target、參數、值、來源）與 unset 靜默。
func TestWarnUnsupportedTokenLimits(t *testing.T) {
	t.Run("warns with target param value and source", func(t *testing.T) {
		limits := &resolvedTokenLimits{MaxOutput: ptrToken(32768), MaxOutputSource: tokenSourceProfileDefault}
		var stderr bytes.Buffer
		warnUnsupportedTokenLimits("codex", limits, &stderr)
		got := stderr.String()
		for _, want := range []string{"codex", "max_output_tokens", "32768", "profile default"} {
			if !strings.Contains(got, want) {
				t.Errorf("warning %q should contain %q", got, want)
			}
		}
	})
	t.Run("model source is named", func(t *testing.T) {
		limits := &resolvedTokenLimits{MaxOutput: ptrToken(4096), MaxOutputSource: tokenSourceModel}
		var stderr bytes.Buffer
		warnUnsupportedTokenLimits("codex", limits, &stderr)
		if !strings.Contains(stderr.String(), "model") {
			t.Errorf("warning should name model source, got: %q", stderr.String())
		}
	})
	t.Run("unset output is silent", func(t *testing.T) {
		var stderr bytes.Buffer
		warnUnsupportedTokenLimits("codex", &resolvedTokenLimits{}, &stderr)
		if stderr.Len() != 0 {
			t.Errorf("unset max_output_tokens must not warn, got: %q", stderr.String())
		}
	})
	t.Run("nil limits is silent", func(t *testing.T) {
		var stderr bytes.Buffer
		warnUnsupportedTokenLimits("codex", nil, &stderr)
		if stderr.Len() != 0 {
			t.Errorf("nil limits must not warn, got: %q", stderr.String())
		}
	})
}

// TestLaunchCodex_UnsupportedMaxOutputWarning 驗證 normal launch 對 codex 的
// 有效 max_output_tokens 寫出 stderr warning、不啟動錯誤且 stdout 無 warning。
func TestLaunchCodex_UnsupportedMaxOutputWarning(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, `profiles:
  - name: openai-official
    provider: openai
    api_base: https://api.openai.com/v1
    api_key: sk-xxxx
    default_model_limits:
      max_output_tokens: 32768
    models: [gpt-4o]
default_profile: openai-official
`)
	stubExe := buildCopilotStubForCodex(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	var stdout, stderr bytes.Buffer
	if err := runLaunchCodex(path, "", "", nil, &stdout, &stderr); err != nil {
		t.Fatalf("launch should continue after warning, got err: %v (stderr=%s)", err, stderr.String())
	}
	got := stderr.String()
	for _, want := range []string{"codex", "max_output_tokens", "32768", "profile default"} {
		if !strings.Contains(got, want) {
			t.Errorf("stderr %q should contain %q", got, want)
		}
	}
	if strings.Contains(stdout.String(), "max_output_tokens") {
		t.Errorf("stdout must not contain warning text, got: %s", stdout.String())
	}
}

// TestLaunchCodex_UnsetOutputNoWarning 驗證 unset max_output_tokens 不提示。
func TestLaunchCodex_UnsetOutputNoWarning(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    models: [gpt-4o]\ndefault_profile: openai-official\n")
	stubExe := buildCopilotStubForCodex(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	var stdout, stderr bytes.Buffer
	if err := runLaunchCodex(path, "", "", nil, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchCodex: %v (stderr=%s)", err, stderr.String())
	}
	if strings.Contains(stderr.String(), "不支援") {
		t.Errorf("no warning expected when max_output_tokens unset, stderr: %s", stderr.String())
	}
}

// TestLaunchCodexApp_UnsupportedMaxOutputWarning 驗證 codex-app 同樣
// warning-only：app 開啟繼續進行。
func TestLaunchCodexApp_UnsupportedMaxOutputWarning(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, `profiles:
  - name: openai-official
    provider: openai
    api_base: https://api.openai.com/v1
    api_key: sk-xxxx
    default_model_limits:
      max_output_tokens: 65536
    models: [gpt-4o]
default_profile: openai-official
`)
	stubExe := buildCopilotStubForCodex(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	var stdout, stderr bytes.Buffer
	if err := runLaunchCodexApp(path, "", "", nil, &stdout, &stderr); err != nil {
		t.Fatalf("launch should continue after warning, got err: %v (stderr=%s)", err, stderr.String())
	}
	got := stderr.String()
	for _, want := range []string{"codex-app", "max_output_tokens", "65536", "profile default"} {
		if !strings.Contains(got, want) {
			t.Errorf("stderr %q should contain %q", got, want)
		}
	}
}

// TestLaunchDryRun_CodexWarningOnStderrOnly 驗證 dry-run 的 unsupported
// warning 分流：stderr 有 warning，stdout 命令不含 warning 文字。
func TestLaunchDryRun_CodexWarningOnStderrOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, `profiles:
  - name: openai-official
    provider: openai
    api_base: https://api.openai.com/v1
    api_key: sk-xxxx
    default_model_limits:
      max_output_tokens: 32768
    models: [gpt-4o]
default_profile: openai-official
`)
	var stdout, stderr bytes.Buffer
	opt := launchOptions{}
	if err := runLaunchDryRun(path, "", "codex", "", opt, nil, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchDryRun: %v (stderr=%s)", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "max_output_tokens") {
		t.Errorf("stderr should contain unsupported warning, got: %s", stderr.String())
	}
	if strings.Contains(stdout.String(), "max_output_tokens") || strings.Contains(stdout.String(), "警告") {
		t.Errorf("stdout command must not contain warning text, got: %s", stdout.String())
	}
}
