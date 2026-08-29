package runner

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

func TestBuildCodexArgs_EnvCarriesAPIKey(t *testing.T) {
	t.Setenv("BYOK_TEST_VAR", "hello")
	profile := config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-codex-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	env, _ := BuildCodexArgs(&profile, "gpt-4o", nil, nil)
	if got := getEnv(t, env, "BYOK_CODEX_API_KEY"); got != "sk-codex-test" {
		t.Errorf("BYOK_CODEX_API_KEY = %q, want %q", got, "sk-codex-test")
	}
	if !slices.Contains(env, "BYOK_TEST_VAR=hello") {
		t.Errorf("env missing preserved var BYOK_TEST_VAR=hello; got %v", env)
	}
}

func TestBuildCodexArgs_OverwritesExistingAPIKey(t *testing.T) {
	t.Setenv("BYOK_CODEX_API_KEY", "old-key")
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "new-key",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	env, _ := BuildCodexArgs(&profile, "gpt-4o", nil, nil)
	if got := getEnv(t, env, "BYOK_CODEX_API_KEY"); got != "new-key" {
		t.Errorf("BYOK_CODEX_API_KEY = %q, want %q", got, "new-key")
	}
	if slices.Contains(env, "BYOK_CODEX_API_KEY=old-key") {
		t.Errorf("env should not contain old BYOK_CODEX_API_KEY, got %v", env)
	}
}

func TestBuildCodexArgs_ConfigArgsShape(t *testing.T) {
	profile := config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-xxxx",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	_, configArgs := BuildCodexArgs(&profile, "gpt-4o", nil, nil)

	// configArgs 為成對的 ["--config", "<key>=<value>", ...]
	want := []string{
		"--config", `model="gpt-4o"`,
		"--config", `model_provider="byok"`,
		"--config", `model_providers.byok.name="BYOK"`,
		"--config", `model_providers.byok.base_url="https://api.openai.com/v1"`,
		"--config", `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`,
	}
	if len(configArgs) != len(want) {
		t.Fatalf("configArgs len = %d, want %d: %v", len(configArgs), len(want), configArgs)
	}
	for i, w := range want {
		if configArgs[i] != w {
			t.Errorf("configArgs[%d] = %q, want %q", i, configArgs[i], w)
		}
	}
}

func TestBuildCodexArgs_ModelOverride(t *testing.T) {
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-xxxx",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	_, configArgs := BuildCodexArgs(&profile, "gemma4", nil, nil)
	wantModel := `model="gemma4"`
	if !slices.Contains(configArgs, wantModel) {
		t.Errorf("configArgs missing %q, got %v", wantModel, configArgs)
	}
	if slices.Contains(configArgs, `model="gpt-4o"`) {
		t.Errorf("configArgs should not contain default model, got %v", configArgs)
	}
}

func TestBuildCodexArgs_ConfigArgsAreFlagPairs(t *testing.T) {
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "k",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	_, configArgs := BuildCodexArgs(&profile, "gpt-4o", nil, nil)
	for i := 0; i < len(configArgs); i += 2 {
		if configArgs[i] != "--config" {
			t.Errorf("configArgs[%d] = %q, want \"--config\" (must be flag pairs)", i, configArgs[i])
		}
		if !strings.Contains(configArgs[i+1], "=") {
			t.Errorf("configArgs[%d] = %q, expected key=value form", i+1, configArgs[i+1])
		}
	}
}

func codexTokenProfile() *config.Profile {
	return &config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-codex-token-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
}

// TestBuildCodexArgs_ContextWindowOverride 驗證 effective context 直接映射為
// 未加引號的整數 --config model_context_window=<value>，位置緊接在五個
// provider 覆寫之後。
func TestBuildCodexArgs_ContextWindowOverride(t *testing.T) {
	limits := &TokenLimits{ContextWindowTokens: ptrLmt64(272000)}
	_, configArgs := BuildCodexArgs(codexTokenProfile(), "gpt-5.4", limits, nil)

	want := []string{
		"--config", `model="gpt-5.4"`,
		"--config", `model_provider="byok"`,
		"--config", `model_providers.byok.name="BYOK"`,
		"--config", `model_providers.byok.base_url="https://api.openai.com/v1"`,
		"--config", `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`,
		"--config", `model_context_window=272000`,
	}
	if len(configArgs) != len(want) {
		t.Fatalf("configArgs len = %d, want %d: %v", len(configArgs), len(want), configArgs)
	}
	for i, w := range want {
		if configArgs[i] != w {
			t.Errorf("configArgs[%d] = %q, want %q", i, configArgs[i], w)
		}
	}
}

