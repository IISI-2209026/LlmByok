package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// writeFile 是測試用的小型輔助函式。
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestMaskAPIKey(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"sk-abcdefghijklmnopqrstuvwxyz1234567890", "sk-a...7890"},
		{"sk-1234", "sk-1...1234"}, // 7 個字元：前 4 + ... + 後 4 重疊
		{"", ""},
		{"abc", "abc"}, // 過短無法遮罩，原樣回傳
	}
	for _, c := range cases {
		if got := maskAPIKey(c.in); got != c.want {
			t.Errorf("maskAPIKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestConfigAdd_NewProfileCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".byok", "config.yaml")
	var out bytes.Buffer
	if err := runConfigAdd(path, "openai-official", "openai",
		"https://api.openai.com/v1", "sk-xxxx", "gpt-4o", &out); err != nil {
		t.Fatalf("runConfigAdd: %v", err)
	}
	if !strings.Contains(out.String(), "已新增 profile") {
		t.Errorf("output missing confirmation, got: %s", out.String())
	}
	// 驗證檔案已建立並包含該設定檔且設為預設值。
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "openai-official") {
		t.Errorf("config missing profile name, got: %s", data)
	}
	if !strings.Contains(string(data), "default_profile: openai-official") {
		t.Errorf("config did not set default_profile, got: %s", data)
	}
}

func TestConfigAdd_DuplicateNameErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	before, _ := os.ReadFile(path)
	var out bytes.Buffer
	err := runConfigAdd(path, "openai-official", "openai",
		"https://api.openai.com/v1", "sk-yyyy", "gpt-4o", &out)
	if err == nil {
		t.Fatalf("expected error for duplicate name, got nil (output: %s)", out.String())
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("config file was modified on duplicate-add error\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestConfigList_Output(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-abcdefghijklmnopqrstuvwxyz1234567890\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	var out bytes.Buffer
	if err := runConfigList(path, &out); err != nil {
		t.Fatalf("runConfigList: %v", err)
	}
	if !strings.Contains(out.String(), "openai-official") {
		t.Errorf("output missing profile name, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "sk-a...7890") {
		t.Errorf("output missing masked api key, got: %s", out.String())
	}
}

func TestConfigRemove_Existing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\n  - name: local-ollama\n    provider: openai\n    api_base: http://localhost:11434\n    api_key: \"\"\n    default_model: llama3.2\ndefault_profile: local-ollama\n")
	var out bytes.Buffer
	if err := runConfigRemove(path, "local-ollama", &out); err != nil {
		t.Fatalf("runConfigRemove: %v", err)
	}
	data, _ := os.ReadFile(path)
	s := string(data)
	if strings.Contains(s, "local-ollama") {
		t.Errorf("removed profile still present:\n%s", s)
	}
	// default_profile 原為 local-ollama -> 應被清除。
	if strings.Contains(s, "default_profile: local-ollama") {
		t.Errorf("default_profile not cleared after removing default:\n%s", s)
	}
}

func TestConfigRemove_NotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	before, _ := os.ReadFile(path)
	var out bytes.Buffer
	err := runConfigRemove(path, "nonexistent", &out)
	if err == nil {
		t.Fatalf("expected error for missing profile, got nil")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("config modified on missing-profile error")
	}
}

func TestSetDefault_Existing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\n  - name: local-ollama\n    provider: openai\n    api_base: http://localhost:11434\n    api_key: \"\"\n    default_model: llama3.2\ndefault_profile: openai-official\n")
	var out bytes.Buffer
	if err := runConfigSetDefault(path, "local-ollama", &out); err != nil {
		t.Fatalf("runConfigSetDefault: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "default_profile: local-ollama") {
		t.Errorf("default_profile not updated, got: %s", data)
	}
}

func TestSetDefault_NotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	var out bytes.Buffer
	err := runConfigSetDefault(path, "nonexistent", &out)
	if err == nil {
		t.Fatalf("expected error for missing profile, got nil")
	}
}