package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// TestRunLaunchClaude_MissingConfigFile 驗證設定檔不存在時印出提示並 exit 1。
func TestRunLaunchClaude_MissingConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	var stdout, stderr bytes.Buffer
	err := runLaunchClaude(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "找不到設定檔") {
		t.Errorf("stderr missing '找不到設定檔', got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "byok config add") {
		t.Errorf("stderr missing hint to run `byok config add`, got: %s", stderr.String())
	}
}

// TestRunLaunchClaude_MissingProfile 驗證具名 profile 不存在時列出可用 profile 並 exit 1。
func TestRunLaunchClaude_MissingProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchClaude(path, "nonexistent", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), `找不到 profile "nonexistent"`) {
		t.Errorf("stderr missing not-found message, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "openai-official") {
		t.Errorf("stderr should list available profiles, got: %s", stderr.String())
	}
}

// TestRunLaunchClaude_NonOpenaiProviderRejected 驗證非 openai provider 被拒。
func TestRunLaunchClaude_NonOpenaiProviderRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: azure-prod\n    provider: azure\n    api_base: https://example.openai.azure.com\n    api_key: az-key\n    default_model: gpt-4o\ndefault_profile: azure-prod\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchClaude(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "僅支援 openai provider") {
		t.Errorf("stderr missing provider rejection, got: %s", stderr.String())
	}
}

// TestRunLaunchClaude_NoDefaultProfile 驗證未指定 profile 且未設定 default_profile 時 exit 1。
func TestRunLaunchClaude_NoDefaultProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchClaude(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "default_profile") {
		t.Errorf("stderr missing default_profile hint, got: %s", stderr.String())
	}
}

// TestRunLaunchClaude_NotInstalled 驗證 claude 不在 PATH 時印出安裝提示並 exit 1。
func TestRunLaunchClaude_NotInstalled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	t.Setenv("PATH", "")
	var stdout, stderr bytes.Buffer
	err := runLaunchClaude(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), `找不到 "claude" 可執行檔`) {
		t.Errorf("stderr missing claude not-found message, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "安裝 Claude Code") {
		t.Errorf("stderr missing install hint, got: %s", stderr.String())
	}
}

// TestRunLaunchClaude_ParentEnvUnchanged 驗證錯誤路徑（claude 未安裝）後
// 父程序環境保持不變。
func TestRunLaunchClaude_ParentEnvUnchanged(t *testing.T) {
	t.Setenv("BYOK_PARENT_CHECK", "intact")
	t.Setenv("ANTHROPIC_API_KEY", "")
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	t.Setenv("PATH", "")
	var stdout, stderr bytes.Buffer
	_ = runLaunchClaude(path, "", "", nil, &stdout, &stderr)
	if got := envLookupOS("BYOK_PARENT_CHECK"); got != "intact" {
		t.Errorf("parent BYOK_PARENT_CHECK = %q, want %q", got, "intact")
	}
	if got := envLookupOS("ANTHROPIC_API_KEY"); got != "" {
		t.Errorf("parent ANTHROPIC_API_KEY leaked = %q (must be empty)", got)
	}
}

// TestRunLaunchClaude_EnvInjected 驗證正常啟動路徑下子程序接收正確的
// ANTHROPIC_BASE_URL / ANTHROPIC_API_KEY / ANTHROPIC_MODEL。
func TestRunLaunchClaude_EnvInjected(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-claude-test\n    default_model: claude-sonnet-4-5\ndefault_profile: openai-official\n")

	stubExe := buildStubForClaude(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	if err := runLaunchClaude(path, "", "", nil, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchClaude failed: %v (stderr=%s)", err, stderr.String())
	}

	envData, _ := os.ReadFile(outFile)
	childEnv := strings.Split(string(envData), "\n")

	if got := envLookupOSIn(childEnv, "ANTHROPIC_BASE_URL"); got != "https://api.openai.com/v1" {
		t.Errorf("child ANTHROPIC_BASE_URL = %q, want https://api.openai.com/v1", got)
	}
	if got := envLookupOSIn(childEnv, "ANTHROPIC_API_KEY"); got != "sk-claude-test" {
		t.Errorf("child ANTHROPIC_API_KEY = %q, want sk-claude-test", got)
	}
	if got := envLookupOSIn(childEnv, "ANTHROPIC_MODEL"); got != "claude-sonnet-4-5" {
		t.Errorf("child ANTHROPIC_MODEL = %q, want claude-sonnet-4-5", got)
	}
}

// TestRunLaunchClaude_ModelOverride 驗證 --model 覆寫 default_model。
func TestRunLaunchClaude_ModelOverride(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-claude-test\n    default_model: claude-sonnet-4-5\ndefault_profile: openai-official\n")

	stubExe := buildStubForClaude(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	if err := runLaunchClaude(path, "", "claude-opus-4-1", nil, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchClaude failed: %v (stderr=%s)", err, stderr.String())
	}

	envData, _ := os.ReadFile(outFile)
	childEnv := strings.Split(string(envData), "\n")
	if got := envLookupOSIn(childEnv, "ANTHROPIC_MODEL"); got != "claude-opus-4-1" {
		t.Errorf("child ANTHROPIC_MODEL = %q, want claude-opus-4-1", got)
	}
}

// TestRunLaunchClaude_ExtraArgs 驗證 extraArgs（含 --dangerously-skip-permissions）
// 原樣轉發給 runner.LaunchClaude。
func TestRunLaunchClaude_ExtraArgs(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-claude-test\n    default_model: claude-sonnet-4-5\ndefault_profile: openai-official\n")

	stubExe := buildStubForClaude(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	extraArgs := []string{"--dangerously-skip-permissions", "review this"}
	if err := runLaunchClaude(path, "", "", extraArgs, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchClaude failed: %v (stderr=%s)", err, stderr.String())
	}

	argsData, _ := os.ReadFile(argsFile)
	got := splitNonEmpty(string(argsData))
	want := []string{"--dangerously-skip-permissions", "review this"}
	if len(got) != len(want) {
		t.Fatalf("child args len = %d, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("child arg[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// buildStubForClaude 編譯 testdata stub 至暫存目錄，命名為 claude
// （Windows 加 .exe），供 runLaunchClaude 的 LookPath("claude") 解析。
func buildStubForClaude(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "claude")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", exe, ".")
	cmd.Dir = filepath.Join("..", "internal", "runner", "testdata", "stub")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build stub: %v\n%s", err, out)
	}
	return exe
}