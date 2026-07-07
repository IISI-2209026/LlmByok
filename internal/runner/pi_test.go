package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// TestBuildPiEnv_SetsPiCodingAgentDir 驗證 BuildPiEnv 正確設定
// PI_CODING_AGENT_DIR 環境變數指向臨時目錄，且父程序環境不被修改。
func TestBuildPiEnv_SetsPiCodingAgentDir(t *testing.T) {
	profile := config.Profile{
		Name:         "p",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey: "sk-test",
		Models: []string{"gpt-4o"},
	}
	tempDir := "/tmp/pi-byok-test-dir"

	parentBefore := snapshotEnv()

	env := BuildPiEnv(&profile, tempDir)

	if got := getEnv(t, env, "PI_CODING_AGENT_DIR"); got != tempDir {
		t.Errorf("PI_CODING_AGENT_DIR = %q, want %q", got, tempDir)
	}

	parentAfter := snapshotEnv()
	if !envEqual(parentBefore, parentAfter) {
		t.Fatalf("parent environment changed after BuildPiEnv\nbefore:\n%s\nafter:\n%s",
			strings.Join(parentBefore, "\n"), strings.Join(parentAfter, "\n"))
	}
}

// TestBuildPiEnv_OverwritesExistingPiCodingAgentDir 驗證既存的
// PI_CODING_AGENT_DIR 值被覆寫為新的臨時目錄。
func TestBuildPiEnv_OverwritesExistingPiCodingAgentDir(t *testing.T) {
	t.Setenv("PI_CODING_AGENT_DIR", "/old/pi/dir")
	profile := config.Profile{
		Name:         "p",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey: "sk-test",
		Models: []string{"gpt-4o"},
	}
	tempDir := "/new/pi/dir"

	env := BuildPiEnv(&profile, tempDir)

	if got := getEnv(t, env, "PI_CODING_AGENT_DIR"); got != tempDir {
		t.Errorf("PI_CODING_AGENT_DIR = %q, want %q", got, tempDir)
	}
	if contains(env, "PI_CODING_AGENT_DIR=/old/pi/dir") {
		t.Errorf("env should not contain old PI_CODING_AGENT_DIR, got %v", env)
	}
}

// TestBuildPiEnv_PreservesOtherVars 驗證其他環境變數保持不變。
func TestBuildPiEnv_PreservesOtherVars(t *testing.T) {
	t.Setenv("BYOK_TEST_VAR", "hello")
	profile := config.Profile{
		Name:         "p",
		Provider:     "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey: "sk-test",
		Models: []string{"gpt-4o"},
	}
	env := BuildPiEnv(&profile, "/tmp/pi-dir")

	if !contains(env, "BYOK_TEST_VAR=hello") {
		t.Errorf("env missing preserved var BYOK_TEST_VAR=hello; got %v", env)
	}
}

