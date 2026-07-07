package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// TestLaunchCodex_MissingConfigFile 驗證設定檔不存在時印出提示並 exit 1。
func TestLaunchCodex_MissingConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	var stdout, stderr bytes.Buffer
	err := runLaunchCodex(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "找不到設定檔") {
		t.Errorf("stderr missing '找不到設定檔', got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "byok config add") {
		t.Errorf("stderr missing hint to run `byok config add`, got: %s", stderr.String())
	}
}

// TestLaunchCodex_MissingProfile 驗證具名 profile 不存在時列出可用 profile 並 exit 1。
func TestLaunchCodex_MissingProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchCodex(path, "nonexistent", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), `找不到 profile "nonexistent"`) {
		t.Errorf("stderr missing not-found message, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "openai-official") {
		t.Errorf("stderr should list available profiles, got: %s", stderr.String())
	}
}

// TestLaunchCodex_NonOpenaiProviderRejected 驗證非 openai provider 被拒。
func TestLaunchCodex_NonOpenaiProviderRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: azure-prod\n    provider: azure\n    api_base: https://example.openai.azure.com\n    api_key: az-key\n    default_model: gpt-4o\ndefault_profile: azure-prod\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchCodex(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "僅支援 openai provider") {
		t.Errorf("stderr missing provider rejection, got: %s", stderr.String())
	}
}

// TestLaunchCodex_NoDefaultProfile 驗證未指定 profile 且未設定 default_profile 時 exit 1。
func TestLaunchCodex_NoDefaultProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchCodex(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "default_profile") {
		t.Errorf("stderr missing default_profile hint, got: %s", stderr.String())
	}
}

// TestLaunchCodex_NotInstalled 驗證 codex 不在 PATH 時印出安裝提示並 exit 1。
func TestLaunchCodex_NotInstalled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	t.Setenv("PATH", "")
	var stdout, stderr bytes.Buffer
	err := runLaunchCodex(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), `找不到 "codex" 可執行檔`) {
		t.Errorf("stderr missing codex not-found message, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "安裝 Codex CLI") {
		t.Errorf("stderr missing install hint, got: %s", stderr.String())
	}
}

// TestLaunchCodex_ParentEnvUnchanged 驗證錯誤路徑（codex 未安裝）後
// 父程序環境保持不變。
func TestLaunchCodex_ParentEnvUnchanged(t *testing.T) {
	t.Setenv("BYOK_PARENT_CHECK", "intact")
	t.Setenv("BYOK_CODEX_API_KEY", "")
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	t.Setenv("PATH", "")
	var stdout, stderr bytes.Buffer
	_ = runLaunchCodex(path, "", "", nil, &stdout, &stderr)
	if got := envLookupOS("BYOK_PARENT_CHECK"); got != "intact" {
		t.Errorf("parent BYOK_PARENT_CHECK = %q, want %q", got, "intact")
	}
	if got := envLookupOS("BYOK_CODEX_API_KEY"); got != "" {
		t.Errorf("parent BYOK_CODEX_API_KEY leaked = %q (must be empty)", got)
	}
}

// TestLaunchCodex_ExtraArgsOrder 驗證 extraArgs（--yolo 與透傳）原樣轉發
// 給 runner.LaunchCodex。以可注入的 fake launcher 攔截參數。
func TestLaunchCodex_ExtraArgsOrder(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")

	// 提供一個可在空 PATH 下解析的 codex 執行檔：將 stub 放進暫存目錄並
	// 將該目錄設為 PATH。使用真實 stub 讓 runLaunchCodex 通過 LookPath。
	stubExe := buildCopilotStubForCodex(t)
	stubDir := filepath.Dir(stubExe)
	t.Setenv("PATH", stubDir)

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	extraArgs := []string{"--yolo", "exec", "review this"}
	err := runLaunchCodex(path, "", "gemma4", extraArgs, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runLaunchCodex returned unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args: %v", err)
	}
	got := splitNonEmpty(string(data))
	// 預期：5 對 --config（共 10 元素）+ --yolo + exec + review this
	if len(got) < 12 {
		t.Fatalf("child args len = %d, want >= 12: %v", len(got), got)
	}
	// --yolo 緊接在 --config 之後，透傳在最後。
	yoloIdx := indexOf(got, "--yolo")
	if yoloIdx != 10 {
		t.Errorf("--yolo index = %d, want 10 (after 5 --config pairs): %v", yoloIdx, got)
	}
	if got[11] != "exec" || got[12] != "review this" {
		t.Errorf("passthrough args = %v after --yolo, want [exec review this]", got[11:])
	}
}

// TestLaunchCodex_ConfigArgsContent 驗證 --config 覆寫內容含正確 model 與 base_url。
func TestLaunchCodex_ConfigArgsContent(t *testing.T) {
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
	if err := runLaunchCodex(path, "", "", nil, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchCodex returned unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	envData, _ := os.ReadFile(outFile)
	childEnv := strings.Split(string(envData), "\n")
	if got := envLookupOSIn(childEnv, "BYOK_CODEX_API_KEY"); got != "sk-xxxx" {
		t.Errorf("child BYOK_CODEX_API_KEY = %q, want sk-xxxx", got)
	}

	argsData, _ := os.ReadFile(argsFile)
	got := splitNonEmpty(string(argsData))
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

// buildCopilotStubForCodex 編譯 testdata stub 至暫存目錄，命名為 codex
// （Windows 加 .exe），供 runLaunchCodex 的 LookPath("codex") 解析。
func buildCopilotStubForCodex(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "codex")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", exe, ".")
	cmd.Dir = filepath.Join("..", "internal", "runner", "testdata", "stub")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build stub: %v\n%s", err, out)
	}
	return exe
}

// splitNonEmpty 將換行分隔的字串拆為切片，過濾空白行。
func splitNonEmpty(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// indexOf 回傳 target 在 slice 中的索引，找不到回傳 -1。
func indexOf(slice []string, target string) int {
	for i, s := range slice {
		if s == target {
			return i
		}
	}
	return -1
}

// containsString 判斷 slice 是否包含 target。
func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// envLookupOSIn 在 "KEY=VALUE" 切片中尋找 KEY 並回傳 VALUE。
func envLookupOSIn(env []string, key string) string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return strings.TrimPrefix(e, prefix)
		}
	}
	return ""
}