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

// TestLaunchCodex_ContextWindowFlagMapsToChildConfigArgs 驗證 cmd dispatch 將
// CLI 旗標解析的有效 context 傳入 runner：子程序參數含 --config
// model_context_window=272000，位置緊接 env_key pair 之後、恰一次。
func TestLaunchCodex_ContextWindowFlagMapsToChildConfigArgs(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    models: [gpt-4o]\ndefault_profile: openai-official\n")

	stubExe := buildCopilotStubForCodex(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	opt := launchOptions{cliContextTokens: ptrToken(272000)}
	if err := runLaunchCodex(path, "", "", nil, &stdout, &stderr, opt); err != nil {
		t.Fatalf("runLaunchCodex returned unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	argsData, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args: %v", err)
	}
	got := splitNonEmpty(string(argsData))
	ctxIdx := indexOf(got, "model_context_window=272000")
	if ctxIdx < 0 {
		t.Fatalf("child args missing model_context_window=272000, got %v", got)
	}
	if got[ctxIdx-1] != "--config" {
		t.Errorf("context value at %d must be paired with --config, got %q", ctxIdx, got[ctxIdx-1])
	}
	envKeyIdx := indexOf(got, `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`)
	if envKeyIdx < 0 {
		t.Fatalf("child args missing env_key pair, got %v", got)
	}
	if !(envKeyIdx < ctxIdx) {
		t.Errorf("context pair must follow env_key pair (env_key %d, ctx %d): %v", envKeyIdx, ctxIdx, got)
	}
	if count := strings.Count(strings.Join(got, "\n"), "model_context_window"); count != 1 {
		t.Errorf("expected exactly one model_context_window arg, got %d: %v", count, got)
	}
}

// TestLaunchCodex_ProfileDefaultContextWindowMapped 驗證僅設定 profile
// default_model_limits.context_window_tokens 時（無 CLI 旗標）同樣映射為
// 子程序 --config model_context_window=<value>。
func TestLaunchCodex_ProfileDefaultContextWindowMapped(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, `profiles:
  - name: openai-official
    provider: openai
    api_base: https://api.openai.com/v1
    api_key: sk-xxxx
    default_model_limits:
      context_window_tokens: 272000
    models: [gpt-4o]
default_profile: openai-official
`)
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

	argsData, _ := os.ReadFile(argsFile)
	got := splitNonEmpty(string(argsData))
	if !containsString(got, "model_context_window=272000") {
		t.Errorf("child args missing model_context_window=272000 (profile default), got %v", got)
	}
	envKeyIdx := indexOf(got, `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`)
	ctxIdx := indexOf(got, "model_context_window=272000")
	if envKeyIdx >= 0 && ctxIdx >= 0 && !(envKeyIdx < ctxIdx) {
		t.Errorf("context pair must follow env_key pair (env_key=%d ctx=%d): %v", envKeyIdx, ctxIdx, got)
	}
}

// TestLaunchCodex_OutputOnlyLimitsLeaveArgsUnchanged 驗證僅有效
// max_output_tokens 時：config 參數不變（--yolo 仍在索引 10）、不注入
// model_context_window；warning 仍寫入 stderr（共用 warning contract）。
func TestLaunchCodex_OutputOnlyLimitsLeaveArgsUnchanged(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, `profiles:
  - name: openai-official
    provider: openai
    api_base: https://api.openai.com/v1
    api_key: sk-xxxx
    default_model_limits:
      max_output_tokens: 32768
    models: [gpt-4o]
default_profile: openai-official
`)

	stubExe := buildCopilotStubForCodex(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	extraArgs := []string{"--yolo", "exec", "review this"}
	if err := runLaunchCodex(path, "", "", extraArgs, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchCodex returned unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args: %v", err)
	}
	got := splitNonEmpty(string(data))
	// output-only：參數順序與既有一致（5 對 --config + extraArgs）。
	if yoloIdx := indexOf(got, "--yolo"); yoloIdx != 10 {
		t.Errorf("--yolo index = %d, want 10 (5 --config pairs, context unset): %v", yoloIdx, got)
	}
	for _, a := range got {
		if strings.Contains(a, "model_context_window") {
			t.Errorf("output-only limits must not produce model_context_window, got %v", got)
		}
		if strings.Contains(a, "max_output") {
			t.Errorf("no output override may reach child args, got %q in %v", a, got)
		}
	}
	if !strings.Contains(stderr.String(), "max_output_tokens") {
		t.Errorf("stderr should contain unsupported output warning, got: %s", stderr.String())
	}
}

// TestLaunchCodex_ContextWithOutputIgnored 驗證 context 與 output 同時有效：
// 子程序僅收到 model_context_window=1000000、無 output override，stderr 恰
// 一行 unsupported warning。
func TestLaunchCodex_ContextWithOutputIgnored(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    models: [gpt-4o]\ndefault_profile: openai-official\n")

	stubExe := buildCopilotStubForCodex(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	opt := launchOptions{cliContextTokens: ptrToken(1000000), cliMaxOutputTokens: ptrToken(128000)}
	if err := runLaunchCodex(path, "", "gpt-5.4", nil, &stdout, &stderr, opt); err != nil {
		t.Fatalf("runLaunchCodex returned unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args: %v", err)
	}
	got := splitNonEmpty(string(data))
	ctxCount := 0
	for _, a := range got {
		if strings.Contains(a, "model_context_window") {
			ctxCount++
		}
		if strings.Contains(a, "max_output") {
			t.Errorf("no output override may reach the child, got %q in %v", a, got)
		}
	}
	if ctxCount != 1 {
		t.Fatalf("expected exactly one model_context_window arg, got %d: %v", ctxCount, got)
	}
	if !containsString(got, "model_context_window=1000000") {
		t.Errorf("child args missing model_context_window=1000000, got %v", got)
	}

	warnCount := 0
	for _, line := range strings.Split(stderr.String(), "\n") {
		if strings.Contains(line, "max_output_tokens") {
			warnCount++
		}
	}
	if warnCount != 1 {
		t.Errorf("expected exactly one unsupported output warning line, got %d: %q", warnCount, stderr.String())
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
