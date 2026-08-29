package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// ptrClaude64 回傳 v 的指標，供測試建構 TokenLimits。
func ptrClaude64(v int64) *int64 { return &v }

// countEntries 計算 env 中鍵名為 key 的項目數（前綴 "KEY=" 判定），
// 用於驗證 byok 管理變數的去重與完全省略。
func countEntries(env []string, key string) int {
	prefix := key + "="
	count := 0
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			count++
		}
	}
	return count
}

// TestBuildClaudeEnv_OverridesByokVars 驗證 BuildClaudeEnv 正確注入三個
// ANTHROPIC_* 環境變數且保留其他變數，滿足 "Launch Claude with BYOK profile"。
// 模型字串原樣傳遞：不自動附加 [1m]（Breaking：舊行為自動附加）。
func TestBuildClaudeEnv_OverridesByokVars(t *testing.T) {
	t.Setenv("BYOK_TEST_VAR", "hello")
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-claude-test",
		Models:   []config.Model{{Name: "claude-sonnet-4-5"}},
	}
	env := BuildClaudeEnv(&profile, "claude-sonnet-4-5", nil, nil)

	if got := getEnv(t, env, "ANTHROPIC_BASE_URL"); got != "https://api.openai.com/v1" {
		t.Errorf("ANTHROPIC_BASE_URL = %q, want %q", got, "https://api.openai.com/v1")
	}
	if got := getEnv(t, env, "ANTHROPIC_API_KEY"); got != "sk-claude-test" {
		t.Errorf("ANTHROPIC_API_KEY = %q, want %q", got, "sk-claude-test")
	}
	if got := getEnv(t, env, "ANTHROPIC_MODEL"); got != "claude-sonnet-4-5" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q (passed through verbatim, no [1m])", got, "claude-sonnet-4-5")
	}
	if !contains(env, "BYOK_TEST_VAR=hello") {
		t.Errorf("env missing preserved var BYOK_TEST_VAR=hello; got %v", env)
	}
}

// TestBuildClaudeEnv_ModelOverride 驗證傳入的 model 字串覆寫候選模型，
// 且不附加 [1m] 後綴。
func TestBuildClaudeEnv_ModelOverride(t *testing.T) {
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []config.Model{{Name: "claude-sonnet-4-5"}},
	}
	env := BuildClaudeEnv(&profile, "claude-opus-4-1", nil, nil)

	if got := getEnv(t, env, "ANTHROPIC_MODEL"); got != "claude-opus-4-1" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q (verbatim)", got, "claude-opus-4-1")
	}
	if contains(env, "ANTHROPIC_MODEL=claude-sonnet-4-5") {
		t.Errorf("env should not contain default model, got %v", env)
	}
	if contains(env, "ANTHROPIC_MODEL=claude-opus-4-1[1m]") {
		t.Errorf("env should not contain [1m]-suffixed ANTHROPIC_MODEL, got %v", env)
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
		Models:   []config.Model{{Name: "new-model"}},
	}
	env := BuildClaudeEnv(&profile, "new-model", nil, nil)

	if got := getEnv(t, env, "ANTHROPIC_MODEL"); got != "new-model" {
		t.Errorf("ANTHROPIC_MODEL = %q, want %q (verbatim override)", got, "new-model")
	}
	if contains(env, "ANTHROPIC_MODEL=old-model") {
		t.Errorf("env should not contain old ANTHROPIC_MODEL, got %v", env)
	}
	if contains(env, "ANTHROPIC_MODEL=new-model[1m]") {
		t.Errorf("env should not contain [1m]-suffixed ANTHROPIC_MODEL, got %v", env)
	}
}

// TestBuildClaudeEnv_PlainModelPassthrough 驗證普通模型名稱原樣傳遞，
// 不自動附加 [1m]（Breaking：舊行為為自動附加）。
func TestBuildClaudeEnv_PlainModelPassthrough(t *testing.T) {
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []config.Model{{Name: "claude-sonnet-4-5"}},
	}
	env := BuildClaudeEnv(&profile, "claude-sonnet-4-5", nil, nil)

	if got := getEnv(t, env, "ANTHROPIC_MODEL"); got != "claude-sonnet-4-5" {
		t.Errorf("ANTHROPIC_MODEL = %q, want exactly %q", got, "claude-sonnet-4-5")
	}
}

// TestBuildClaudeEnv_Explicit1mSuffixPreserved 驗證顯式 [1m] 後綴原樣保留，
// 不重複附加。
func TestBuildClaudeEnv_Explicit1mSuffixPreserved(t *testing.T) {
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []config.Model{{Name: "claude-sonnet-4-5[1m]"}},
	}
	env := BuildClaudeEnv(&profile, "claude-sonnet-4-5[1m]", nil, nil)

	if got := getEnv(t, env, "ANTHROPIC_MODEL"); got != "claude-sonnet-4-5[1m]" {
		t.Errorf("ANTHROPIC_MODEL = %q, want exactly %q (explicit suffix preserved)", got, "claude-sonnet-4-5[1m]")
	}
}

