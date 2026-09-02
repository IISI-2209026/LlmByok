package runner

import (
	"slices"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// getEnv 回傳第一個鍵名相符的環境項目值，找不到則回傳 ""。
// 若要檢查完整 "KEY=VALUE" 是否存在，請使用 slices.Contains。
func getEnv(t *testing.T, env []string, key string) string {
	t.Helper()
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return strings.TrimPrefix(e, prefix)
		}
	}
	return ""
}

func TestBuildEnv_OverridesByokVars(t *testing.T) {
	t.Setenv("BYOK_TEST_VAR", "hello")
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	env := BuildEnv(&profile, "gpt-4o", nil, nil)

	if got := getEnv(t, env, "COPILOT_PROVIDER_BASE_URL"); got != "https://api.openai.com/v1" {
		t.Errorf("COPILOT_PROVIDER_BASE_URL = %q, want %q", got, "https://api.openai.com/v1")
	}
	if got := getEnv(t, env, "COPILOT_PROVIDER_TYPE"); got != "openai" {
		t.Errorf("COPILOT_PROVIDER_TYPE = %q, want %q", got, "openai")
	}
	if got := getEnv(t, env, "COPILOT_PROVIDER_API_KEY"); got != "sk-test" {
		t.Errorf("COPILOT_PROVIDER_API_KEY = %q, want %q", got, "sk-test")
	}
	if got := getEnv(t, env, "COPILOT_MODEL"); got != "gpt-4o" {
		t.Errorf("COPILOT_MODEL = %q, want %q", got, "gpt-4o")
	}
	if !slices.Contains(env, "BYOK_TEST_VAR=hello") {
		t.Errorf("env missing preserved var BYOK_TEST_VAR=hello; got %v", env)
	}
}

func TestBuildEnv_ModelOverride(t *testing.T) {
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	env := BuildEnv(&profile, "gemma4", nil, nil)

	if got := getEnv(t, env, "COPILOT_MODEL"); got != "gemma4" {
		t.Errorf("COPILOT_MODEL = %q, want %q", got, "gemma4")
	}
	if slices.Contains(env, "COPILOT_MODEL=gpt-4o") {
		t.Errorf("env should not contain COPILOT_MODEL=gpt-4o, got %v", env)
	}
}

func TestBuildEnv_EmptyProviderDefaultsOpenai(t *testing.T) {
	profile := config.Profile{
		Name:     "p",
		Provider: "",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	env := BuildEnv(&profile, "gpt-4o", nil, nil)

	if got := getEnv(t, env, "COPILOT_PROVIDER_TYPE"); got != "openai" {
		t.Errorf("COPILOT_PROVIDER_TYPE = %q, want %q (default)", got, "openai")
	}
}

func TestBuildEnv_PreservesOtherVars(t *testing.T) {
	t.Setenv("MY_CUSTOM_VAR", "keepme")
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	env := BuildEnv(&profile, "gpt-4o", nil, nil)

	if !slices.Contains(env, "MY_CUSTOM_VAR=keepme") {
		t.Errorf("env missing preserved var MY_CUSTOM_VAR=keepme; got %v", env)
	}
}

func TestBuildEnv_OverwritesExistingByokVar(t *testing.T) {
	t.Setenv("COPILOT_MODEL", "old-model")
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []config.Model{{Name: "new-model"}},
	}
	env := BuildEnv(&profile, "new-model", nil, nil)

	if got := getEnv(t, env, "COPILOT_MODEL"); got != "new-model" {
		t.Errorf("COPILOT_MODEL = %q, want %q", got, "new-model")
	}
	if slices.Contains(env, "COPILOT_MODEL=old-model") {
		t.Errorf("env should not contain COPILOT_MODEL=old-model, got %v", env)
	}
}

// ptrLmt64 回傳 v 的指標，供測試建構 TokenLimits。
func ptrLmt64(v int64) *int64 { return &v }

// TestBuildEnv_TokenLimitsMappedToPromptAndOutputEnv 驗證 Copilot mapping：
// effective context 直接映射 COPILOT_PROVIDER_MAX_PROMPT_TOKENS、
// effective output 映射 COPILOT_PROVIDER_MAX_OUTPUT_TOKENS。
func TestBuildEnv_TokenLimitsMappedToPromptAndOutputEnv(t *testing.T) {
	profile := config.Profile{Name: "p", Provider: "openai", APIBase: "https://x", APIKey: "sk", Models: []config.Model{{Name: "gpt-4o"}}}
	limits := &TokenLimits{ContextWindowTokens: ptrLmt64(1000000), MaxOutputTokens: ptrLmt64(128000)}
	env := BuildEnv(&profile, "gpt-4o", limits, nil)

	if got := getEnv(t, env, "COPILOT_PROVIDER_MAX_PROMPT_TOKENS"); got != "1000000" {
		t.Errorf("COPILOT_PROVIDER_MAX_PROMPT_TOKENS = %q, want 1000000 (direct mapping)", got)
	}
	if got := getEnv(t, env, "COPILOT_PROVIDER_MAX_OUTPUT_TOKENS"); got != "128000" {
		t.Errorf("COPILOT_PROVIDER_MAX_OUTPUT_TOKENS = %q, want 128000", got)
	}
}

