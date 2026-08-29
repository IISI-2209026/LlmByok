package cmd

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

func TestRenderLaunchDryRun_CodexMasksKeyAndMapsEffort(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test/v1", APIKey: "real-secret", Provider: "openai"}
	got := renderLaunchDryRun("codex", p, "gpt-5", launchOptions{effort: "high"}, []string{"--yolo", "exec"}, nil, nil)
	if strings.Contains(got, "real-secret") || !strings.Contains(got, "'***'") {
		t.Fatalf("key masking failed: %s", got)
	}
	for _, want := range []string{"codex", "model_reasoning_effort", "high", "--yolo", "exec"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %s", want, got)
		}
	}
}

func TestRenderLaunchDryRun_ClaudeSubModelOnlyClaude(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test", APIKey: "real-secret"}
	got := renderLaunchDryRun("claude", p, "sonnet", launchOptions{effort: "high", subModel: "claude-haiku-4-5"}, nil, nil, nil)
	for _, want := range []string{"CLAUDE_CODE_EFFORT_LEVEL", "CLAUDE_CODE_SUBAGENT_MODEL", "claude-haiku-4-5"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %s", want, got)
		}
	}
}

func TestRunLaunchDryRun_DoesNotResolveKeyOrExecutable(t *testing.T) {
	path := t.TempDir() + "\\config.yaml"
	writeFile(t, path, "profiles:\n  - name: demo\n    provider: openai\n    api_base: https://example.test/v1\n    api_key: real-secret\n    models: [gpt-5]\ndefault_profile: demo\n")
	t.Setenv("PATH", "")
	var stdout, stderr bytes.Buffer
	if err := runLaunchDryRun(path, "", "codex", "", launchOptions{effort: "high"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v (%s)", err, stderr.String())
	}
	if strings.Contains(stdout.String(), "real-secret") || !strings.Contains(stdout.String(), "***") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRenderLaunchDryRun_PiMasksKeyAndRendersTemporaryConfig(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test/v1", APIKey: "real-secret", Provider: "openai"}
	got := renderLaunchDryRun("pi", p, "gpt-5", launchOptions{effort: "high"}, []string{"--approve"}, nil, nil)
	if strings.Contains(got, "real-secret") {
		t.Fatalf("API key leaked in output: %s", got)
	}
	for _, want := range []string{"***", "models.json", "PI_CODING_AGENT_DIR", "pi --model", "gpt-5", "--thinking", "high", "--approve"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %s", want, got)
		}
	}
	if runtime.GOOS == "windows" {
		for _, want := range []string{"Join-Path $env:TEMP", "finally", "Remove-Item"} {
			if !strings.Contains(got, want) {
				t.Errorf("windows output missing %q: %s", want, got)
			}
		}
	} else {
		for _, want := range []string{"mktemp", "trap", "rm -rf"} {
			if !strings.Contains(got, want) {
				t.Errorf("posix output missing %q: %s", want, got)
			}
		}
	}
}

func TestLaunchHelpIncludesOptionalFlags(t *testing.T) {
	cmd := newLaunchCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.Help(); err != nil {
		t.Fatalf("help failed: %v", err)
	}
	text := output.String()
	for _, want := range []string{"--effort", "--sub-model", "--dry-run"} {
		if !strings.Contains(text, want) {
			t.Errorf("help missing %q: %s", want, text)
		}
	}
}

// dryRunTokenLimits 建構帶 token 限制的 resolvedTokenLimits。
func dryRunLimits(ctx, out *int64) *resolvedTokenLimits {
	return &resolvedTokenLimits{ContextWindow: ctx, ContextWindowSource: tokenSourceCLI, MaxOutput: out, MaxOutputSource: tokenSourceCLI}
}

// TestRenderLaunchDryRun_CopilotRendersTokenEnvs 驗證 Copilot dry-run 在
// 平台原生命令中輸出兩個 token 環境變數（POSIX/Windows 皆含子字串）。
func TestRenderLaunchDryRun_CopilotRendersTokenEnvs(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test/v1", APIKey: "sk-real", Provider: "openai", Models: []config.Model{{Name: "gpt-4o"}}}
	got := renderLaunchDryRun("copilot", p, "gpt-4o", launchOptions{}, nil, nil, dryRunLimits(ptrToken(1000000), ptrToken(128000)))
	for _, want := range []string{"COPILOT_PROVIDER_MAX_PROMPT_TOKENS='1000000'", "COPILOT_PROVIDER_MAX_OUTPUT_TOKENS='128000'"} {
		if !strings.Contains(got, want) {
			t.Errorf("copilot dry-run missing %q: %s", want, got)
		}
	}
}

// TestRenderLaunchDryRun_CodexRendersContextConfigOnly 驗證 Codex dry-run
// 渲染 context --config、忽略 output，且 stdout 無 warning 文字。
func TestRenderLaunchDryRun_CodexRendersContextConfigOnly(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test/v1", APIKey: "sk-real", Provider: "openai", Models: []config.Model{{Name: "gpt-4o"}}}
	got := renderLaunchDryRun("codex", p, "gpt-4o", launchOptions{}, nil, nil, dryRunLimits(ptrToken(272000), ptrToken(32768)))
	if !strings.Contains(got, "model_context_window=272000") {
		t.Errorf("codex dry-run missing model_context_window=272000: %s", got)
	}
	if strings.Contains(got, "max_output_tokens") {
		t.Errorf("codex dry-run must not render output override: %s", got)
	}

	// 输出值被忽略：stdout 只有 context config。
	app := renderLaunchDryRun("codex-app", p, "gpt-4o", launchOptions{}, nil, nil, dryRunLimits(ptrToken(272000), ptrToken(32768)))
	if !strings.Contains(app, "codex app") || !strings.Contains(app, "model_context_window=272000") {
		t.Errorf("codex-app dry-run missing context config: %s", app)
	}
	if strings.Index(app, "codex app") > strings.Index(app, "model_context_window") {
		t.Errorf("app must precede context config, got: %s", app)
	}
}

// TestRenderLaunchDryRun_ClaudeRendersTokenEnvs 驗證 Claude dry-run 渲染兩個 token 環境變數。
func TestRenderLaunchDryRun_ClaudeRendersTokenEnvs(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test", APIKey: "sk-real", Provider: "openai", Models: []config.Model{{Name: "m"}}}
	got := renderLaunchDryRun("claude", p, "m", launchOptions{}, nil, nil, dryRunLimits(ptrToken(1000000), ptrToken(128000)))
	for _, want := range []string{"CLAUDE_CODE_MAX_CONTEXT_TOKENS='1000000'", "CLAUDE_CODE_MAX_OUTPUT_TOKENS='128000'"} {
		if !strings.Contains(got, want) {
			t.Errorf("claude dry-run missing %q: %s", want, got)
		}
	}
}

// TestRenderLaunchDryRun_UnsetTokenLimitsAbsent 驗證 unset 時 stdout 不含任何
// token 環境變數或 codex token config。
func TestRenderLaunchDryRun_UnsetTokenLimitsAbsent(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test/v1", APIKey: "sk-real", Provider: "openai", Models: []config.Model{{Name: "m"}}}
	for _, target := range []string{"copilot", "codex", "codex-app", "claude"} {
		got := renderLaunchDryRun(target, p, "m", launchOptions{}, nil, nil, nil)
		for _, banned := range []string{"COPILOT_PROVIDER_MAX_PROMPT_TOKENS", "COPILOT_PROVIDER_MAX_OUTPUT_TOKENS", "CLAUDE_CODE_MAX_CONTEXT_TOKENS", "CLAUDE_CODE_MAX_OUTPUT_TOKENS", "model_context_window"} {
			if strings.Contains(got, banned) {
				t.Errorf("%s dry-run with unset limits must not contain %q: %s", target, banned, got)
			}
		}
	}
}

// TestRenderLaunchDryRun_PartialLimitsOmitUnsetField 驗證單一欄位設定時只渲染該欄位。
func TestRenderLaunchDryRun_PartialLimitsOmitUnsetField(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test/v1", APIKey: "sk-real", Provider: "openai", Models: []config.Model{{Name: "m"}}}
	got := renderLaunchDryRun("copilot", p, "m", launchOptions{}, nil, nil, dryRunLimits(ptrToken(272000), nil))
	if !strings.Contains(got, "COPILOT_PROVIDER_MAX_PROMPT_TOKENS='272000'") {
		t.Errorf("context should render: %s", got)
	}
	if strings.Contains(got, "COPILOT_PROVIDER_MAX_OUTPUT_TOKENS") {
		t.Errorf("unset output must be omitted: %s", got)
	}
}

// TestRenderLaunchDryRun_TokenRenderingNeverLeaksKey 驗證 token 渲染不含 API key。
func TestRenderLaunchDryRun_TokenRenderingNeverLeaksKey(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test/v1", APIKey: "super-secret-key", Provider: "openai", Models: []config.Model{{Name: "m"}}}
	for _, target := range []string{"copilot", "codex", "codex-app", "claude"} {
		got := renderLaunchDryRun(target, p, "m", launchOptions{}, nil, nil, dryRunLimits(ptrToken(1000000), ptrToken(128000)))
		if strings.Contains(got, "super-secret-key") {
			t.Errorf("%s dry-run leaked API key: %s", target, got)
		}
	}
}

// TestRenderLaunchDryRun_PiTokenLifecycle 驗證 Pi dry-run 渲染 masked
// modelOverrides、settings headroom、Pi invocation 與清理片段。
func TestRenderLaunchDryRun_PiTokenLifecycle(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test/v1", APIKey: "sk-real", Provider: "openai", Models: []config.Model{{Name: "gpt-5.4"}}}
	got := renderLaunchDryRun("pi", p, "gpt-5.4", launchOptions{}, nil, nil, dryRunLimits(ptrToken(1000000), ptrToken(128000)))
	for _, want := range []string{
		`"modelOverrides":{"gpt-5.4":{"contextWindow":1000000,"maxTokens":128000}}`,
		`"compaction":{"reserveTokens":128000}`,
		"models.json",
		"settings.json",
		"PI_CODING_AGENT_DIR",
		"pi --model",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("pi dry-run missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "sk-real") {
		t.Errorf("pi dry-run leaked API key: %s", got)
	}
	// 清理片段：平台原生（POSIX rm -rf 或 PowerShell Remove-Item）。
	if !strings.Contains(got, "rm -rf") && !strings.Contains(got, "Remove-Item") {
		t.Errorf("pi dry-run missing cleanup fragment: %s", got)
	}
}

// TestRenderLaunchDryRun_PiContextOnlyNoSettings 驗證 context-only 時不產生
// settings.json / reserveTokens 片段。
func TestRenderLaunchDryRun_PiContextOnlyNoSettings(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test/v1", APIKey: "sk-real", Provider: "openai", Models: []config.Model{{Name: "gpt-5.4-mini"}}}
	got := renderLaunchDryRun("pi", p, "gpt-5.4-mini", launchOptions{}, nil, nil, dryRunLimits(ptrToken(272000), nil))
	if !strings.Contains(got, `"contextWindow":272000`) {
		t.Errorf("pi dry-run missing context override:\n%s", got)
	}
	if strings.Contains(got, "reserveTokens") || strings.Contains(got, "settings.json") {
		t.Errorf("context-only pi dry-run must not create settings fragment:\n%s", got)
	}
}

// TestRenderLaunchDryRun_PiUnsetNoModelOverrides 驗證 fully unset 時不渲染
// modelOverrides/settings 片段，維持 provider-only 設定。
func TestRenderLaunchDryRun_PiUnsetNoModelOverrides(t *testing.T) {
	p := &config.Profile{APIBase: "https://example.test/v1", APIKey: "sk-real", Provider: "openai", Models: []config.Model{{Name: "m"}}}
	got := renderLaunchDryRun("pi", p, "m", launchOptions{}, nil, nil, nil)
	if strings.Contains(got, "modelOverrides") || strings.Contains(got, "settings.json") || strings.Contains(got, "reserveTokens") {
		t.Errorf("unset pi dry-run must be provider-only:\n%s", got)
	}
}