// TestBuildClaudeEnv_TokenLimitsMappedToClaudeEnv 驗證 Claude model token limit
// mapping：effective context → CLAUDE_CODE_MAX_CONTEXT_TOKENS、
// effective output → CLAUDE_CODE_MAX_OUTPUT_TOKENS（十進位字串）。
func TestBuildClaudeEnv_TokenLimitsMappedToClaudeEnv(t *testing.T) {
	profile := config.Profile{Name: "p", Provider: "openai", APIBase: "https://api.openai.com/v1", APIKey: "sk-test", Models: []config.Model{{Name: "claude-sonnet-4-5"}}}
	limits := &TokenLimits{ContextWindowTokens: ptrClaude64(1000000), MaxOutputTokens: ptrClaude64(128000)}
	env := BuildClaudeEnv(&profile, "claude-sonnet-4-5", limits, nil)

	if got := getEnv(t, env, "CLAUDE_CODE_MAX_CONTEXT_TOKENS"); got != "1000000" {
		t.Errorf("CLAUDE_CODE_MAX_CONTEXT_TOKENS = %q, want 1000000", got)
	}
	if got := getEnv(t, env, "CLAUDE_CODE_MAX_OUTPUT_TOKENS"); got != "128000" {
		t.Errorf("CLAUDE_CODE_MAX_OUTPUT_TOKENS = %q, want 128000", got)
	}
}

// TestBuildClaudeEnv_PartialTokenLimits 驗證僅設定 context 時只注入對應變數，
// unset 的 output 欄位不注入（以前綴掃描確認整個環境無對應項目）。
func TestBuildClaudeEnv_PartialTokenLimits(t *testing.T) {
	profile := config.Profile{Name: "p", Provider: "openai", APIBase: "https://api.openai.com/v1", APIKey: "sk-test", Models: []config.Model{{Name: "claude-sonnet-4-5"}}}
	limits := &TokenLimits{ContextWindowTokens: ptrClaude64(272000)}
	env := BuildClaudeEnv(&profile, "claude-sonnet-4-5", limits, nil)

	if got := getEnv(t, env, "CLAUDE_CODE_MAX_CONTEXT_TOKENS"); got != "272000" {
		t.Errorf("CLAUDE_CODE_MAX_CONTEXT_TOKENS = %q, want 272000", got)
	}
	if n := countEntries(env, "CLAUDE_CODE_MAX_OUTPUT_TOKENS"); n != 0 {
		t.Errorf("unset output limit must be omitted entirely; found %d entries", n)
	}
}

// TestBuildClaudeEnv_NilLimitsOmitsTokenVars 驗證 limits 為 nil 或欄位皆 nil 時
// 兩個 token 變數皆不出現在子程序環境。
func TestBuildClaudeEnv_NilLimitsOmitsTokenVars(t *testing.T) {
	profile := config.Profile{Name: "p", Provider: "openai", APIBase: "https://api.openai.com/v1", APIKey: "sk-test", Models: []config.Model{{Name: "claude-sonnet-4-5"}}}

	env := BuildClaudeEnv(&profile, "claude-sonnet-4-5", nil, nil)
	if n := countEntries(env, "CLAUDE_CODE_MAX_CONTEXT_TOKENS"); n != 0 {
		t.Errorf("nil limits: %d CLAUDE_CODE_MAX_CONTEXT_TOKENS entries, want 0", n)
	}
	if n := countEntries(env, "CLAUDE_CODE_MAX_OUTPUT_TOKENS"); n != 0 {
		t.Errorf("nil limits: %d CLAUDE_CODE_MAX_OUTPUT_TOKENS entries, want 0", n)
	}

	env = BuildClaudeEnv(&profile, "claude-sonnet-4-5", &TokenLimits{}, nil)
	if n := countEntries(env, "CLAUDE_CODE_MAX_CONTEXT_TOKENS"); n != 0 {
		t.Errorf("nil ContextWindowTokens field: %d entries, want 0", n)
	}
	if n := countEntries(env, "CLAUDE_CODE_MAX_OUTPUT_TOKENS"); n != 0 {
		t.Errorf("nil MaxOutputTokens field: %d entries, want 0", n)
	}
}

