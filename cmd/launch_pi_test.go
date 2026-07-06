package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// TestRunLaunchPi_MissingConfigFile 驗證設定檔不存在時印出提示並 exit 1。
func TestRunLaunchPi_MissingConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	var stdout, stderr bytes.Buffer
	err := runLaunchPi(path, "", "", nil, &stdout, &stderr)
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

// TestRunLaunchPi_MissingProfile 驗證具名 profile 不存在時列出可用 profile 並 exit 1。
func TestRunLaunchPi_MissingProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchPi(path, "nonexistent", "", nil, &stdout, &stderr)
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

// TestRunLaunchPi_NonOpenaiProviderRejected 驗證非 openai provider 被拒。
func TestRunLaunchPi_NonOpenaiProviderRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: azure-prod\n    provider: azure\n    api_base: https://example.openai.azure.com\n    api_key: az-key\n    default_model: gpt-4o\ndefault_profile: azure-prod\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchPi(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "僅支援 openai provider") {
		t.Errorf("stderr missing provider rejection, got: %s", stderr.String())
	}
}

// TestRunLaunchPi_NoDefaultProfile 驗證未指定 profile 且未設定 default_profile 時 exit 1。
func TestRunLaunchPi_NoDefaultProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchPi(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "default_profile") {
		t.Errorf("stderr missing default_profile hint, got: %s", stderr.String())
	}
}

// TestRunLaunchPi_NotInstalled 驗證 pi 不在 PATH 時印出安裝提示並 exit 1。
func TestRunLaunchPi_NotInstalled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	t.Setenv("PATH", "")
	var stdout, stderr bytes.Buffer
	err := runLaunchPi(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), `找不到 "pi" 可執行檔`) {
		t.Errorf("stderr missing pi not-found message, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "pi.dev") {
		t.Errorf("stderr missing install hint with pi.dev link, got: %s", stderr.String())
	}
}

// TestRunLaunchPi_ParentEnvUnchanged 驗證錯誤路徑（pi 未安裝）後
// 父程序環境保持不變。
func TestRunLaunchPi_ParentEnvUnchanged(t *testing.T) {
	t.Setenv("BYOK_PARENT_CHECK", "intact")
	t.Setenv("PI_CODING_AGENT_DIR", "")
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	t.Setenv("PATH", "")
	var stdout, stderr bytes.Buffer
	_ = runLaunchPi(path, "", "", nil, &stdout, &stderr)
	if got := envLookupOS("BYOK_PARENT_CHECK"); got != "intact" {
		t.Errorf("parent BYOK_PARENT_CHECK = %q, want %q", got, "intact")
	}
	if got := envLookupOS("PI_CODING_AGENT_DIR"); got != "" {
		t.Errorf("parent PI_CODING_AGENT_DIR leaked = %q (must be empty)", got)
	}
}

// TestRunLaunchPi_EnvInjected 驗證正常啟動路徑下子程序接收正確的
// PI_CODING_AGENT_DIR 且臨時目錄含正確 models.json。
func TestRunLaunchPi_EnvInjected(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-pi-test\n    default_model: gpt-4o\ndefault_profile: openai-official\n")

	stubExe := buildStubForPi(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	modelsFile := filepath.Join(t.TempDir(), "models.json")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)
	t.Setenv("BYOK_STUB_MODELS_OUT", modelsFile)

	var stdout, stderr bytes.Buffer
	if err := runLaunchPi(path, "", "", nil, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchPi failed: %v (stderr=%s)", err, stderr.String())
	}

	envData, _ := os.ReadFile(outFile)
	childEnv := strings.Split(string(envData), "\n")
	piDir := envLookupOSIn(childEnv, "PI_CODING_AGENT_DIR")
	if piDir == "" {
		t.Fatalf("child env missing PI_CODING_AGENT_DIR")
	}

	modelsData, _ := os.ReadFile(modelsFile)
	var models map[string]map[string]map[string]string
	if err := json.Unmarshal(modelsData, &models); err != nil {
		t.Fatalf("unmarshal models.json: %v", err)
	}
	if got := models["providers"]["openai"]["baseUrl"]; got != "https://api.openai.com/v1" {
		t.Errorf("models.json baseUrl = %q, want %q", got, "https://api.openai.com/v1")
	}
	if got := models["providers"]["openai"]["apiKey"]; got != "sk-pi-test" {
		t.Errorf("models.json apiKey = %q, want %q", got, "sk-pi-test")
	}

	argsData, _ := os.ReadFile(argsFile)
	gotArgs := splitNonEmpty(string(argsData))
	if len(gotArgs) < 2 || gotArgs[0] != "--model" || gotArgs[1] != "gpt-4o" {
		t.Errorf("child args = %v, want [--model gpt-4o] at front", gotArgs)
	}
}

