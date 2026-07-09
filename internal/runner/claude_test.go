package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// TestBuildClaudeEnv_OverridesByokVars 驗證 BuildClaudeEnv 正確注入三個
// ANTHROPIC_* 環境變數且保留其他變數，滿足 "Launch Claude with BYOK profile"。
func TestBuildClaudeEnv_OverridesByokVars(t *testing.T) {
	t.Setenv("BYOK_TEST_VAR", "hello")
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-claude-test",
		Models:   []string{"claude-sonnet-4-5"},
	}
	env := BuildClaudeEnv(&profile, "claude-sonnet-4-5")

	if got := getEnv(t, env, "ANTHROPIC_BASE_URL"); got != "https://api.openai.com/v1" {
		t.Errorf("ANTHROPIC_BASE_URL = %q, want %q", got, "https://api.openai.com/v1")
	}
	if got := getEnv(t, env, "ANTHROPIC_API_KEY"); got != "sk-claude-test" {
		t.Errorf("ANTHROPIC_API_KEY = %q, want %q", got, "sk-claude-test")
	}
	if got := getEnv(t, env, "ANTHROPIC_MODEL"); got != "claude-sonnet-4-5[1m]" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q", got, "claude-sonnet-4-5[1m]")
	}
	if !contains(env, "BYOK_TEST_VAR=hello") {
		t.Errorf("env missing preserved var BYOK_TEST_VAR=hello; got %v", env)
	}
}

// TestBuildClaudeEnv_ModelOverride 驗證傳入的 model 字串覆寫候選模型。
func TestBuildClaudeEnv_ModelOverride(t *testing.T) {
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []string{"claude-sonnet-4-5"},
	}
	env := BuildClaudeEnv(&profile, "claude-opus-4-1")

	if got := getEnv(t, env, "ANTHROPIC_MODEL"); got != "claude-opus-4-1[1m]" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q", got, "claude-opus-4-1[1m]")
	}
	if contains(env, "ANTHROPIC_MODEL=claude-sonnet-4-5") {
		t.Errorf("env should not contain default model, got %v", env)
	}
}

// TestBuildClaudeEnv_OverwritesExistingByokVar 驗證既存的 ANTHROPIC_* 值被覆寫，
// 滿足 "Parent process environment unchanged for claude"。
func TestBuildClaudeEnv_OverwritesExistingByokVar(t *testing.T) {
	t.Setenv("ANTHROPIC_MODEL", "old-model")
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []string{"new-model"},
	}
	env := BuildClaudeEnv(&profile, "new-model")

	if got := getEnv(t, env, "ANTHROPIC_MODEL"); got != "new-model[1m]" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q", got, "new-model[1m]")
	}
	if contains(env, "ANTHROPIC_MODEL=old-model") {
		t.Errorf("env should not contain old ANTHROPIC_MODEL, got %v", env)
	}
	if contains(env, "ANTHROPIC_MODEL=new-model") {
		t.Errorf("env should not contain unsuffixed ANTHROPIC_MODEL, got %v", env)
	}
}

// TestLaunchClaude_ByokVarsInjected 驗證 LaunchClaude 以真實 profile 啟動 stub，
// stub 接收到三個 ANTHROPIC_* 環境變數，同時父程序環境保持不變。
func TestLaunchClaude_ByokVarsInjected(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")

	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)
	t.Setenv("BYOK_PARENT_MARKER", "before")

	parentBefore := snapshotEnv()

	profile := &config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-claude-integration",
		Models:   []string{"claude-sonnet-4-5"},
	}

	var stdout, stderr strings.Builder
	if err := LaunchClaude(profile, "claude-opus-4-1", stub, []string{"--dangerously-skip-permissions"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("LaunchClaude failed: %v (stderr=%s)", err, stderr.String())
	}

	// 父程序環境必須保持不變。
	parentAfter := snapshotEnv()
	if !envEqual(parentBefore, parentAfter) {
		t.Fatalf("parent environment changed after launch\nbefore:\n%s\nafter:\n%s",
			strings.Join(parentBefore, "\n"), strings.Join(parentAfter, "\n"))
	}
	if got := os.Getenv("BYOK_PARENT_MARKER"); got != "before" {
		t.Fatalf("BYOK_PARENT_MARKER = %q, want %q (parent env must be untouched)", got, "before")
	}

	// 子程序環境必須包含三個 ANTHROPIC_* 變數。
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read stub env output: %v", err)
	}
	childEnv := strings.Split(string(data), "\n")

	want := map[string]string{
		"ANTHROPIC_BASE_URL": "https://api.openai.com/v1",
		"ANTHROPIC_API_KEY":  "sk-claude-integration",
		"ANTHROPIC_MODEL":    "claude-opus-4-1[1m]",
	}
	for key, expected := range want {
		got := envLookup(childEnv, key)
		if got != expected {
			t.Errorf("child %s = %q, want %q", key, got, expected)
		}
	}

	// 命令列參數必須包含 --dangerously-skip-permissions。
	argsData, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args output: %v", err)
	}
	gotArgs := splitArgs(string(argsData))
	wantArgs := []string{"--dangerously-skip-permissions"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("child args len = %d, want %d: %v", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i, w := range wantArgs {
		if gotArgs[i] != w {
			t.Errorf("child arg[%d] = %q, want %q", i, gotArgs[i], w)
		}
	}
}

// TestLaunchClaude_NoExtraArgs 驗證不傳 extraArgs 時子程序收到零命令列參數。
func TestLaunchClaude_NoExtraArgs(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	argsFile := filepath.Join(t.TempDir(), "args.txt")
	outFile := filepath.Join(t.TempDir(), "env.txt")

	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	profile := &config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-claude-integration",
		Models:   []string{"claude-sonnet-4-5"},
	}

	var stdout, stderr strings.Builder
	if err := LaunchClaude(profile, "claude-sonnet-4-5", stub, nil, nil, &stdout, &stderr); err != nil {
		t.Fatalf("LaunchClaude failed: %v (stderr=%s)", err, stderr.String())
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args output: %v", err)
	}
	if len(strings.TrimSpace(string(data))) > 0 {
		t.Errorf("expected zero args, got: %s", string(data))
	}
}

// contains 判斷 env 切片是否包含指定項目。
func contains(env []string, s string) bool {
	for _, e := range env {
		if e == s {
			return true
		}
	}
	return false
}