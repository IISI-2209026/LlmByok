package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

func TestSetModels_MultipleViaFlags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\ndefault_profile: openai-official\n")
	var out bytes.Buffer
	if err := runSetModels(path, "openai-official", []string{"gpt-4o", "gpt-4o-mini"}, &out); err != nil {
		t.Fatalf("runSetModels: %v", err)
	}
	cfg, _ := config.Load(path)
	if len(cfg.Profiles[0].Models) != 2 || cfg.Profiles[0].Models[0] != "gpt-4o" || cfg.Profiles[0].Models[1] != "gpt-4o-mini" {
		t.Errorf("Models = %v, want [gpt-4o gpt-4o-mini]", cfg.Profiles[0].Models)
	}
}

func TestSetModels_ReplacesExistingList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    models:\n      - gpt-4o\n      - gpt-4o-mini\ndefault_profile: openai-official\n")
	var out bytes.Buffer
	if err := runSetModels(path, "openai-official", []string{"gpt-4o"}, &out); err != nil {
		t.Fatalf("runSetModels: %v", err)
	}
	cfg, _ := config.Load(path)
	if len(cfg.Profiles[0].Models) != 1 || cfg.Profiles[0].Models[0] != "gpt-4o" {
		t.Errorf("Models = %v, want [gpt-4o] (full replace)", cfg.Profiles[0].Models)
	}
}

func TestSetModels_EmptyListRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    models:\n      - gpt-4o\ndefault_profile: openai-official\n")
	before, _ := os.ReadFile(path)
	var out bytes.Buffer
	err := runSetModels(path, "openai-official", nil, &out)
	if err == nil {
		t.Fatal("expected error for empty model list, got nil")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("config modified on empty-list error")
	}
}

func TestSetModels_NonExistentProfileRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n  - name: local-ollama\n    provider: openai\n    api_base: http://localhost:11434\n    api_key: \"\"\ndefault_profile: openai-official\n")
	before, _ := os.ReadFile(path)
	var out bytes.Buffer
	err := runSetModels(path, "nonexistent", []string{"gpt-4o"}, &out)
	if err == nil {
		t.Fatal("expected error for non-existent profile, got nil")
	}
	if !strings.Contains(err.Error(), "openai-official") || !strings.Contains(err.Error(), "local-ollama") {
		t.Errorf("error should list available profiles, got: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("config modified on non-existent-profile error")
	}
}

func TestSetModels_InteractiveCollectsUntilEmptyLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\ndefault_profile: openai-official\n")
	// 逐行輸入 gpt-4o、gpt-4o-mini，再以空行結束。
	in := strings.NewReader("gpt-4o\ngpt-4o-mini\n\n")
	var out bytes.Buffer
	models, err := promptModels(in, &out)
	if err != nil {
		t.Fatalf("promptModels: %v", err)
	}
	if len(models) != 2 || models[0] != "gpt-4o" || models[1] != "gpt-4o-mini" {
		t.Errorf("collected = %v, want [gpt-4o gpt-4o-mini]", models)
	}
	if err := runSetModels(path, "openai-official", models, &out); err != nil {
		t.Fatalf("runSetModels: %v", err)
	}
	cfg, _ := config.Load(path)
	if len(cfg.Profiles[0].Models) != 2 || cfg.Profiles[0].Models[0] != "gpt-4o" {
		t.Errorf("Models = %v, want [gpt-4o gpt-4o-mini]", cfg.Profiles[0].Models)
	}
}

func TestSetModels_InteractiveNonTTYRejected(t *testing.T) {
	orig := isTerminal
	t.Cleanup(func() { isTerminal = orig })
	isTerminal = func(int) bool { return false }
	cmd := newSetModelsCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"openai-official"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for interactive mode in non-TTY, got nil")
	}
}

func TestSetModels_MissingPositionalNameRejected(t *testing.T) {
	cmd := newSetModelsCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for missing positional profile name, got nil")
	}
}

// TestSetModels_RegisteredUnderConfig 驗證 set-models 為 config 的子指令（非頂層指令）。
func TestSetModels_RegisteredUnderConfig(t *testing.T) {
	root := NewRoot("test")
	// 確認 set-models 不是頂層指令。
	for _, c := range root.Commands() {
		if c.Name() == "set-models" {
			t.Fatal("set-models should NOT be a top-level command")
		}
	}
	// 確認它是 config 的子指令。
	cfgCmd, _, err := root.Find([]string{"config"})
	if err != nil {
		t.Fatalf("find config: %v", err)
	}
	found := false
	for _, c := range cfgCmd.Commands() {
		if c.Name() == "set-models" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("set-models not registered as a subcommand of config")
	}
}