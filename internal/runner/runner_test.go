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
		Models:   []string{"gpt-4o"},
	}
	env := BuildEnv(&profile, "gpt-4o")

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
		Models:   []string{"gpt-4o"},
	}
	env := BuildEnv(&profile, "gemma4")

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
		Models:   []string{"gpt-4o"},
	}
	env := BuildEnv(&profile, "gpt-4o")

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
		Models:   []string{"gpt-4o"},
	}
	env := BuildEnv(&profile, "gpt-4o")

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
		Models:   []string{"new-model"},
	}
	env := BuildEnv(&profile, "new-model")

	if got := getEnv(t, env, "COPILOT_MODEL"); got != "new-model" {
		t.Errorf("COPILOT_MODEL = %q, want %q", got, "new-model")
	}
	if slices.Contains(env, "COPILOT_MODEL=old-model") {
		t.Errorf("env should not contain COPILOT_MODEL=old-model, got %v", env)
	}
}