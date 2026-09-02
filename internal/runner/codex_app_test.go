package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// TestLaunchCodexApp_AppSubcommandPrecedesConfig 驗證 LaunchCodexApp 將
// app 子命令插入為命令列第一個參數，接著是 --config 旗標，最後是
// extraArgs。子程序環境必須包含 BYOK_CODEX_API_KEY。
func TestLaunchCodexApp_AppSubcommandPrecedesConfig(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")

	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	profile := &config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-codex-app-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}

	var stdout, stderr strings.Builder
	if err := LaunchCodexApp(profile, "gemma4", nil, stub, []string{"--yolo", "exec"}, nil, &stdout, &stderr, nil); err != nil {
		t.Fatalf("LaunchCodexApp failed: %v (stderr=%s)", err, stderr.String())
	}

	// 子程序環境必須包含 BYOK_CODEX_API_KEY。
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read stub env output: %v", err)
	}
	childEnv := strings.Split(string(data), "\n")
	if got := envLookup(childEnv, "BYOK_CODEX_API_KEY"); got != "sk-codex-app-test" {
		t.Errorf("child BYOK_CODEX_API_KEY = %q, want %q", got, "sk-codex-app-test")
	}

	// 命令列順序：app → --config 旗標 → --yolo → exec。
	argsData, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args output: %v", err)
	}
	gotArgs := splitArgs(string(argsData))
	wantArgs := []string{
		"app",
		"--config", `model="gemma4"`,
		"--config", `model_provider="byok"`,
		"--config", `model_providers.byok.name="BYOK"`,
		"--config", `model_providers.byok.base_url="https://api.openai.com/v1"`,
		"--config", `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`,
		"--yolo", "exec",
	}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("child args len = %d, want %d: %v", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i, w := range wantArgs {
		if gotArgs[i] != w {
			t.Errorf("child arg[%d] = %q, want %q", i, gotArgs[i], w)
		}
	}
}

// TestLaunchCodexApp_NoExtraArgs 驗證不傳 extraArgs 時子程序僅收到
// app 子命令與 --config 旗標。
func TestLaunchCodexApp_NoExtraArgs(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	argsFile := filepath.Join(t.TempDir(), "args.txt")
	outFile := filepath.Join(t.TempDir(), "env.txt")

	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	profile := &config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-codex-app-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}

	var stdout, stderr strings.Builder
	if err := LaunchCodexApp(profile, "", nil, stub, nil, nil, &stdout, &stderr, nil); err != nil {
		t.Fatalf("LaunchCodexApp failed: %v (stderr=%s)", err, stderr.String())
	}

	argsData, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args output: %v", err)
	}
	gotArgs := splitArgs(string(argsData))
	// app + 5 對 --config（共 10 元素）= 11。
	if len(gotArgs) != 11 {
		t.Fatalf("child args len = %d, want 11: %v", len(gotArgs), gotArgs)
	}
	if gotArgs[0] != "app" {
		t.Errorf("first arg = %q, want %q", gotArgs[0], "app")
	}
	for i := 1; i < 11; i += 2 {
		if gotArgs[i] != "--config" {
			t.Errorf("child arg[%d] = %q, want \"--config\"", i, gotArgs[i])
		}
	}
}

// TestLaunchCodexApp_ParentEnvUnchanged 驗證 LaunchCodexApp 啟動子程序後
// 父程序環境保持不變。
func TestLaunchCodexApp_ParentEnvUnchanged(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")

	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)
	t.Setenv("BYOK_PARENT_MARKER", "before")

	parentBefore := snapshotEnv()

	profile := &config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-codex-app-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}

	var stdout, stderr strings.Builder
	if err := LaunchCodexApp(profile, "", nil, stub, nil, nil, &stdout, &stderr, nil); err != nil {
		t.Fatalf("LaunchCodexApp failed: %v (stderr=%s)", err, stderr.String())
	}

	parentAfter := snapshotEnv()
	if !envEqual(parentBefore, parentAfter) {
		t.Fatalf("parent environment changed after launch\nbefore:\n%s\nafter:\n%s",
			strings.Join(parentBefore, "\n"), strings.Join(parentAfter, "\n"))
	}
	if got := os.Getenv("BYOK_CODEX_API_KEY"); got != "" {
		t.Errorf("parent BYOK_CODEX_API_KEY leaked = %q (must be empty)", got)
	}
}