// TestBuildEnv_CopilotMappingNoArithmetic 驗證 prompt 映射不做 context-output 算術。
func TestBuildEnv_CopilotMappingNoArithmetic(t *testing.T) {
	profile := config.Profile{Name: "p", Provider: "openai", APIBase: "https://x", APIKey: "sk", Models: []config.Model{{Name: "gpt-4o"}}}
	limits := &TokenLimits{ContextWindowTokens: ptrLmt64(200000), MaxOutputTokens: ptrLmt64(64000)}
	env := BuildEnv(&profile, "gpt-4o", limits, nil)

	if got := getEnv(t, env, "COPILOT_PROVIDER_MAX_PROMPT_TOKENS"); got != "200000" {
		t.Errorf("COPILOT_PROVIDER_MAX_PROMPT_TOKENS = %q, want exactly 200000 (no subtraction)", got)
	}
}

// TestBuildEnv_UnsetTokenLimitsOmitted 驗證未設定時不再注入固定 token 值。
func TestBuildEnv_UnsetTokenLimitsOmitted(t *testing.T) {
	t.Setenv("COPILOT_PROVIDER_MAX_PROMPT_TOKENS", "999")
	t.Setenv("COPILOT_PROVIDER_MAX_OUTPUT_TOKENS", "999")
	profile := config.Profile{Name: "p", Provider: "openai", APIBase: "https://x", APIKey: "sk", Models: []config.Model{{Name: "gpt-4o"}}}

	env := BuildEnv(&profile, "gpt-4o", nil, nil)
	for _, entry := range env {
		if strings.HasPrefix(entry, "COPILOT_PROVIDER_MAX_PROMPT_TOKENS=") || strings.HasPrefix(entry, "COPILOT_PROVIDER_MAX_OUTPUT_TOKENS=") {
			t.Errorf("unset token limits must be omitted (no hard-coded defaults); got %q", entry)
		}
	}

	// 單一欄位設定：只注入該欄位。
	partial := &TokenLimits{ContextWindowTokens: ptrLmt64(272000)}
	env = BuildEnv(&profile, "gpt-4o", partial, nil)
	if got := getEnv(t, env, "COPILOT_PROVIDER_MAX_PROMPT_TOKENS"); got != "272000" {
		t.Errorf("COPILOT_PROVIDER_MAX_PROMPT_TOKENS = %q, want 272000", got)
	}
	if slices.ContainsFunc(env, func(e string) bool { return strings.HasPrefix(e, "COPILOT_PROVIDER_MAX_OUTPUT_TOKENS=") }) {
		t.Errorf("unset output must be omitted, got %v", env)
	}
}

// TestBuildEnv_DedupesParentTokenEnv 驗證由 byok 管理的 token 環境變數在子程序
// 環境中不出現重複項：父環境值被移除，僅保留 byok 解析值（若有）。
func TestBuildEnv_DedupesParentTokenEnv(t *testing.T) {
	t.Setenv("COPILOT_PROVIDER_MAX_PROMPT_TOKENS", "999")
	profile := config.Profile{Name: "p", Provider: "openai", APIBase: "https://x", APIKey: "sk", Models: []config.Model{{Name: "gpt-4o"}}}

	// byok 未設定 → 父環境的管理鍵被移除且不回填。
	env := BuildEnv(&profile, "gpt-4o", nil, nil)
	count := 0
	for _, e := range env {
		if strings.HasPrefix(e, "COPILOT_PROVIDER_MAX_PROMPT_TOKENS=") {
			count++
		}
	}
	if count != 0 {
		t.Errorf("expected 0 byok-managed prompt token entries, got %d", count)
	}

	// byok 有值時唯一存在且為 byok 值。
	limits := &TokenLimits{ContextWindowTokens: ptrLmt64(500000)}
	env = BuildEnv(&profile, "gpt-4o", limits, nil)
	count = 0
	for _, e := range env {
		if strings.HasPrefix(e, "COPILOT_PROVIDER_MAX_PROMPT_TOKENS=") {
			count++
		}
	}
	if count != 1 || getEnv(t, env, "COPILOT_PROVIDER_MAX_PROMPT_TOKENS") != "500000" {
		t.Errorf("expected exactly one byok value 500000, got %d entries", count)
	}
}
