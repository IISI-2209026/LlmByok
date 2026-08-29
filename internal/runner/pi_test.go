package runner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// TestBuildPiEnv_SetsPiCodingAgentDir 驗證 BuildPiEnv 正確設定
// PI_CODING_AGENT_DIR 環境變數指向臨時目錄，且父程序環境不被修改。
func TestBuildPiEnv_SetsPiCodingAgentDir(t *testing.T) {
	profile := config.Profile{
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	tempDir := "/tmp/pi-byok-test-dir"

	parentBefore := snapshotEnv()

	env := BuildPiEnv(&profile, tempDir, nil)

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
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	tempDir := "/new/pi/dir"

	env := BuildPiEnv(&profile, tempDir, nil)

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
		Name:     "p",
		Provider: "openai",
		APIBase:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Models:   []config.Model{{Name: "gpt-4o"}},
	}
	env := BuildPiEnv(&profile, "/tmp/pi-dir", nil)

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
		Models:   []config.Model{{Name: "gpt-4o"}},
	}

	var stdout, stderr strings.Builder
	if err := LaunchPi(profile, "gpt-4o", nil, stub, nil, nil, &stdout, &stderr, nil); err != nil {
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
		Models:   []config.Model{{Name: "gpt-4o"}},
	}

	var stdout, stderr strings.Builder
	if err := LaunchPi(profile, "o4-mini", nil, stub, nil, nil, &stdout, &stderr, nil); err != nil {
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
		Models:   []config.Model{{Name: "gpt-4o"}},
	}

	var stdout, stderr strings.Builder
	if err := LaunchPi(profile, "gpt-4o", nil, stub, nil, nil, &stdout, &stderr, nil); err != nil {
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
		Models:   []config.Model{{Name: "gpt-4o"}},
	}

	var stdout, stderr strings.Builder
	extra := []string{"--approve", "fix this bug"}
	if err := LaunchPi(profile, "gpt-4o", nil, stub, extra, nil, &stdout, &stderr, nil); err != nil {
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
		Models:   []config.Model{{Name: "gpt-4o"}},
	}

	var stdout, stderr strings.Builder
	_ = LaunchPi(profile, "gpt-4o", nil, stub, nil, nil, &stdout, &stderr, nil)

	if got := os.Getenv("BYOK_PARENT_MARKER"); got != "before" {
		t.Errorf("BYOK_PARENT_MARKER = %q, want %q", got, "before")
	}
	if got := os.Getenv("PI_CODING_AGENT_DIR"); got != "" {
		t.Errorf("parent PI_CODING_AGENT_DIR = %q, want empty", got)
	}
}

// piProbeStubSource 是 Pi token limit 測試使用的子程序 stub 原始碼。
// 它把 PI_CODING_AGENT_DIR 下每個檔案的名稱、權限位元與內容以 JSON
// 陣列寫入 BYOK_PROBE_REPORT 指定的路徑，讓測試能在 LaunchPi defer
// 掉暫存目錄後仍能驗證 models.json / settings.json 的 JSON 結構與
// 0600 檔案權限；並可透過 BYOK_PROBE_EXIT 以指定結束碼退出，
// 供失敗清理路徑測試使用。
const piProbeStubSource = `package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	out := os.Getenv("BYOK_STUB_OUT")
	if out != "" {
		env := append([]string(nil), os.Environ()...)
		sort.Strings(env)
		_ = os.WriteFile(out, []byte(strings.Join(env, "\n")), 0600)
	}

	dir := os.Getenv("PI_CODING_AGENT_DIR")
	report := os.Getenv("BYOK_PROBE_REPORT")
	if report != "" && dir != "" {
		entries, _ := os.ReadDir(dir)
		files := []map[string]any{}
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			files = append(files, map[string]any{
				"name":    entry.Name(),
				"mode":    uint32(info.Mode().Perm()),
				"content": string(data),
			})
		}
		if data, err := json.Marshal(files); err == nil {
			_ = os.WriteFile(report, data, 0600)
		}
	}

	if v := os.Getenv("BYOK_PROBE_EXIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			os.Exit(n)
		}
	}
}
`

