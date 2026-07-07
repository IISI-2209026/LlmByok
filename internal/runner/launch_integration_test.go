package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// TestLaunchIntegration_ByokVarsInjected 編譯 stub "copilot" 輔助程式，
// 以 profile + model 覆寫呼叫 Launch，並斷言子程序接收到四個覆寫後的
// BYOK 環境變數，同時父程序環境保持不變。
func TestLaunchIntegration_ByokVarsInjected(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	// stub 將其環境寫入的檔案。
	outFile := filepath.Join(t.TempDir(), "env.txt")

	// 設定一個標記變數以確認非 BYOK 變數被保留，且 Launch 不會
	// 修改父程序環境。
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_PARENT_MARKER", "before")

	// 啟動前快照父程序環境。
	parentBefore := snapshotEnv()

	profile := &config.Profile{
		Name:         "openai-official",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey: "sk-test-integration",
		Models: []string{"gpt-4o"},
	}

	var stdout, stderr strings.Builder
	if err := Launch(profile, "gemma4", stub, nil, nil, &stdout, &stderr); err != nil {
		t.Fatalf("Launch failed: %v (stderr=%s)", err, stderr.String())
	}

	// 父程序環境必須保持不變。
	parentAfter := snapshotEnv()
	if !envEqual(parentBefore, parentAfter) {
		t.Fatalf("parent environment changed after launch\nbefore:\n%s\nafter:\n%s",
			strings.Join(parentBefore, "\n"), strings.Join(parentAfter, "\n"))
	}
	if got := os.Getenv("BYOK_PARENT_MARKER"); got != "before" {
		t.Fatalf("BYOK_PARENT_MARKER = %q, want %q (parent env must be untouched)", got, "before")
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read stub env output: %v", err)
	}
	childEnv := strings.Split(string(data), "\n")

	want := map[string]string{
		"COPILOT_PROVIDER_BASE_URL": "https://api.openai.com/v1",
		"COPILOT_PROVIDER_TYPE":      "openai",
		"COPILOT_PROVIDER_API_KEY":   "sk-test-integration",
		"COPILOT_MODEL":              "gemma4",
	}
	for key, expected := range want {
		got := envLookup(childEnv, key)
		if got != expected {
			t.Errorf("child %s = %q, want %q", key, got, expected)
		}
	}

	// 非 BYOK 標記也必須傳遞給子程序。
	if got := envLookup(childEnv, "BYOK_PARENT_MARKER"); got != "before" {
		t.Errorf("child BYOK_PARENT_MARKER = %q, want %q (preserved vars should reach child)", got, "before")
	}
}

// TestLaunchIntegration_ExtraArgsForwarded 驗證 Launch 將 extraArgs
// 原樣轉發給子程序作為命令列參數，同時父程序環境保持不變。
func TestLaunchIntegration_ExtraArgsForwarded(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	argsFile := filepath.Join(t.TempDir(), "args.txt")
	outFile := filepath.Join(t.TempDir(), "env.txt")

	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)
	t.Setenv("BYOK_PARENT_MARKER", "before")

	parentBefore := snapshotEnv()

	profile := &config.Profile{
		Name:         "openai-official",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey: "sk-test-integration",
		Models: []string{"gpt-4o"},
	}

	extraArgs := []string{"--yolo", "--continue", "--model", "x"}

	var stdout, stderr strings.Builder
	if err := Launch(profile, "gemma4", stub, extraArgs, nil, &stdout, &stderr); err != nil {
		t.Fatalf("Launch failed: %v (stderr=%s)", err, stderr.String())
	}

	// 父程序環境必須保持不變。
	parentAfter := snapshotEnv()
	if !envEqual(parentBefore, parentAfter) {
		t.Fatalf("parent environment changed after launch\nbefore:\n%s\nafter:\n%s",
			strings.Join(parentBefore, "\n"), strings.Join(parentAfter, "\n"))
	}

	// 驗證 extraArgs 原樣轉發給子程序。
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args output: %v", err)
	}
	gotArgs := strings.Split(string(data), "\n")
	// 空白行過濾
	var got []string
	for _, a := range gotArgs {
		if a != "" {
			got = append(got, a)
		}
	}
	if len(got) != len(extraArgs) {
		t.Fatalf("child received %d args, want %d: %v", len(got), len(extraArgs), got)
	}
	for i, want := range extraArgs {
		if got[i] != want {
			t.Errorf("child arg[%d] = %q, want %q", i, got[i], want)
		}
	}
}

// TestLaunchIntegration_NoExtraArgs 驗證不傳 extraArgs 時子程序
// 收到零命令列參數，行為與舊版一致。
func TestLaunchIntegration_NoExtraArgs(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	argsFile := filepath.Join(t.TempDir(), "args.txt")
	outFile := filepath.Join(t.TempDir(), "env.txt")

	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	profile := &config.Profile{
		Name:         "openai-official",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey: "sk-test-integration",
		Models: []string{"gpt-4o"},
	}

	var stdout, stderr strings.Builder
	if err := Launch(profile, "gpt-4o", stub, nil, nil, &stdout, &stderr); err != nil {
		t.Fatalf("Launch failed: %v (stderr=%s)", err, stderr.String())
	}

	// 不傳 extraArgs 時，參數輸出檔應為空（零命令列參數）。
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read stub args output: %v", err)
	}
	if len(strings.TrimSpace(string(data))) > 0 {
		t.Errorf("expected zero args, got: %s", string(data))
	}
}

// buildStub 將 testdata/stub 程式編譯為暫存二進位檔並回傳其絕對路徑。
// 它使用 `go build`，使 stub 成為現行平台上的真實可執行檔。
func buildStub(t *testing.T, srcDir string) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "copilot")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", exe, ".")
	cmd.Dir = srcDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build stub: %v\n%s", err, out)
	}
	return exe
}

// snapshotEnv 回傳現行程序環境的排序後副本。
func snapshotEnv() []string {
	env := append([]string(nil), os.Environ()...)
	sort.Strings(env)
	return env
}

// envEqual 比較兩個排序後的環境切片是否相等。
func envEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// envLookup 在 "KEY=VALUE" 切片中尋找 KEY 並回傳 VALUE，找不到則
// 回傳 ""。
func envLookup(env []string, key string) string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return strings.TrimPrefix(e, prefix)
		}
	}
	return ""
}