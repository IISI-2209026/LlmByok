package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// TestLaunchCodex_ByokApiKeyAndConfigInjected 驗證 LaunchCodex 以真實
// profile 啟動 stub，stub 接收到 BYOK_CODEX_API_KEY 環境變數與一組
// --config 旗標，且命令列順序為 --config 在前、extraArgs 在後。
// 同時驗證父程序環境保持不變。
func TestLaunchCodex_ByokApiKeyAndConfigInjected(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")

	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)
	t.Setenv("BYOK_PARENT_MARKER", "before")

	// 將 HOME/USERPROFILE 指向暫存目錄，並斷言 byok 不會建立或修改
	// ~/.codex/config.toml（LaunchCodex 僅以子程序環境覆寫，不寫檔）。
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome)
	codexConfigBefore := readIfExists(filepath.Join(fakeHome, ".codex", "config.toml"))

	parentBefore := snapshotEnv()

	profile := &config.Profile{
		Name:         "openai-official",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey:       "sk-codex-integration",
		DefaultModel: "gpt-4o",
	}

	var stdout, stderr strings.Builder
	if err := LaunchCodex(profile, "gemma4", stub, []string{"--yolo", "exec"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("LaunchCodex failed: %v (stderr=%s)", err, stderr.String())
	}

	// 父程序環境必須保持不變。
	parentAfter := snapshotEnv()
	if !envEqual(parentBefore, parentAfter) {
		t.Fatalf("parent environment changed after launch\nbefore:\n%s\nafter:\n%s",
			strings.Join(parentBefore, "\n"), strings.Join(parentAfter, "\n"))
	}
	if got := os.Getenv("BYOK_CODEX_API_KEY"); got != "" {
		t.Errorf("parent BYOK_CODEX_API_KEY leaked = %q (must be empty)", got)
	}
	if got := os.Getenv("BYOK_PARENT_MARKER"); got != "before" {
		t.Errorf("parent BYOK_PARENT_MARKER = %q, want %q", got, "before")
	}

	// ~/.codex/config.toml 不應被 byok 建立或修改。
	codexConfigAfter := readIfExists(filepath.Join(fakeHome, ".codex", "config.toml"))
	if codexConfigBefore != codexConfigAfter {
		t.Errorf("~/.codex/config.toml changed by LaunchCodex\nbefore: %q\nafter: %q",
			codexConfigBefore, codexConfigAfter)
	}

	// 子程序環境必須包含 BYOK_CODEX_API_KEY。
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read stub env output: %v", err)
	}
	childEnv := strings.Split(string(data), "\n")
	if got := envLookup(childEnv, "BYOK_CODEX_API_KEY"); got != "sk-codex-integration" {
		t.Errorf("child BYOK_CODEX_API_KEY = %q, want %q", got, "sk-codex-integration")
	}

	// 命令列順序：--config 旗標在前、--yolo、exec 在後。
	argsData, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args output: %v", err)
	}
	gotArgs := splitArgs(string(argsData))
	wantArgs := []string{
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

// TestLaunchCodex_NoExtraArgs 驗證不傳 extraArgs 時子程序僅收到 --config
// 旗標，無 --yolo 或透傳參數。
func TestLaunchCodex_NoExtraArgs(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	argsFile := filepath.Join(t.TempDir(), "args.txt")
	outFile := filepath.Join(t.TempDir(), "env.txt")

	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	profile := &config.Profile{
		Name:         "openai-official",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey:       "sk-codex-integration",
		DefaultModel: "gpt-4o",
	}

	var stdout, stderr strings.Builder
	if err := LaunchCodex(profile, "", stub, nil, nil, &stdout, &stderr); err != nil {
		t.Fatalf("LaunchCodex failed: %v (stderr=%s)", err, stderr.String())
	}

	argsData, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args output: %v", err)
	}
	gotArgs := splitArgs(string(argsData))
	// 僅 5 對 --config，共 10 個元素。
	if len(gotArgs) != 10 {
		t.Fatalf("child args len = %d, want 10: %v", len(gotArgs), gotArgs)
	}
	for i := 0; i < 10; i += 2 {
		if gotArgs[i] != "--config" {
			t.Errorf("child arg[%d] = %q, want \"--config\"", i, gotArgs[i])
		}
	}
}

// splitArgs 將 stub 寫入的參數字串（每行一個）拆為切片，並過濾空白行。
func splitArgs(data string) []string {
	var out []string
	for _, a := range strings.Split(data, "\n") {
		if a != "" {
			out = append(out, a)
		}
	}
	return out
}

// readIfExists 回傳 path 的內容；檔案不存在時回傳空字串。
func readIfExists(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}