// buildPiProbeStub 將 piProbeStubSource 編譯為暫存執行檔並回傳其路徑。
func buildPiProbeStub(t *testing.T) string {
	t.Helper()
	src := filepath.Join(t.TempDir(), "pi_probe_stub.go")
	if err := os.WriteFile(src, []byte(piProbeStubSource), 0600); err != nil {
		t.Fatalf("write pi probe stub source: %v", err)
	}
	exe := filepath.Join(t.TempDir(), "pi-probe")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", exe, src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build pi probe stub: %v\n%s", err, out)
	}
	return exe
}

// piProbeFile 描述 probe stub 回報的暫存目錄檔案（名稱、權限、內容）。
type piProbeFile struct {
	Name    string `json:"name"`
	Mode    uint32 `json:"mode"`
	Content string `json:"content"`
}

// readPiProbeFiles 讀取 probe stub 寫出的檔案回報。
func readPiProbeFiles(t *testing.T, reportPath string) []piProbeFile {
	t.Helper()
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read probe report: %v", err)
	}
	var files []piProbeFile
	if err := json.Unmarshal(data, &files); err != nil {
		t.Fatalf("unmarshal probe report: %v", err)
	}
	return files
}

// findProbeFile 依名稱尋找 probe 回報中的檔案。
func findProbeFile(files []piProbeFile, name string) (piProbeFile, bool) {
	for _, f := range files {
		if f.Name == name {
			return f, true
		}
	}
	return piProbeFile{}, false
}

// piProbeProfile 回傳測試用 profile；模型名稱含點號以驗證
// modelOverrides 以精確字串作為 JSON 物件鍵。
func piProbeProfile() *config.Profile {
	return &config.Profile{
		Name:     "pi-probe",
		Provider: "openai",
		APIBase:  "https://api.example.com/v1",
		APIKey:   "sk-pi-probe",
		Models:   []config.Model{{Name: "gpt-5.4"}},
	}
}

// launchPiWithProbe 以 probe stub 成功執行 LaunchPi，回傳 probe 回報
// 的暫存檔清單與子程序收到的 PI_CODING_AGENT_DIR 路徑。
func launchPiWithProbe(t *testing.T, limits *TokenLimits, model string) (files []piProbeFile, piDir string) {
	t.Helper()
	probe := buildPiProbeStub(t)

	tmp := t.TempDir()
	envOut := filepath.Join(tmp, "env.txt")
	reportPath := filepath.Join(tmp, "probe.json")
	t.Setenv("BYOK_STUB_OUT", envOut)
	t.Setenv("BYOK_PROBE_REPORT", reportPath)

	var stdout, stderr strings.Builder
	if err := LaunchPi(piProbeProfile(), model, limits, probe, nil, nil, &stdout, &stderr, nil); err != nil {
		t.Fatalf("LaunchPi failed: %v (stderr=%s)", err, stderr.String())
	}

	envData, err := os.ReadFile(envOut)
	if err != nil {
		t.Fatalf("read stub env output: %v", err)
	}
	piDir = envLookup(strings.Split(string(envData), "\n"), "PI_CODING_AGENT_DIR")
	if piDir == "" {
		t.Fatalf("child env missing PI_CODING_AGENT_DIR")
	}

	return readPiProbeFiles(t, reportPath), piDir
}

// piOverrideShape 描述 providers.openai.modelOverrides[<model>] 物件。
type piOverrideShape struct {
	ContextWindow *int64 `json:"contextWindow"`
	MaxTokens     *int64 `json:"maxTokens"`
}

// decodePiModelsOverride 解析 probe 回報的 models.json，回傳
// providers.openai.modelOverrides[<model>] 物件（未設定 modelOverrides
// 時回傳 false）。
func decodePiModelsOverride(t *testing.T, content, model string) (piOverrideShape, bool) {
	t.Helper()
	var models struct {
		Providers map[string]struct {
			BaseURL        string                     `json:"baseUrl"`
			APIKey         string                     `json:"apiKey"`
			ModelOverrides map[string]piOverrideShape `json:"modelOverrides"`
		} `json:"providers"`
	}
	if err := json.Unmarshal([]byte(content), &models); err != nil {
		t.Fatalf("unmarshal models.json %s: %v", content, err)
	}
	openai, ok := models.Providers["openai"]
	if !ok {
		t.Fatalf("models.json missing providers.openai: %s", content)
	}
	override, ok := openai.ModelOverrides[model]
	if !ok {
		return piOverrideShape{}, false
	}
	return override, true
}

