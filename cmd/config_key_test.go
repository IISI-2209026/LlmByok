package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
	"github.com/IISI-2209026/LlmByok/internal/secret"
	"github.com/zalando/go-keyring"
)

// restoreSecretFns 儲存並還原 secret 套件的可注入函式。
func restoreSecretFns(t *testing.T) {
	t.Helper()
	origStore, origLoad, origDelete := secret.StoreFnForTest()
	t.Cleanup(func() {
		secret.RestoreFnsForTest(origStore, origLoad, origDelete)
	})
}

// setupMockKeychain 初始化 mock keychain 並還原 secret 函式。
func setupMockKeychain(t *testing.T) {
	t.Helper()
	keyring.MockInit()
	restoreSecretFns(t)
}

const testYAMLTwoProfiles = "profiles:\n" +
	"  - name: openai-official\n" +
	"    provider: openai\n" +
	"    api_base: https://api.openai.com/v1\n" +
	"    api_key: sk-official-key\n" +
	"    default_model: gpt-4o\n" +
	"  - name: local-ollama\n" +
	"    provider: openai\n" +
	"    api_base: http://localhost:11434\n" +
	"    api_key: sk-ollama-key\n" +
	"    default_model: llama3.2\n" +
	"default_profile: openai-official\n"

// --- set-key tests ---

func TestSetKey_ExistingProfile(t *testing.T) {
	setupMockKeychain(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, testYAMLTwoProfiles)

	var out bytes.Buffer
	if err := runConfigSetKey(path, "openai-official", "sk-new", &out); err != nil {
		t.Fatalf("runConfigSetKey: %v", err)
	}
	if !strings.Contains(out.String(), "已將金鑰存入 keychain") {
		t.Errorf("output missing confirmation, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "已清除設定檔中的明碼 api_key") {
		t.Errorf("output missing plaintext clear notice, got: %s", out.String())
	}

	// Verify keychain has the new key
	key, err := secret.Load("openai-official")
	if err != nil {
		t.Fatalf("secret.Load: %v", err)
	}
	if key != "sk-new" {
		t.Errorf("keychain key = %q, want %q", key, "sk-new")
	}

	// Verify config no longer has plaintext api_key for this profile
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "sk-official-key") {
		t.Errorf("config still contains old plaintext key:\n%s", data)
	}
}

func TestSetKey_EmptyKey(t *testing.T) {
	setupMockKeychain(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, testYAMLTwoProfiles)

	var out bytes.Buffer
	err := runConfigSetKey(path, "openai-official", "", &out)
	if err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
	if !strings.Contains(err.Error(), "金鑰不可為空") {
		t.Errorf("error = %v, want message containing 金鑰不可為空", err)
	}
}

func TestSetKey_NonExistentProfile(t *testing.T) {
	setupMockKeychain(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, testYAMLTwoProfiles)

	var out bytes.Buffer
	err := runConfigSetKey(path, "nonexistent", "sk-key", &out)
	if err == nil {
		t.Fatal("expected error for non-existent profile, got nil")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("error = %v, want message containing 不存在", err)
	}
}

// --- del-key tests ---