// TestRunLaunchPi_ModelOverride 驗證 --model 覆寫 default_model。
func TestRunLaunchPi_ModelOverride(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-pi-test\n    default_model: gpt-4o\ndefault_profile: openai-official\n")

	stubExe := buildStubForPi(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	if err := runLaunchPi(path, "", "o4-mini", nil, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchPi failed: %v (stderr=%s)", err, stderr.String())
	}

	argsData, _ := os.ReadFile(argsFile)
	gotArgs := splitNonEmpty(string(argsData))
	if len(gotArgs) < 2 || gotArgs[0] != "--model" || gotArgs[1] != "o4-mini" {
		t.Errorf("child args = %v, want [--model o4-mini] at front", gotArgs)
	}
}

// TestRunLaunchPi_ProfileSelect 驗證 --profile 選擇非預設 profile。
func TestRunLaunchPi_ProfileSelect(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-official\n    default_model: gpt-4o\n  - name: custom-endpoint\n    provider: openai\n    api_base: https://custom.example.com/v1\n    api_key: sk-custom\n    default_model: gpt-4o-mini\ndefault_profile: openai-official\n")

	stubExe := buildStubForPi(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	modelsFile := filepath.Join(t.TempDir(), "models.json")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)
	t.Setenv("BYOK_STUB_MODELS_OUT", modelsFile)

	var stdout, stderr bytes.Buffer
	if err := runLaunchPi(path, "custom-endpoint", "", nil, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchPi failed: %v (stderr=%s)", err, stderr.String())
	}

	modelsData, _ := os.ReadFile(modelsFile)
	var models map[string]map[string]map[string]string
	if err := json.Unmarshal(modelsData, &models); err != nil {
		t.Fatalf("unmarshal models.json: %v", err)
	}
	if got := models["providers"]["openai"]["baseUrl"]; got != "https://custom.example.com/v1" {
		t.Errorf("models.json baseUrl = %q, want %q", got, "https://custom.example.com/v1")
	}
}

// TestRunLaunchPi_Passthrough 驗證 -- 後的參數原樣透傳給 pi 子程序。
func TestRunLaunchPi_Passthrough(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-pi-test\n    default_model: gpt-4o\ndefault_profile: openai-official\n")

	stubExe := buildStubForPi(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	extraArgs := []string{"--", "fix this bug"}
	if err := runLaunchPi(path, "", "", extraArgs, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchPi failed: %v (stderr=%s)", err, stderr.String())
	}

	argsData, _ := os.ReadFile(argsFile)
	gotArgs := splitNonEmpty(string(argsData))
	wantArgs := []string{"--model", "gpt-4o", "--", "fix this bug"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("child args len = %d, want %d: %v", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i, w := range wantArgs {
		if gotArgs[i] != w {
			t.Errorf("child arg[%d] = %q, want %q", i, gotArgs[i], w)
		}
	}
}

// TestRunLaunchPi_YoloFlag 驗證 --yolo 對 pi 映射為 --approve。
func TestRunLaunchPi_YoloFlag(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-pi-test\n    default_model: gpt-4o\ndefault_profile: openai-official\n")

	stubExe := buildStubForPi(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	extraArgs := []string{"--approve"}
	if err := runLaunchPi(path, "", "", extraArgs, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchPi failed: %v (stderr=%s)", err, stderr.String())
	}

	argsData, _ := os.ReadFile(argsFile)
	gotArgs := splitNonEmpty(string(argsData))
	wantArgs := []string{"--model", "gpt-4o", "--approve"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("child args len = %d, want %d: %v", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i, w := range wantArgs {
		if gotArgs[i] != w {
			t.Errorf("child arg[%d] = %q, want %q", i, gotArgs[i], w)
		}
	}
}

// TestRunLaunchPi_YoloWithPassthrough 驗證 --approve 在透傳參數之前。
func TestRunLaunchPi_YoloWithPassthrough(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-pi-test\n    default_model: gpt-4o\ndefault_profile: openai-official\n")

	stubExe := buildStubForPi(t)
	t.Setenv("PATH", filepath.Dir(stubExe))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	var stdout, stderr bytes.Buffer
	extraArgs := []string{"--approve", "--", "fix this bug"}
	if err := runLaunchPi(path, "", "", extraArgs, &stdout, &stderr); err != nil {
		t.Fatalf("runLaunchPi failed: %v (stderr=%s)", err, stderr.String())
	}

	argsData, _ := os.ReadFile(argsFile)
	gotArgs := splitNonEmpty(string(argsData))
	wantArgs := []string{"--model", "gpt-4o", "--approve", "--", "fix this bug"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("child args len = %d, want %d: %v", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i, w := range wantArgs {
		if gotArgs[i] != w {
			t.Errorf("child arg[%d] = %q, want %q", i, gotArgs[i], w)
		}
	}
}

// buildStubForPi 編譯 testdata stub 至暫存目錄，命名為 pi
// （Windows 加 .exe），供 runLaunchPi 的 LookPath("pi") 解析。
func buildStubForPi(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "pi")
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