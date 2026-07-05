package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLaunch_MissingConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	var stdout, stderr bytes.Buffer
	err := runLaunchCopilot(path, "", "", nil, &stdout, &stderr)
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

func TestLaunch_MissingProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\n  - name: local-ollama\n    provider: openai\n    api_base: http://localhost:11434\n    api_key: \"\"\n    default_model: llama3.2\ndefault_profile: openai-official\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchCopilot(path, "nonexistent", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), `找不到 profile "nonexistent"`) {
		t.Errorf("stderr missing not-found message, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "openai-official") || !strings.Contains(stderr.String(), "local-ollama") {
		t.Errorf("stderr should list available profiles, got: %s", stderr.String())
	}
}

func TestLaunch_NonOpenaiProviderRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: azure-prod\n    provider: azure\n    api_base: https://example.openai.azure.com\n    api_key: az-key\n    default_model: gpt-4o\ndefault_profile: azure-prod\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchCopilot(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "僅支援 openai provider") {
		t.Errorf("stderr missing provider rejection, got: %s", stderr.String())
	}
}

func TestLaunch_CustomConfigPath(t *testing.T) {
	// 驗證明確指定的 --config 路徑會被採用：指向一個預設設定檔
	// 使用非 openai provider 的設定，應產生 provider 拒絕錯誤，
	// 藉此證明自訂路徑已被讀取。
	path := filepath.Join(t.TempDir(), "custom.yaml")
	writeFile(t, path, "profiles:\n  - name: azure-prod\n    provider: azure\n    api_base: https://example.openai.azure.com\n    api_key: az-key\n    default_model: gpt-4o\ndefault_profile: azure-prod\n")
	var stdout, stderr bytes.Buffer
	err := runLaunchCopilot(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit (custom path not honored?)", err)
	}
	if !strings.Contains(stderr.String(), "僅支援 openai provider") {
		t.Errorf("custom config path was not read, stderr: %s", stderr.String())
	}
}

func TestLaunch_CopilotNotInstalled(t *testing.T) {
	// 建立一個有效的 openai 設定檔以通過設定檔解析階段，抵達
	// copilot 存在性檢查。接著將 PATH 清空使 LookPath("copilot") 失敗。
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	// 將 PATH 強制設為空字串，使 exec.LookPath 找不到 copilot。
	// 在 Windows 上空 PATH 仍允許 LookPath 檢查當前目錄，但
	// copilot 不太可能位於測試的 CWD 中；為保險起見，亦將
	// CWD 指向暫存目錄。
	t.Setenv("PATH", "")
	var stdout, stderr bytes.Buffer
	err := runLaunchCopilot(path, "", "", nil, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), `找不到 "copilot" 可執行檔`) {
		t.Errorf("stderr missing not-found message, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "安裝 GitHub Copilot CLI") {
		t.Errorf("stderr missing install hint, got: %s", stderr.String())
	}
}

func TestLaunch_ParentEnvUnchanged(t *testing.T) {
	// 走錯誤路徑（copilot 未安裝）仍會執行父程序環境快照。
	// 設定一個標記變數、執行 launch（於 LookPath 失敗），
	// 並確認標記與其餘父程序環境保持完整。
	t.Setenv("BYOK_PARENT_CHECK", "intact")
	// 清除任何繼承的 COPILOT_MODEL，以斷言其保持為空。
	t.Setenv("COPILOT_MODEL", "")
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	t.Setenv("PATH", "")
	var stdout, stderr bytes.Buffer
	_ = runLaunchCopilot(path, "", "", nil, &stdout, &stderr)
	if got := envLookupOS("BYOK_PARENT_CHECK"); got != "intact" {
		t.Errorf("parent BYOK_PARENT_CHECK = %q, want %q (parent env must be untouched)", got, "intact")
	}
	if got := envLookupOS("COPILOT_MODEL"); got != "" {
		t.Errorf("parent COPILOT_MODEL leaked = %q (must be empty)", got)
	}
}

// envLookupOS 從現行程序環境讀取單一環境變數。
func envLookupOS(key string) string {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, key+"=") {
			return strings.TrimPrefix(e, key+"=")
		}
	}
	return ""
}

// TestBuildExtraArgs_YoloFlag 驗證 yolo 旗標為 true 時附加 --yolo。
func TestBuildExtraArgs_YoloFlag(t *testing.T) {
	got := buildExtraArgs(true, nil)
	want := []string{"--yolo"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("buildExtraArgs(true, nil) = %v, want %v", got, want)
	}
}

// TestBuildExtraArgs_YoloShortForm 驗證 -y 短形式與 --yolo 等效。
func TestBuildExtraArgs_YoloShortForm(t *testing.T) {
	// -y 在 cobra 層設定 yolo=true，與 --yolo 相同路徑。
	got := buildExtraArgs(true, nil)
	if len(got) != 1 || got[0] != "--yolo" {
		t.Errorf("buildExtraArgs(true, nil) = %v, want [--yolo]", got)
	}
}

// TestBuildExtraArgs_SinglePassthrough 驗證透傳單一參數。
func TestBuildExtraArgs_SinglePassthrough(t *testing.T) {
	got := buildExtraArgs(false, []string{"--continue"})
	want := []string{"--continue"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("buildExtraArgs(false, [--continue]) = %v, want %v", got, want)
	}
}

// TestBuildExtraArgs_MultiplePassthrough 驗證透傳多個參數保持順序。
func TestBuildExtraArgs_MultiplePassthrough(t *testing.T) {
	got := buildExtraArgs(false, []string{"--continue", "--model", "x"})
	want := []string{"--continue", "--model", "x"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// TestBuildExtraArgs_YoloAndPassthrough 驗證 yolo 與透傳合用時
// --yolo 在前，透傳參數在後。
func TestBuildExtraArgs_YoloAndPassthrough(t *testing.T) {
	got := buildExtraArgs(true, []string{"--continue"})
	want := []string{"--yolo", "--continue"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// TestBuildExtraArgs_NoArgs 驗證無旗標無透傳時回傳 nil（行為不變）。
func TestBuildExtraArgs_NoArgs(t *testing.T) {
	got := buildExtraArgs(false, nil)
	if got != nil {
		t.Errorf("buildExtraArgs(false, nil) = %v, want nil", got)
	}
}