// TestBuildCodexArgs_ContextWindowBeforeEffort 驗證 context pair 位於
// env_key 覆寫之後、model_reasoning_effort 之前。
func TestBuildCodexArgs_ContextWindowBeforeEffort(t *testing.T) {
	limits := &TokenLimits{ContextWindowTokens: ptrLmt64(272000)}
	_, configArgs := BuildCodexArgs(codexTokenProfile(), "gpt-5.4", limits, nil, "high")

	envKeyIdx := slices.Index(configArgs, `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`)
	ctxIdx := slices.Index(configArgs, `model_context_window=272000`)
	effortIdx := slices.Index(configArgs, `model_reasoning_effort="high"`)
	if envKeyIdx < 0 || ctxIdx < 0 || effortIdx < 0 {
		t.Fatalf("missing expected overrides: envKey=%d ctx=%d effort=%d in %v", envKeyIdx, ctxIdx, effortIdx, configArgs)
	}
	if configArgs[ctxIdx-1] != "--config" {
		t.Errorf("context value at %d must be paired with --config, got %q", ctxIdx, configArgs[ctxIdx-1])
	}
	if !(envKeyIdx < ctxIdx && ctxIdx < effortIdx) {
		t.Errorf("context pair must be after env_key(%d) and before effort(%d); ctx value at %d: %v", envKeyIdx, effortIdx, ctxIdx, configArgs)
	}
}

// TestBuildCodexArgs_UnsetContextOmitted 驗證 context 未設定（limits nil、
// 空欄位、僅 output）時不注入任何 model_context_window。
func TestBuildCodexArgs_UnsetContextOmitted(t *testing.T) {
	cases := []struct {
		name   string
		limits *TokenLimits
	}{
		{"nil limits", nil},
		{"empty limits", &TokenLimits{}},
		{"output-only limits", &TokenLimits{MaxOutputTokens: ptrLmt64(32768)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, configArgs := BuildCodexArgs(codexTokenProfile(), "gpt-4o", tc.limits, nil)
			if slices.ContainsFunc(configArgs, func(a string) bool { return strings.Contains(a, "model_context_window") }) {
				t.Errorf("unset context must not appear in config args, got %v", configArgs)
			}
		})
	}
}

// TestBuildCodexArgs_ContextWithOutputIgnored 驗證 context 與 output 同時有效
// 時：context 以未加引號整數注入一次，output 完全被忽略（呼叫端的 warning
// contract 負責提示，不在此注入），模型字串原樣保留、不改寫。
func TestBuildCodexArgs_ContextWithOutputIgnored(t *testing.T) {
	limits := &TokenLimits{ContextWindowTokens: ptrLmt64(1000000), MaxOutputTokens: ptrLmt64(128000)}
	_, configArgs := BuildCodexArgs(codexTokenProfile(), "gpt-5.4", limits, nil)

	count := 0
	for _, a := range configArgs {
		if strings.Contains(a, "max_output") {
			t.Errorf("output override must be ignored entirely, got %q in %v", a, configArgs)
		}
		if strings.Contains(a, "model_context_window") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one model_context_window arg, got %d: %v", count, configArgs)
	}
	if !slices.Contains(configArgs, `model_context_window=1000000`) {
		t.Errorf("context must be plain integer 1000000, got %v", configArgs)
	}
	if !slices.Contains(configArgs, `model="gpt-5.4"`) {
		t.Errorf("model must pass through unchanged (no rewriting), got %v", configArgs)
	}
}

// TestLaunchCodex_ContextWindowArgsOrder 驗證 LaunchCodex 以 limits 啟動 stub
// 時命令列順序：5 對 provider --config → model_context_window → extraArgs。
func TestLaunchCodex_ContextWindowArgsOrder(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	profile := codexTokenProfile()
	limits := &TokenLimits{ContextWindowTokens: ptrLmt64(272000)}

	var stdout, stderr strings.Builder
	if err := LaunchCodex(profile, "gemma4", limits, stub, []string{"--yolo", "exec"}, nil, &stdout, &stderr, nil); err != nil {
		t.Fatalf("LaunchCodex failed: %v (stderr=%s)", err, stderr.String())
	}

	argsData, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args: %v", err)
	}
	got := splitArgs(string(argsData))
	want := []string{
		"--config", `model="gemma4"`,
		"--config", `model_provider="byok"`,
		"--config", `model_providers.byok.name="BYOK"`,
		"--config", `model_providers.byok.base_url="https://api.openai.com/v1"`,
		"--config", `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`,
		"--config", `model_context_window=272000`,
		"--yolo", "exec",
	}
	if len(got) != len(want) {
		t.Fatalf("child args len = %d, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("child arg[%d] = %q, want %q", i, got[i], w)
		}
	}
}