// TestLaunchPi_CreatesTempDirWithModelsJson 驗證 LaunchPi 建立臨時目錄、
// 寫入正確的 models.json，且子程序命令列包含 --model <default_model>。
func TestLaunchPi_CreatesTempDirWithModelsJson(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	modelsFile := filepath.Join(t.TempDir(), "models.json")

	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)
	t.Setenv("BYOK_STUB_MODELS_OUT", modelsFile)

	t.Setenv("BYOK_PARENT_MARKER", "before")
	parentBefore := snapshotEnv()

	profile := &config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-pi-integration",
		Models:   []string{"gpt-4o"},
	}

	var stdout, stderr strings.Builder
	if err := LaunchPi(profile, "gpt-4o", stub, nil, nil, &stdout, &stderr); err != nil {
		t.Fatalf("LaunchPi failed: %v (stderr=%s)", err, stderr.String())
	}

	// 父程序環境必須保持不變。
	parentAfter := snapshotEnv()
	if !envEqual(parentBefore, parentAfter) {
		t.Fatalf("parent environment changed\nbefore:\n%s\nafter:\n%s",
			strings.Join(parentBefore, "\n"), strings.Join(parentAfter, "\n"))
	}

	// 子程序環境必須包含 PI_CODING_AGENT_DIR 指向臨時目錄。
	envData, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read stub env output: %v", err)
	}
	childEnv := strings.Split(string(envData), "\n")
	piDir := envLookup(childEnv, "PI_CODING_AGENT_DIR")
	if piDir == "" {
		t.Fatalf("child env missing PI_CODING_AGENT_DIR")
	}

	// models.json 必須包含正確的 provider override。
	modelsData, err := os.ReadFile(modelsFile)
	if err != nil {
		t.Fatalf("read models.json output: %v", err)
	}
	var models map[string]map[string]map[string]string
	if err := json.Unmarshal(modelsData, &models); err != nil {
		t.Fatalf("unmarshal models.json: %v", err)
	}
	if got := models["providers"]["openai"]["baseUrl"]; got != "https://api.openai.com/v1" {
		t.Errorf("models.json providers.openai.baseUrl = %q, want %q", got, "https://api.openai.com/v1")
	}
	if got := models["providers"]["openai"]["apiKey"]; got != "sk-pi-integration" {
		t.Errorf("models.json providers.openai.apiKey = %q, want %q", got, "sk-pi-integration")
	}

	// 命令列參數必須包含 --model <default_model>。
	argsData, _ := os.ReadFile(argsFile)
	gotArgs := splitArgs(string(argsData))
	wantArgs := []string{"--model", "gpt-4o"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("child args len = %d, want %d: %v", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i, w := range wantArgs {
		if gotArgs[i] != w {
			t.Errorf("child arg[%d] = %q, want %q", i, gotArgs[i], w)
		}
	}
}

// TestLaunchPi_OverrideModel 驗證傳入的 model 字串覆寫候選模型。
func TestLaunchPi_OverrideModel(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	profile := &config.Profile{
		Name:     "openai-official",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []string{"gpt-4o"},
	}

	var stdout, stderr strings.Builder
	if err := LaunchPi(profile, "o4-mini", stub, nil, nil, &stdout, &stderr); err != nil {
		t.Fatalf("LaunchPi failed: %v (stderr=%s)", err, stderr.String())
	}

	argsData, _ := os.ReadFile(argsFile)
	gotArgs := splitArgs(string(argsData))
	wantArgs := []string{"--model", "o4-mini"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("child args len = %d, want %d: %v", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i, w := range wantArgs {
		if gotArgs[i] != w {
			t.Errorf("child arg[%d] = %q, want %q", i, gotArgs[i], w)
		}
	}
}

// TestLaunchPi_CleansUpTempDir 驗證子程序結束後臨時目錄不存在。
func TestLaunchPi_CleansUpTempDir(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)

	profile := &config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []string{"gpt-4o"},
	}

	var stdout, stderr strings.Builder
	if err := LaunchPi(profile, "gpt-4o", stub, nil, nil, &stdout, &stderr); err != nil {
		t.Fatalf("LaunchPi failed: %v (stderr=%s)", err, stderr.String())
	}

	// 從子程序環境取得 PI_CODING_AGENT_DIR 路徑。
	envData, _ := os.ReadFile(outFile)
	childEnv := strings.Split(string(envData), "\n")
	piDir := envLookup(childEnv, "PI_CODING_AGENT_DIR")
	if piDir == "" {
		t.Fatalf("child env missing PI_CODING_AGENT_DIR")
	}

	if _, err := os.Stat(piDir); !os.IsNotExist(err) {
		t.Errorf("temp dir %q should not exist after LaunchPi returns", piDir)
	}
}

// TestLaunchPi_ExtraArgsPassthrough 驗證 extraArgs 原樣附加到 --model 之後。
func TestLaunchPi_ExtraArgsPassthrough(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_STUB_ARGS_OUT", argsFile)

	profile := &config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []string{"gpt-4o"},
	}

	var stdout, stderr strings.Builder
	extra := []string{"--approve", "fix this bug"}
	if err := LaunchPi(profile, "gpt-4o", stub, extra, nil, &stdout, &stderr); err != nil {
		t.Fatalf("LaunchPi failed: %v (stderr=%s)", err, stderr.String())
	}

	argsData, _ := os.ReadFile(argsFile)
	gotArgs := splitArgs(string(argsData))
	wantArgs := []string{"--model", "gpt-4o", "--approve", "fix this bug"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("child args len = %d, want %d: %v", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i, w := range wantArgs {
		if gotArgs[i] != w {
			t.Errorf("child arg[%d] = %q, want %q", i, gotArgs[i], w)
		}
	}
}

// TestLaunchPi_ParentEnvUnchanged 驗證 LaunchPi 後父程序環境不含
// PI_CODING_AGENT_DIR。
func TestLaunchPi_ParentEnvUnchanged(t *testing.T) {
	stub := buildStub(t, filepath.Join("testdata", "stub"))

	outFile := filepath.Join(t.TempDir(), "env.txt")
	t.Setenv("BYOK_STUB_OUT", outFile)
	t.Setenv("BYOK_PARENT_MARKER", "before")
	t.Setenv("PI_CODING_AGENT_DIR", "")

	profile := &config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []string{"gpt-4o"},
	}

	var stdout, stderr strings.Builder
	_ = LaunchPi(profile, "gpt-4o", stub, nil, nil, &stdout, &stderr)

	if got := os.Getenv("BYOK_PARENT_MARKER"); got != "before" {
		t.Errorf("BYOK_PARENT_MARKER = %q, want %q", got, "before")
	}
	if got := os.Getenv("PI_CODING_AGENT_DIR"); got != "" {
		t.Errorf("parent PI_CODING_AGENT_DIR = %q, want empty", got)
	}
}