// TestLaunchPi_TokenLimits_WritesModelOverridesAndSettings 驗證 context 與
// output 限制同時設定時，暫存 models.json 含
// providers.openai.modelOverrides["gpt-5.4"]（contextWindow/maxTokens）、
// settings.json 含 compaction.reserveTokens，兩檔案權限皆為 0600，
// 且子程序結束後整個暫存目錄被清理。
func TestLaunchPi_TokenLimits_WritesModelOverridesAndSettings(t *testing.T) {
	limits := &TokenLimits{
		ContextWindowTokens: ptrLmt64(1000000),
		MaxOutputTokens:     ptrLmt64(128000),
	}

	files, piDir := launchPiWithProbe(t, limits, "gpt-5.4")

	modelsFile, ok := findProbeFile(files, "models.json")
	if !ok {
		t.Fatalf("probe report missing models.json; got %v", files)
	}
	override, ok := decodePiModelsOverride(t, modelsFile.Content, "gpt-5.4")
	if !ok {
		t.Fatalf("modelOverrides must contain exact key \"gpt-5.4\"; models.json=%s", modelsFile.Content)
	}
	if override.ContextWindow == nil || *override.ContextWindow != 1000000 {
		t.Errorf("modelOverrides[\"gpt-5.4\"].contextWindow = %v, want 1000000", override.ContextWindow)
	}
	if override.MaxTokens == nil || *override.MaxTokens != 128000 {
		t.Errorf("modelOverrides[\"gpt-5.4\"].maxTokens = %v, want 128000", override.MaxTokens)
	}

	settingsFile, ok := findProbeFile(files, "settings.json")
	if !ok {
		t.Fatalf("probe report missing settings.json; got %v", files)
	}
	var settings struct {
		Compaction struct {
			ReserveTokens *int64 `json:"reserveTokens"`
		} `json:"compaction"`
	}
	if err := json.Unmarshal([]byte(settingsFile.Content), &settings); err != nil {
		t.Fatalf("unmarshal settings.json %s: %v", settingsFile.Content, err)
	}
	if settings.Compaction.ReserveTokens == nil || *settings.Compaction.ReserveTokens != 128000 {
		t.Errorf("settings.json compaction.reserveTokens = %v, want 128000", settings.Compaction.ReserveTokens)
	}

	// 兩個暫存檔權限必須為 0600（probe 保留原始 stat 權限）。
	if modelsFile.Mode != uint32(0600) {
		t.Errorf("models.json mode = %o, want 600", modelsFile.Mode)
	}
	if settingsFile.Mode != uint32(0600) {
		t.Errorf("settings.json mode = %o, want 600", settingsFile.Mode)
	}

	// 成功執行後整個暫存目錄必須被清理。
	if _, err := os.Stat(piDir); !os.IsNotExist(err) {
		t.Errorf("temp dir %q should not exist after LaunchPi returns", piDir)
	}
}

// TestLaunchPi_TokenLimits_ContextOnlyOmitsMaxTokensAndSettings 驗證僅
// 設定 context 限制時，override 只含 contextWindow、完全沒有 maxTokens
// 鍵，且不會建立 settings.json。
func TestLaunchPi_TokenLimits_ContextOnlyOmitsMaxTokensAndSettings(t *testing.T) {
	limits := &TokenLimits{ContextWindowTokens: ptrLmt64(272000)}

	files, _ := launchPiWithProbe(t, limits, "gpt-5.4")

	modelsFile, ok := findProbeFile(files, "models.json")
	if !ok {
		t.Fatalf("probe report missing models.json; got %v", files)
	}
	override, ok := decodePiModelsOverride(t, modelsFile.Content, "gpt-5.4")
	if !ok {
		t.Fatalf("modelOverrides must contain exact key \"gpt-5.4\"; models.json=%s", modelsFile.Content)
	}
	if override.ContextWindow == nil || *override.ContextWindow != 272000 {
		t.Errorf("modelOverrides[\"gpt-5.4\"].contextWindow = %v, want 272000", override.ContextWindow)
	}
	if strings.Contains(modelsFile.Content, "\"maxTokens\"") {
		t.Errorf("models.json must not contain maxTokens key when output unset: %s", modelsFile.Content)
	}
	if override.MaxTokens != nil {
		t.Errorf("modelOverrides[\"gpt-5.4\"].maxTokens must be absent, got %v", *override.MaxTokens)
	}

	if _, ok := findProbeFile(files, "settings.json"); ok {
		t.Errorf("settings.json must not be created when output limit unset")
	}
}