func TestDelKey_Existing(t *testing.T) {
	setupMockKeychain(t)
	if err := secret.Store("openai-official", "sk-test"); err != nil {
		t.Fatalf("secret.Store: %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, testYAMLTwoProfiles)

	var out bytes.Buffer
	if err := runConfigDelKey(path, "openai-official", &out); err != nil {
		t.Fatalf("runConfigDelKey: %v", err)
	}
	if !strings.Contains(out.String(), "已自 keychain 刪除金鑰") {
		t.Errorf("output missing confirmation, got: %s", out.String())
	}
	exists, _ := secret.Exists("openai-official")
	if exists {
		t.Errorf("key still exists in keychain after delete")
	}
}

func TestDelKey_NotInKeychain(t *testing.T) {
	setupMockKeychain(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, testYAMLTwoProfiles)

	var out bytes.Buffer
	err := runConfigDelKey(path, "openai-official", &out)
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
	if !strings.Contains(err.Error(), "未在 keychain 中") {
		t.Errorf("error = %v, want message containing 未在 keychain 中", err)
	}
}

func TestDelKey_NonExistentProfile(t *testing.T) {
	setupMockKeychain(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, testYAMLTwoProfiles)

	var out bytes.Buffer
	err := runConfigDelKey(path, "nonexistent", &out)
	if err == nil {
		t.Fatal("expected error for non-existent profile, got nil")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("error = %v, want message containing 不存在", err)
	}
}

// --- import-keys tests ---

func TestImportKeys_AllSuccess(t *testing.T) {
	setupMockKeychain(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, testYAMLTwoProfiles)

	var out bytes.Buffer
	if err := runConfigImportKeys(path, &out); err != nil {
		t.Fatalf("runConfigImportKeys: %v", err)
	}
	if !strings.Contains(out.String(), "匯入 2 個金鑰至 keychain") {
		t.Errorf("output missing import count, got: %s", out.String())
	}

	// Verify both keys in keychain
	key1, _ := secret.Load("openai-official")
	if key1 != "sk-official-key" {
		t.Errorf("keychain key for openai-official = %q, want %q", key1, "sk-official-key")
	}
	key2, _ := secret.Load("local-ollama")
	if key2 != "sk-ollama-key" {
		t.Errorf("keychain key for local-ollama = %q, want %q", key2, "sk-ollama-key")
	}

	// Verify config has no plaintext keys
	data, _ := os.ReadFile(path)
	s := string(data)
	if strings.Contains(s, "sk-official-key") || strings.Contains(s, "sk-ollama-key") {
		t.Errorf("config still contains plaintext keys:\n%s", s)
	}
}

func TestImportKeys_PartialFailure(t *testing.T) {
	keyring.MockInit()
	origStore, origLoad, origDelete := secret.StoreFnForTest()
	t.Cleanup(func() {
		secret.RestoreFnsForTest(origStore, origLoad, origDelete)
	})

	// Make storeFn fail for "local-ollama"
	secret.SetStoreFnForTest(func(service, user, password string) error {
		if strings.Contains(user, "local-ollama") {
			return errors.New("mock: keychain unavailable")
		}
		return keyring.Set(service, user, password)
	})

	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, testYAMLTwoProfiles)

	var out bytes.Buffer
	err := runConfigImportKeys(path, &out)
	if err == nil {
		t.Fatal("expected error for partial failure, got nil")
	}
	if !strings.Contains(out.String(), "匯入失敗") {
		t.Errorf("output missing failure notice, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "local-ollama") {
		t.Errorf("output missing failed profile name, got: %s", out.String())
	}

	// openai-official should be cleared, local-ollama should still have plaintext
	data, _ := os.ReadFile(path)
	s := string(data)
	if strings.Contains(s, "sk-official-key") {
		t.Errorf("successful profile still has plaintext key:\n%s", s)
	}
	if !strings.Contains(s, "sk-ollama-key") {
		t.Errorf("failed profile should still have plaintext key:\n%s", s)
	}
}

func TestImportKeys_NoPlaintextKeys(t *testing.T) {
	setupMockKeychain(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n"+
		"  - name: empty-profile\n"+
		"    provider: openai\n"+
		"    api_base: https://api.openai.com/v1\n"+
		"    default_model: gpt-4o\n"+
		"default_profile: empty-profile\n")

	before, _ := os.ReadFile(path)
	var out bytes.Buffer
	if err := runConfigImportKeys(path, &out); err != nil {
		t.Fatalf("runConfigImportKeys: %v", err)
	}
	if !strings.Contains(out.String(), "設定檔中無明碼金鑰可匯入") {
		t.Errorf("output missing no-keys notice, got: %s", out.String())
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("config file was modified when no keys to import")
	}
}

// --- list with key source ---

func TestConfigList_MixedKeySources(t *testing.T) {
	setupMockKeychain(t)

	// Store a key in keychain for openai-official
	if err := secret.Store("openai-official", "sk-chain-key"); err != nil {
		t.Fatalf("secret.Store: %v", err)
	}

	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n"+
		"  - name: openai-official\n"+
		"    provider: openai\n"+
		"    api_base: https://api.openai.com/v1\n"+
		"    api_key: \"\"\n"+
		"    default_model: gpt-4o\n"+
		"  - name: local-ollama\n"+
		"    provider: openai\n"+
		"    api_base: http://localhost:11434\n"+
		"    api_key: sk-plain-key\n"+
		"    default_model: llama3.2\n"+
		"  - name: empty-profile\n"+
		"    provider: openai\n"+
		"    api_base: https://api.openai.com/v1\n"+
		"    default_model: gpt-4o\n"+
		"default_profile: openai-official\n")

	var out bytes.Buffer
	if err := runConfigList(path, &out); err != nil {
		t.Fatalf("runConfigList: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "keychain") {
		t.Errorf("output missing 'keychain' source indicator, got: %s", s)
	}
	if !strings.Contains(s, "plaintext") {
		t.Errorf("output missing 'plaintext' source indicator, got: %s", s)
	}
	if !strings.Contains(s, "missing") {
		t.Errorf("output missing 'missing' source indicator, got: %s", s)
	}
}

// --- config add without api-key ---

func TestConfigAdd_WithoutApiKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".byok", "config.yaml")
	var out bytes.Buffer
	if err := runConfigAdd(path, "no-key-profile", "openai",
		"https://api.openai.com/v1", "", "gpt-4o", &out); err != nil {
		t.Fatalf("runConfigAdd without api-key: %v", err)
	}
	if !strings.Contains(out.String(), "已新增 profile") {
		t.Errorf("output missing confirmation, got: %s", out.String())
	}
	data, _ := os.ReadFile(path)
	s := string(data)
	if !strings.Contains(s, "no-key-profile") {
		t.Errorf("config missing profile name, got: %s", s)
	}
	// With omitempty, empty api_key should not appear in YAML
	if strings.Contains(s, "api_key") {
		t.Errorf("config should not contain api_key field for empty key, got: %s", s)
	}
}

// Ensure config.Resolver is restorable for tests that swap it.
func restoreResolver(t *testing.T) {
	t.Helper()
	orig := config.Resolver
	t.Cleanup(func() { config.Resolver = orig })
}