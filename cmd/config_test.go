package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
	"github.com/IISI-2209026/LlmByok/internal/secret"
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
		{"sk-1234", "sk-1...1234"},
		{"", ""},
		{"abc", "abc"},
	}
	for _, c := range cases {
		if got := maskAPIKey(c.in); got != c.want {
			t.Errorf("maskAPIKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// --- Add 參數模式 ---

func TestConfigAdd_NewProfileCreatesFile(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), ".byok", "config.yaml")
	var out bytes.Buffer
	if err := runConfigAdd(path, "openai-official", "openai",
		"https://api.openai.com/v1", "sk-xxxx", "gpt-4o", "keychain", &out); err != nil {
		t.Fatalf("runConfigAdd: %v", err)
	}
	if !strings.Contains(out.String(), "已新增 profile") {
		t.Errorf("output missing confirmation, got: %s", out.String())
	}
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
	// keychain 模式：config 不應包含明碼金鑰
	if strings.Contains(string(data), "sk-xxxx") {
		t.Errorf("config should not contain plaintext api_key in keychain mode, got: %s", data)
	}
	// keychain 中應有金鑰
	stored, err := secret.Load("openai-official")
	if err != nil {
		t.Fatalf("secret.Load: %v", err)
	}
	if stored != "sk-xxxx" {
		t.Errorf("keychain value = %q, want sk-xxxx", stored)
	}
}

func TestConfigAdd_DuplicateNameErrors(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	before, _ := os.ReadFile(path)
	var out bytes.Buffer
	err := runConfigAdd(path, "openai-official", "openai",
		"https://api.openai.com/v1", "sk-yyyy", "gpt-4o", "keychain", &out)
	if err == nil {
		t.Fatalf("expected error for duplicate name, got nil (output: %s)", out.String())
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("config file was modified on duplicate-add error\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestConfigAdd_KeyStoragePlaintextClearsKeychain(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	// 先在 keychain 放一個既有金鑰
	if err := secret.Store("my-profile", "sk-old"); err != nil {
		t.Fatalf("secret.Store: %v", err)
	}
	var out bytes.Buffer
	if err := runConfigAdd(path, "my-profile", "openai",
		"https://api.openai.com/v1", "sk-new", "gpt-4o", "plaintext", &out); err != nil {
		t.Fatalf("runConfigAdd: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "sk-new") {
		t.Errorf("config should contain plaintext api_key in plaintext mode, got: %s", data)
	}
	// keychain 中的舊金鑰應已被刪除
	_, err := secret.Load("my-profile")
	if err == nil {
		t.Error("keychain entry should have been deleted in plaintext mode")
	}
}

func TestConfigAdd_KeychainBackendUnavailableFails(t *testing.T) {
	orig := keys
	t.Cleanup(func() { keys = orig })
	keys = failKeyStore{storeErr: secret.ErrBackendUnavailable}
	path := filepath.Join(t.TempDir(), "config.yaml")
	var out bytes.Buffer
	err := runConfigAdd(path, "p", "openai", "https://x", "sk-key", "m", "keychain", &out)
	if err == nil {
		t.Fatalf("expected error for keychain backend unavailable, got nil")
	}
	if !strings.Contains(err.Error(), "plaintext") {
		t.Errorf("error should suggest --key-storage plaintext, got: %v", err)
	}
	// config 不應被寫入
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Errorf("config should not be written when keychain fails")
	}
}

func TestConfigAdd_EmptyApiKeyNoKeyHandling(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	var out bytes.Buffer
	if err := runConfigAdd(path, "nokey", "openai", "https://x", "", "m", "keychain", &out); err != nil {
		t.Fatalf("runConfigAdd: %v", err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "api_key") && strings.Contains(string(data), "api_key: ") {
		// api_key should be empty/absent
		s := string(data)
		if strings.Contains(s, "api_key: sk") {
			t.Errorf("config should not contain api_key value, got: %s", data)
		}
	}
}

func TestConfigAdd_FirstProfileBecomesDefault(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	var out bytes.Buffer
	if err := runConfigAdd(path, "first", "openai", "https://x", "sk-k", "m", "keychain", &out); err != nil {
		t.Fatalf("runConfigAdd: %v", err)
	}
	if !strings.Contains(out.String(), "預設") {
		t.Errorf("output should mention default, got: %s", out.String())
	}
}

// --- Add 互動模式 ---

func TestConfigAdd_InteractivePromptsAllFields(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	input := strings.NewReader("interactive-profile\nopenai\nhttps://api.test.com/v1\ngpt-4o-mini\nsk-inter-key\nkeychain\n")
	var out bytes.Buffer
	p := &config.Prompter{In: input, Out: &out, IsTTY: func(int) bool { return false }}
	if err := runConfigAddInteractive(path, "keychain", p, &out); err != nil {
		t.Fatalf("runConfigAddInteractive: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "interactive-profile") {
		t.Errorf("config missing profile name, got: %s", data)
	}
	stored, err := secret.Load("interactive-profile")
	if err != nil {
		t.Fatalf("secret.Load: %v", err)
	}
	if stored != "sk-inter-key" {
		t.Errorf("keychain = %q, want sk-inter-key", stored)
	}
}

func TestConfigAdd_InteractiveNonTTYFails(t *testing.T) {
	orig := isTerminal
	t.Cleanup(func() { isTerminal = orig })
	isTerminal = func(int) bool { return false }
	cmd := newConfigAddCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for interactive mode in non-TTY, got nil")
	}
}

// --- Update 參數模式 ---

func TestConfigUpdate_KeepsUnspecifiedFields(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: p\n    provider: openai\n    api_base: https://old.com\n    default_model: old-model\ndefault_profile: p\n")
	var out bytes.Buffer
	if err := runConfigUpdate(path, "p", nil, nil, nil, "", false, "keychain", &out); err != nil {
		t.Fatalf("runConfigUpdate: %v", err)
	}
	cfg, _ := config.Load(path)
	prof := cfg.Profiles[0]
	if prof.Provider != "openai" || prof.APIBase != "https://old.com" || prof.DefaultModel != "old-model" {
		t.Errorf("fields changed unexpectedly: %+v", prof)
	}
}

func TestConfigUpdate_OverwritesSpecifiedFields(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: p\n    provider: openai\n    api_base: https://old.com\n    default_model: old-model\ndefault_profile: p\n")
	var out bytes.Buffer
	provider := "openai"
	apiBase := "https://new.com"
	if err := runConfigUpdate(path, "p", &provider, &apiBase, nil, "", false, "keychain", &out); err != nil {
		t.Fatalf("runConfigUpdate: %v", err)
	}
	cfg, _ := config.Load(path)
	prof := cfg.Profiles[0]
	if prof.Provider != "openai" || prof.APIBase != "https://new.com" {
		t.Errorf("fields not updated: %+v", prof)
	}
	if prof.DefaultModel != "old-model" {
		t.Errorf("DefaultModel should be unchanged: %+v", prof)
	}
}

func TestConfigUpdate_ApiKeyIntoKeychain(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: p\n    provider: openai\n    api_base: https://x\n    default_model: m\ndefault_profile: p\n")
	var out bytes.Buffer
	if err := runConfigUpdate(path, "p", nil, nil, nil, "sk-update-key", true, "keychain", &out); err != nil {
		t.Fatalf("runConfigUpdate: %v", err)
	}
	stored, err := secret.Load("p")
	if err != nil {
		t.Fatalf("secret.Load: %v", err)
	}
	if stored != "sk-update-key" {
		t.Errorf("keychain = %q, want sk-update-key", stored)
	}
	cfg, _ := config.Load(path)
	if cfg.Profiles[0].APIKey != "" {
		t.Errorf("config api_key should be empty in keychain mode, got: %q", cfg.Profiles[0].APIKey)
	}
}

func TestConfigUpdate_ApiKeyClearByKey(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	// 先存 keychain 金鑰
	if err := secret.Store("p", "sk-old"); err != nil {
		t.Fatalf("secret.Store: %v", err)
	}
	writeFile(t, path, "profiles:\n  - name: p\n    provider: openai\n    api_base: https://x\n    api_key: \"\"\n    default_model: m\ndefault_profile: p\n")
	var out bytes.Buffer
	if err := runConfigUpdate(path, "p", nil, nil, nil, "", true, "keychain", &out); err != nil {
		t.Fatalf("runConfigUpdate: %v", err)
	}
	// keychain 應被清除
	_, err2 := secret.Load("p")
	if err2 == nil {
		t.Error("keychain entry should be cleared")
	}
}

func TestConfigUpdate_NonExistentRejected(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: p\n    provider: openai\n    api_base: https://x\n    default_model: m\ndefault_profile: p\n")
	var out bytes.Buffer
	err := runConfigUpdate(path, "nonexistent", nil, nil, nil, "", false, "keychain", &out)
	if err == nil {
		t.Fatal("expected error for non-existent profile, got nil")
	}
}

func TestConfigUpdate_NonExistentConfigRejected(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	var out bytes.Buffer
	err := runConfigUpdate(path, "p", nil, nil, nil, "", false, "keychain", &out)
	if err == nil {
		t.Fatal("expected error for non-existent config, got nil")
	}
}

// --- Update 互動模式 ---

func TestConfigUpdate_InteractiveUpdatesFields(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: p\n    provider: openai\n    api_base: https://old.com\n    default_model: old-model\ndefault_profile: p\n")
	input := strings.NewReader("openai\nhttps://new.com\nnew-model\nsk-new-key\nkeychain\n")
	var out bytes.Buffer
	p := &config.Prompter{In: input, Out: &out, IsTTY: func(int) bool { return false }}
	if err := runConfigUpdateInteractive(path, "p", "keychain", p, &out); err != nil {
		t.Fatalf("runConfigUpdateInteractive: %v", err)
	}
	cfg, _ := config.Load(path)
	prof := cfg.Profiles[0]
	if prof.APIBase != "https://new.com" || prof.DefaultModel != "new-model" {
		t.Errorf("fields not updated: %+v", prof)
	}
	stored, err := secret.Load("p")
	if err != nil {
		t.Fatalf("secret.Load: %v", err)
	}
	if stored != "sk-new-key" {
		t.Errorf("keychain = %q, want sk-new-key", stored)
	}
}

func TestConfigUpdate_InteractiveKeepsExistingKeyOnEmpty(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := secret.Store("p", "sk-existing"); err != nil {
		t.Fatalf("secret.Store: %v", err)
	}
	writeFile(t, path, "profiles:\n  - name: p\n    provider: openai\n    api_base: https://x\n    default_model: m\ndefault_profile: p\n")
	input := strings.NewReader("openai\nhttps://x\nm\n\nkeychain\n")
	var out bytes.Buffer
	p := &config.Prompter{In: input, Out: &out, IsTTY: func(int) bool { return false }}
	if err := runConfigUpdateInteractive(path, "p", "keychain", p, &out); err != nil {
		t.Fatalf("runConfigUpdateInteractive: %v", err)
	}
	stored, err := secret.Load("p")
	if err != nil {
		t.Fatalf("secret.Load: %v", err)
	}
	if stored != "sk-existing" {
		t.Errorf("keychain = %q, want sk-existing (should be preserved)", stored)
	}
}

func TestConfigUpdate_InteractiveNonExistentRejected(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: other\n    provider: openai\n    api_base: https://x\n    default_model: m\ndefault_profile: other\n")
	input := strings.NewReader("\n\n\n\nkeychain\n")
	var out bytes.Buffer
	p := &config.Prompter{In: input, Out: &out, IsTTY: func(int) bool { return false }}
	err := runConfigUpdateInteractive(path, "nonexistent", "keychain", p, &out)
	if err == nil {
		t.Fatal("expected error for non-existent profile, got nil")
	}
}

// --- Delete ---

func TestConfigDelete_SyncsKeychain(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := secret.Store("doomed", "sk-doomed"); err != nil {
		t.Fatalf("secret.Store: %v", err)
	}
	writeFile(t, path, "profiles:\n  - name: doomed\n    provider: openai\n    api_base: https://x\n    api_key: \"\"\n    default_model: m\ndefault_profile: doomed\n")
	var out bytes.Buffer
	if err := runConfigDelete(path, "doomed", &out); err != nil {
		t.Fatalf("runConfigDelete: %v", err)
	}
	if !strings.Contains(out.String(), "keychain") {
		t.Errorf("output should mention keychain cleanup, got: %s", out.String())
	}
	_, err := secret.Load("doomed")
	if err == nil {
		t.Error("keychain entry should be deleted")
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "doomed") {
		t.Errorf("profile still in config: %s", data)
	}
	if strings.Contains(string(data), "default_profile: doomed") {
		t.Errorf("default_profile not cleared: %s", data)
	}
}

func TestConfigDelete_NoKeychainEntrySucceeds(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: nokc\n    provider: openai\n    api_base: https://x\n    default_model: m\ndefault_profile: nokc\n")
	var out bytes.Buffer
	if err := runConfigDelete(path, "nokc", &out); err != nil {
		t.Fatalf("runConfigDelete: %v", err)
	}
	// Should not mention keychain deletion (no entry to delete)
	if strings.Contains(out.String(), "已自 keychain 刪除") {
		t.Errorf("should not report keychain deletion when no entry exists, got: %s", out.String())
	}
}

func TestConfigDelete_NonExistentRejected(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: p\n    provider: openai\n    api_base: https://x\n    default_model: m\ndefault_profile: p\n")
	before, _ := os.ReadFile(path)
	var out bytes.Buffer
	err := runConfigDelete(path, "nonexistent", &out)
	if err == nil {
		t.Fatal("expected error for missing profile, got nil")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("config modified on missing-profile error")
	}
}

func TestConfigDelete_KeychainFailureWarnsButExits0(t *testing.T) {
	orig := keys
	t.Cleanup(func() { keys = orig })
	keys = failKeyStore{deleteErr: fmt.Errorf("db locked")}
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: p\n    provider: openai\n    api_base: https://x\n    default_model: m\ndefault_profile: p\n")
	var out bytes.Buffer
	if err := runConfigDelete(path, "p", &out); err != nil {
		t.Fatalf("delete should succeed (exit 0) even if keychain fails, got: %v", err)
	}
	if !strings.Contains(out.String(), "警告") {
		t.Errorf("output should contain warning, got: %s", out.String())
	}
}

// --- List / SetDefault ---

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

// --- Removed commands ---

func TestConfigRemovedCommands_UnknownCommand(t *testing.T) {
	for _, sub := range []string{"set-key", "del-key", "import-keys"} {
		t.Run(sub, func(t *testing.T) {
			root := newConfigCmd()
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			root.SetArgs([]string{sub})
			err := root.Execute()
			if err == nil {
				t.Fatalf("expected unknown command error for %q, got nil", sub)
			}
		})
	}
}

// --- Helpers ---

// failKeyStore 是測試用 keyStore，可注入 Store/Delete 的失敗。
type failKeyStore struct {
	storeErr   error
	deleteErr  error
	storeCalled bool
}

func (f failKeyStore) Store(profileName, apiKey string) error {
	if f.storeErr != nil {
		return f.storeErr
	}
	return nil
}

func (f failKeyStore) Delete(profileName string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return nil
}

// 確保 errors 套件被使用（failKeyStore 使用 errors.New 於測試中）
var _ = errors.New