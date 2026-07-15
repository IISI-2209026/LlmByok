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
	got := renderLaunchDryRun("codex", p, "gpt-5", launchOptions{effort: "high"}, []string{"--yolo", "exec"})
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
	got := renderLaunchDryRun("claude", p, "sonnet", launchOptions{effort: "high", subModel: "claude-haiku-4-5"}, nil)
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
	got := renderLaunchDryRun("pi", p, "gpt-5", launchOptions{effort: "high"}, []string{"--approve"})
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