// codexAppTokenProfile 供 token limit 測試建構共用 profile。
func codexAppTokenProfile() *config.Profile {
	return &config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-codex-app-token-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
}

// TestLaunchCodexApp_ContextWindowAfterAppSubcommand 驗證有效 context 映射時
// 命令列以 app 開頭，接著為既有五對 provider --config、model_context_window、
// 最後 extraArgs；既有參數順序不變。
func TestLaunchCodexApp_ContextWindowAfterAppSubcommand(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	limits := &TokenLimits{ContextWindowTokens: ptrLmt64(272000)}

	var stdout, stderr strings.Builder
	if err := LaunchCodexApp(codexAppTokenProfile(), "gemma4", limits, stub, []string{"--yolo", "exec"}, nil, &stdout, &stderr, nil); err != nil {
		t.Fatalf("LaunchCodexApp failed: %v (stderr=%s)", err, stderr.String())
	}

	argsData, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args: %v", err)
	}
	got := splitArgs(string(argsData))
	want := []string{
		"app",
		"--config", `model="gemma4"`,
		"--config", `model_provider="byok"`,
		"--config", `model_providers.byok.name="BYOK"`,
		"--config", `model_providers.byok.base_url="https://api.openai.com/v1"`,
		"--config", `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`,
		"--config", `model_context_window=272000`,
		"--yolo", "exec",
	}
	if len(got) != len(want) {
		t.Fatalf("child args len = %d, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("child arg[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// TestLaunchCodexApp_UnsetContextOmittedOutputIgnored 驗證 context 未設定時
// 不注入 model_context_window；output 欄位即使有效也被忽略，且 context 有效
// 時僅注入整數 context override、無任何 output override。
func TestLaunchCodexApp_UnsetContextOmittedOutputIgnored(t *testing.T) {
	// baseArgs 為 app 首參數 + 既有五對 provider --config。
	baseArgs := []string{
		"app",
		"--config", `model="gemma4"`,
		"--config", `model_provider="byok"`,
		"--config", `model_providers.byok.name="BYOK"`,
		"--config", `model_providers.byok.base_url="https://api.openai.com/v1"`,
		"--config", `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`,
	}
	// wantArgs 組合 expected config args 與結尾透傳參數。
	wantArgs := func(ctxValue string) []string {
		out := append([]string(nil), baseArgs...)
		if ctxValue != "" {
			out = append(out, "--config", "model_context_window="+ctxValue)
		}
		return append(out, "--yolo", "exec")
	}
	cases := []struct {
		name     string
		limits   *TokenLimits
		ctxValue string
	}{
		{"nil limits", nil, ""},
		{"empty limits", &TokenLimits{}, ""},
		{"output-only limits", &TokenLimits{MaxOutputTokens: ptrLmt64(128000)}, ""},
		{"context with output ignored", &TokenLimits{ContextWindowTokens: ptrLmt64(1000000), MaxOutputTokens: ptrLmt64(128000)}, "1000000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := buildStub(t, filepath.Join("testdata", "stub"))
			outFile := filepath.Join(t.TempDir(), "env.txt")
			argsFile := filepath.Join(t.TempDir(), "args.txt")
			t.Setenv("BYOK_STUB_OUT", outFile)
			t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

			var stdout, stderr strings.Builder
			if err := LaunchCodexApp(codexAppTokenProfile(), "gemma4", tc.limits, stub, []string{"--yolo", "exec"}, nil, &stdout, &stderr, nil); err != nil {
				t.Fatalf("LaunchCodexApp failed: %v (stderr=%s)", err, stderr.String())
			}

			argsData, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatalf("read stub args: %v", err)
			}
			got := splitArgs(string(argsData))
			want := wantArgs(tc.ctxValue)
			if len(got) != len(want) {
				t.Fatalf("child args len = %d, want %d: %v", len(got), len(want), got)
			}
			for _, a := range got {
				if strings.Contains(a, "max_output") {
					t.Errorf("output override must be ignored entirely, got %q in %v", a, got)
				}
			}
			for i, w := range want {
				if got[i] != w {
					t.Errorf("child arg[%d] = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}
