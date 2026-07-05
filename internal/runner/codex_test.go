package runner

import (
	"slices"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

func TestBuildCodexArgs_EnvCarriesAPIKey(t *testing.T) {
	t.Setenv("BYOK_TEST_VAR", "hello")
	profile := config.Profile{
		Name:         "openai-official",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey:       "sk-codex-test",
		DefaultModel: "gpt-4o",
	}
	env, _ := BuildCodexArgs(&profile, "")
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
		Name:         "p",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey:       "new-key",
		DefaultModel: "gpt-4o",
	}
	env, _ := BuildCodexArgs(&profile, "")
	if got := getEnv(t, env, "BYOK_CODEX_API_KEY"); got != "new-key" {
		t.Errorf("BYOK_CODEX_API_KEY = %q, want %q", got, "new-key")
	}
	if slices.Contains(env, "BYOK_CODEX_API_KEY=old-key") {
		t.Errorf("env should not contain old BYOK_CODEX_API_KEY, got %v", env)
	}
}

func TestBuildCodexArgs_ConfigArgsShape(t *testing.T) {
	profile := config.Profile{
		Name:         "openai-official",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey:       "sk-xxxx",
		DefaultModel: "gpt-4o",
	}
	_, configArgs := BuildCodexArgs(&profile, "")

	// configArgs 為成對的 ["--config", "<key>=<value>", ...]
	want := []string{
		"--config", `model="gpt-4o"`,
		"--config", `model_provider="byok"`,
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
		Name:         "p",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey:       "sk-xxxx",
		DefaultModel: "gpt-4o",
	}
	_, configArgs := BuildCodexArgs(&profile, "gemma4")
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
		Name:         "p",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey:       "k",
		DefaultModel: "gpt-4o",
	}
	_, configArgs := BuildCodexArgs(&profile, "")
	for i := 0; i < len(configArgs); i += 2 {
		if configArgs[i] != "--config" {
			t.Errorf("configArgs[%d] = %q, want \"--config\" (must be flag pairs)", i, configArgs[i])
		}
		if !strings.Contains(configArgs[i+1], "=") {
			t.Errorf("configArgs[%d] = %q, expected key=value form", i+1, configArgs[i+1])
		}
	}
}