// TestLaunchPi_TokenLimits_OutputOnlyWritesSettingsAndMaxTokens 驗證僅
// 設定 output 限制時，override 只含 maxTokens、完全沒有 contextWindow
// 鍵，settings.json 含 compaction.reserveTokens 且權限為 0600。
func TestLaunchPi_TokenLimits_OutputOnlyWritesSettingsAndMaxTokens(t *testing.T) {
	limits := &TokenLimits{MaxOutputTokens: ptrLmt64(128000)}

	files, _ := launchPiWithProbe(t, limits, "gpt-5.4")

	modelsFile, ok := findProbeFile(files, "models.json")
	if !ok {
		t.Fatalf("probe report missing models.json; got %v", files)
	}
	override, ok := decodePiModelsOverride(t, modelsFile.Content, "gpt-5.4")
	if !ok {
		t.Fatalf("modelOverrides must contain exact key \"gpt-5.4\"; models.json=%s", modelsFile.Content)
	}
	if override.MaxTokens == nil || *override.MaxTokens != 128000 {
		t.Errorf("modelOverrides[\"gpt-5.4\"].maxTokens = %v, want 128000", override.MaxTokens)
	}
	if strings.Contains(modelsFile.Content, "\"contextWindow\"") {
		t.Errorf("models.json must not contain contextWindow key when context unset: %s", modelsFile.Content)
	}
	if override.ContextWindow != nil {
		t.Errorf("modelOverrides[\"gpt-5.4\"].contextWindow must be absent, got %v", *override.ContextWindow)
	}

	settingsFile, ok := findProbeFile(files, "settings.json")
	if !ok {
		t.Fatalf("probe report missing settings.json; got %v", files)
	}
	var settings struct {
		Compaction struct {
			ReserveTokens *int64 `json:"reserveTokens"`
		} `json:"compaction"`
	}
	if err := json.Unmarshal([]byte(settingsFile.Content), &settings); err != nil {
		t.Fatalf("unmarshal settings.json %s: %v", settingsFile.Content, err)
	}
	if settings.Compaction.ReserveTokens == nil || *settings.Compaction.ReserveTokens != 128000 {
		t.Errorf("settings.json compaction.reserveTokens = %v, want 128000", settings.Compaction.ReserveTokens)
	}
	if settingsFile.Mode != uint32(0600) {
		t.Errorf("settings.json mode = %o, want 600", settingsFile.Mode)
	}
}

// TestLaunchPi_TokenLimits_UnsetKeepsProviderOnlyModelsJson 驗證 limits
// 全部未設定時，models.json 維持 provider-only（只有 baseUrl 與
// apiKey），完全沒有 modelOverrides 鍵，也不會建立 settings.json。
func TestLaunchPi_TokenLimits_UnsetKeepsProviderOnlyModelsJson(t *testing.T) {
	files, _ := launchPiWithProbe(t, nil, "gpt-5.4")

	modelsFile, ok := findProbeFile(files, "models.json")
	if !ok {
		t.Fatalf("probe report missing models.json; got %v", files)
	}
	if strings.Contains(modelsFile.Content, "modelOverrides") {
		t.Errorf("models.json must not contain modelOverrides when limits unset: %s", modelsFile.Content)
	}
	var models struct {
		Providers map[string]struct {
			BaseURL        string `json:"baseUrl"`
			APIKey         string `json:"apiKey"`
			ModelOverrides map[string]struct {
				ContextWindow *int64 `json:"contextWindow"`
				MaxTokens     *int64 `json:"maxTokens"`
			} `json:"modelOverrides"`
		} `json:"providers"`
	}
	if err := json.Unmarshal([]byte(modelsFile.Content), &models); err != nil {
		t.Fatalf("unmarshal models.json %s: %v", modelsFile.Content, err)
	}
	openai, ok := models.Providers["openai"]
	if !ok {
		t.Fatalf("models.json missing providers.openai: %s", modelsFile.Content)
	}
	if openai.BaseURL != "https://api.example.com/v1" {
		t.Errorf("models.json providers.openai.baseUrl = %q, want %q", openai.BaseURL, "https://api.example.com/v1")
	}
	if openai.APIKey != "sk-pi-probe" {
		t.Errorf("models.json providers.openai.apiKey = %q, want %q", openai.APIKey, "sk-pi-probe")
	}
	if openai.ModelOverrides != nil {
		t.Errorf("models.json providers.openai.modelOverrides must be absent, got %v", openai.ModelOverrides)
	}

	if _, ok := findProbeFile(files, "settings.json"); ok {
		t.Errorf("settings.json must not be created when output limit unset")
	}

	if modelsFile.Mode != uint32(0600) {
		t.Errorf("models.json mode = %o, want 600", modelsFile.Mode)
	}
}