// TestBuildClaudeEnv_DedupesClaudeTokenEnv 驗證子程序環境去重：父環境的同名
// 管理鍵被移除，僅在 byok 有值時以 byok 值唯一回填。
func TestBuildClaudeEnv_DedupesClaudeTokenEnv(t *testing.T) {
	t.Setenv("CLAUDE_CODE_MAX_CONTEXT_TOKENS", "999")
	profile := config.Profile{Name: "p", Provider: "openai", APIBase: "https://api.openai.com/v1", APIKey: "sk-test", Models: []config.Model{{Name: "claude-sonnet-4-5"}}}

	// byok 未設定 → 鍵必須自環境完全移除（不殘留父環境值）。
	env := BuildClaudeEnv(&profile, "claude-sonnet-4-5", nil, nil)
	if n := countEntries(env, "CLAUDE_CODE_MAX_CONTEXT_TOKENS"); n != 0 {
		t.Errorf("unset limit with stale parent value: %d entries with prefix, want 0", n)
	}

	// byok 有值 → 恰一個項目且為 byok 值。
	limits := &TokenLimits{ContextWindowTokens: ptrClaude64(500000)}
	env = BuildClaudeEnv(&profile, "claude-sonnet-4-5", limits, nil)
	if n := countEntries(env, "CLAUDE_CODE_MAX_CONTEXT_TOKENS"); n != 1 {
		t.Errorf("expected exactly 1 CLAUDE_CODE_MAX_CONTEXT_TOKENS entry, got %d: %v", n, env)
	}
	if got := getEnv(t, env, "CLAUDE_CODE_MAX_CONTEXT_TOKENS"); got != "500000" {
		t.Errorf("CLAUDE_CODE_MAX_CONTEXT_TOKENS = %q, want 500000 (byok value wins)", got)
	}
}

// TestLaunchClaude_ByokVarsInjected 驗證 LaunchClaude 以真實 profile 啟動 stub，
// stub 接收到三個 ANTHROPIC_* 環境變數（模型原樣傳遞），同時父程序環境保持不變。
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
		Models:   []config.Model{{Name: "claude-sonnet-4-5"}},
	}

	var stdout, stderr strings.Builder
	if err := LaunchClaude(profile, "claude-opus-4-1", nil, stub, []string{"--dangerously-skip-permissions"}, nil, &stdout, &stderr, nil); err != nil {
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
		"ANTHROPIC_MODEL":    "claude-opus-4-1",
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

// TestLaunchClaude_TokenVarsInjectedParentEnvUnchanged 驗證 limits 經 LaunchClaude
// 抵達子程序環境為兩個 token 變數，父環境的陳舊值被去重，且父程序環境不變。
func TestLaunchClaude_TokenVarsInjectedParentEnvUnchanged(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")

	// 父環境預先存在陳舊值，驗證去重。
	t.Setenv("CLAUDE_CODE_MAX_CONTEXT_TOKENS", "999")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_PARENT_MARKER", "before")

	parentBefore := snapshotEnv()

	profile := &config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-claude-integration",
		Models:   []config.Model{{Name: "claude-sonnet-4-5"}},
	}
	limits := &TokenLimits{ContextWindowTokens: ptrClaude64(272000), MaxOutputTokens: ptrClaude64(16384)}

	var stdout, stderr strings.Builder
	if err := LaunchClaude(profile, "claude-sonnet-4-5", limits, stub, nil, nil, &stdout, &stderr, nil); err != nil {
		t.Fatalf("LaunchClaude failed: %v (stderr=%s)", err, stderr.String())
	}

	parentAfter := snapshotEnv()
	if !envEqual(parentBefore, parentAfter) {
		t.Fatalf("parent environment changed after launch (token vars must only reach the child)\nbefore:\n%s\nafter:\n%s",
			strings.Join(parentBefore, "\n"), strings.Join(parentAfter, "\n"))
	}
	if got := os.Getenv("CLAUDE_CODE_MAX_CONTEXT_TOKENS"); got != "999" {
		t.Fatalf("parent CLAUDE_CODE_MAX_CONTEXT_TOKENS = %q, want %q (parent env must be untouched)", got, "999")
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read stub env output: %v", err)
	}
	childEnv := strings.Split(string(data), "\n")

	if n := countEntries(childEnv, "CLAUDE_CODE_MAX_CONTEXT_TOKENS"); n != 1 {
		t.Errorf("expected exactly 1 child CLAUDE_CODE_MAX_CONTEXT_TOKENS entry, got %d", n)
	}
	if got := envLookup(childEnv, "CLAUDE_CODE_MAX_CONTEXT_TOKENS"); got != "272000" {
		t.Errorf("child CLAUDE_CODE_MAX_CONTEXT_TOKENS = %q, want 272000 (stale 999 must be deduped)", got)
	}
	if n := countEntries(childEnv, "CLAUDE_CODE_MAX_OUTPUT_TOKENS"); n != 1 {
		t.Errorf("expected exactly 1 child CLAUDE_CODE_MAX_OUTPUT_TOKENS entry, got %d", n)
	}
	if got := envLookup(childEnv, "CLAUDE_CODE_MAX_OUTPUT_TOKENS"); got != "16384" {
		t.Errorf("child CLAUDE_CODE_MAX_OUTPUT_TOKENS = %q, want 16384", got)
	}
	if contains(childEnv, "CLAUDE_CODE_MAX_CONTEXT_TOKENS=999") {
		t.Errorf("stale parent value 999 must not reach child env: %v", childEnv)
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
		Models:   []config.Model{{Name: "claude-sonnet-4-5"}},
	}

	var stdout, stderr strings.Builder
	if err := LaunchClaude(profile, "claude-sonnet-4-5", nil, stub, nil, nil, &stdout, &stderr, nil); err != nil {
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
