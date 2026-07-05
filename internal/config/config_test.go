package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigStructYAMLTags(t *testing.T) {
	src := `profiles:
  - name: openai-official
    provider: openai
    api_base: https://api.openai.com/v1
    api_key: sk-xxxx
    default_model: gpt-4o
default_profile: openai-official
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg.Profiles))
	}
	p := cfg.Profiles[0]
	if p.Name != "openai-official" {
		t.Errorf("Name = %q, want %q", p.Name, "openai-official")
	}
	if p.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", p.Provider, "openai")
	}
	if p.APIBase != "https://api.openai.com/v1" {
		t.Errorf("APIBase = %q, want %q", p.APIBase, "https://api.openai.com/v1")
	}
	if p.APIKey != "sk-xxxx" {
		t.Errorf("APIKey = %q, want %q", p.APIKey, "sk-xxxx")
	}
	if p.DefaultModel != "gpt-4o" {
		t.Errorf("DefaultModel = %q, want %q", p.DefaultModel, "gpt-4o")
	}
	if cfg.DefaultProfile != "openai-official" {
		t.Errorf("DefaultProfile = %q, want %q", cfg.DefaultProfile, "openai-official")
	}
}

// errContainsPath 回報 err 是否提及 path，比對時使用正規化
// （正斜線）路徑，避免 Go 在 Windows 上以 %q 反斜線跳脫造成誤判。
func errContainsPath(t *testing.T, err error, path string) bool {
	t.Helper()
	got := filepath.ToSlash(err.Error())
	want := filepath.ToSlash(path)
	return strings.Contains(got, want)
}

func TestLoad_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "config.yaml")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !errContainsPath(t, err, path) {
		t.Errorf("error %q does not contain path %q", err.Error(), path)
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	bad := []byte("profiles:\n  - name: x\n    broken: [unbalanced")
	if err := os.WriteFile(path, bad, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
	if !errContainsPath(t, err, path) {
		t.Errorf("error %q does not contain path %q", err.Error(), path)
	}
}

func TestSave_CreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".byok", "config.yaml")
	cfg := &Config{
		Profiles: []Profile{
			{
				Name:     "openai-official",
				APIKey:   "sk-xxxx",
				Provider: "openai",
			},
		},
		DefaultProfile: "openai-official",
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(loaded.Profiles))
	}
	if loaded.Profiles[0].Name != "openai-official" {
		t.Errorf("Name = %q, want %q", loaded.Profiles[0].Name, "openai-official")
	}
	if loaded.Profiles[0].APIKey != "sk-xxxx" {
		t.Errorf("APIKey = %q, want %q", loaded.Profiles[0].APIKey, "sk-xxxx")
	}
	if loaded.DefaultProfile != "openai-official" {
		t.Errorf("DefaultProfile = %q, want %q", loaded.DefaultProfile, "openai-official")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath failed: %v", err)
	}
	want := filepath.Join(".byok", "config.yaml")
	if !strings.HasSuffix(path, want) {
		t.Errorf("path %q does not end with %q", path, want)
	}
}