// TestLaunchPi_CleansUpTempDirOnFailure 驗證 pi 子程序以非零結束碼
// 失敗時，LaunchPi 回傳後整個暫存目錄仍被完整移除。
func TestLaunchPi_CleansUpTempDirOnFailure(t *testing.T) {
	probe := buildPiProbeStub(t)

	tmp := t.TempDir()
	envOut := filepath.Join(tmp, "env.txt")
	reportPath := filepath.Join(tmp, "probe.json")
	t.Setenv("BYOK_STUB_OUT", envOut)
	t.Setenv("BYOK_PROBE_REPORT", reportPath)
	t.Setenv("BYOK_PROBE_EXIT", "3")

	var stdout, stderr strings.Builder
	if err := LaunchPi(piProbeProfile(), "gpt-5.4", nil, probe, nil, nil, &stdout, &stderr, nil); err == nil {
		t.Fatalf("LaunchPi should return an error when pi exits non-zero")
	}

	// 失敗前 stub 已回報暫存檔內容，確保檔案確實被寫入過。
	files, err := os.ReadFile(reportPath)
	if err != nil || len(files) == 0 {
		t.Fatalf("stub did not emit probe report before failing (err=%v): %q", err, files)
	}

	envData, err := os.ReadFile(envOut)
	if err != nil {
		t.Fatalf("read stub env output: %v", err)
	}
	piDir := envLookup(strings.Split(string(envData), "\n"), "PI_CODING_AGENT_DIR")
	if piDir == "" {
		t.Fatalf("child env missing PI_CODING_AGENT_DIR")
	}

	if _, err := os.Stat(piDir); !os.IsNotExist(err) {
		t.Errorf("temp dir %q should be removed even when pi fails", piDir)
	}
}

// listPiTempDirs 回傳目前 OS 暫存目錄下仍存在的 byok-pi-* 目錄集合。
func listPiTempDirs(t *testing.T) map[string]struct{} {
	t.Helper()
	set := map[string]struct{}{}
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "byok-pi-*"))
	if err != nil {
		t.Fatalf("glob pi temp dirs: %v", err)
	}
	for _, m := range matches {
		info, statErr := os.Stat(m)
		if statErr != nil || !info.IsDir() {
			continue
		}
		set[m] = struct{}{}
	}
	return set
}

// TestLaunchPi_CleansUpTempDirWhenExeMissing 驗證 exePath 不存在而
// LaunchPi 回傳錯誤時，暫存目錄同樣被完整清理。
func TestLaunchPi_CleansUpTempDirWhenExeMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-byok-pi-exe")

	limits := &TokenLimits{
		ContextWindowTokens: ptrLmt64(1000000),
		MaxOutputTokens:     ptrLmt64(128000),
	}

	// 以呼叫前後的集合差異判斷洩漏，排除其他行程殘留的舊目錄。
	before := listPiTempDirs(t)

	var stdout, stderr strings.Builder
	if err := LaunchPi(piProbeProfile(), "gpt-5.4", limits, missing, nil, nil, &stdout, &stderr, nil); err == nil {
		t.Fatalf("LaunchPi should return an error for nonexistent exePath")
	}

	for dir := range listPiTempDirs(t) {
		if _, existed := before[dir]; !existed {
			t.Errorf("temp dir left behind after failed LaunchPi: %s", dir)
		}
	}
}
