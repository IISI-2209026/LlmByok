package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// TestLaunchCodexApp_MissingConfigFile 驗證設定檔不存在時印出提示並 exit 1。
func TestLaunchCodexApp_MissingConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	var stdout, stderr bytes.Buffer
	err := runLaunchCodexApp(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "找不到設定檔") {
		t.Errorf("stderr missing '找不到設定檔', got: %s", stderr.String())
	}
}

// TestLaunchCodexApp_MissingProfile 驗證具名 profile 不存在時列出可用 profile 並 exit 1。
func TestLaunchCodexApp_MissingProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchCodexApp(path, "nonexistent", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), `找不到 profile "nonexistent"`) {
		t.Errorf("stderr missing not-found message, got: %s", stderr.String())
	}
}

// TestLaunchCodexApp_NonOpenaiProviderRejected 驗證非 openai provider 被拒。
func TestLaunchCodexApp_NonOpenaiProviderRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: azure-prod\n    provider: azure\n    api_base: https://example.openai.azure.com\n    api_key: az-key\n    default_model: gpt-4o\ndefault_profile: azure-prod\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchCodexApp(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "僅支援 openai provider") {
		t.Errorf("stderr missing provider rejection, got: %s", stderr.String())
	}
}

// TestLaunchCodexApp_NotInstalled 驗證 codex 不在 PATH 時印出安裝提示並 exit 1。
func TestLaunchCodexApp_NotInstalled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	t.Setenv("PATH", "")
	var stdout, stderr bytes.Buffer
	err := runLaunchCodexApp(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), `找不到 "codex" 可執行檔`) {
		t.Errorf("stderr missing codex not-found message, got: %s", stderr.String())
	}
}

// TestLaunchCodexApp_AppSubcommandFirst 驗證 app 子命令為命令列第一個參數，
// 接著是 --config 旗標，最後是透傳 extraArgs。
func TestLaunchCodexApp_AppSubcommandFirst(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")

	stubExe := buildCopilotStubForCodex(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	extraArgs := []string{"--yolo", "exec"}
	if err := runLaunchCodexApp(path, "", "gemma4", extraArgs, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchCodexApp returned unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	argsData, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args: %v", err)
	}
	got := splitNonEmpty(string(argsData))
	// 預期：app + 5 對 --config（共 10 元素）+ --yolo + exec = 13
	if len(got) != 13 {
		t.Fatalf("child args len = %d, want 13: %v", len(got), got)
	}
	if got[0] != "app" {
		t.Errorf("first arg = %q, want %q", got[0], "app")
	}
	// --config 旗標從索引 1 開始，共 5 對。
	for i := 1; i <= 9; i += 2 {
		if got[i] != "--config" {
			t.Errorf("child arg[%d] = %q, want \"--config\"", i, got[i])
		}
	}
	// --yolo 在 --config 之後。
	yoloIdx := indexOf(got, "--yolo")
	if yoloIdx != 11 {
		t.Errorf("--yolo index = %d, want 11 (after app + 5 --config pairs): %v", yoloIdx, got)
	}
	if got[12] != "exec" {
		t.Errorf("passthrough arg = %q, want \"exec\"", got[12])
	}
}

// TestLaunchCodexApp_ConfigArgsContent 驗證 --config 覆寫內容含正確 model 與 base_url，
// 且 app 子命令位於第一個參數。
func TestLaunchCodexApp_ConfigArgsContent(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")

	stubExe := buildCopilotStubForCodex(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	if err := runLaunchCodexApp(path, "", "", nil, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchCodexApp returned unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	envData, _ := os.ReadFile(outFile)
	childEnv := strings.Split(string(envData), "\n")
	if got := envLookupOSIn(childEnv, "BYOK_CODEX_API_KEY"); got != "sk-xxxx" {
		t.Errorf("child BYOK_CODEX_API_KEY = %q, want sk-xxxx", got)
	}

	argsData, _ := os.ReadFile(argsFile)
	got := splitNonEmpty(string(argsData))
	if len(got) == 0 || got[0] != "app" {
		t.Fatalf("first arg = %q, want %q", got, "app")
	}
	wantFragments := []string{
		`model="gpt-4o"`,
		`model_provider="byok"`,
		`model_providers.byok.name="BYOK"`,
		`model_providers.byok.base_url="https://api.openai.com/v1"`,
		`model_providers.byok.env_key="BYOK_CODEX_API_KEY"`,
	}
	for _, w := range wantFragments {
		if !containsString(got, w) {
			t.Errorf("child args missing %q, got %v", w, got)
		}
	}
}