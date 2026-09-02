package runner

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// BuildPiEnv 回傳環境切片（os.Environ() 形式的 "KEY=VALUE" 字串），
// 適合指定給 exec.Cmd.Env。它以現行程序環境為起點，覆寫
// PI_CODING_AGENT_DIR=<tempDir>，其餘環境變數保持不變。
// 父程序環境永不被修改。
//
// 與 BuildClaudeEnv 不同，pi 的 base URL 與 API key 透過臨時目錄中的
// models.json 注入（由 LaunchPi 負責），而非環境變數，因此 profile
// 參數在此函式中不參與環境組裝。
func BuildPiEnv(profile *config.Profile, tempDir string, telemetry *config.Telemetry) []string {
	_ = profile

	env := make([]string, 0, len(os.Environ())+1)

	for _, entry := range os.Environ() {
		key := entry
		if i := strings.IndexByte(entry, '='); i >= 0 {
			key = entry[:i]
		}
		if key == "PI_CODING_AGENT_DIR" {
			continue
		}
		env = append(env, entry)
	}

	env = append(env, "PI_CODING_AGENT_DIR="+tempDir)

	// Telemetry injection: Pi 僅支援 HTTP。
	if telemetry != nil && telemetry.Enabled && telemetry.HTTP != nil && telemetry.HTTP.Endpoint != "" {
		env = append(env,
			"PI_OTEL_ENABLED=true",
			"OTEL_EXPORTER_OTLP_ENDPOINT="+telemetry.HTTP.Endpoint,
		)
		if sn := config.ComposeServiceName(telemetry.ServiceName, "pi"); sn != "" {
			env = append(env, "OTEL_SERVICE_NAME="+sn)
		}
	}

	return env
}

// LaunchPi 以臨時目錄 + models.json + PI_CODING_AGENT_DIR 環境變數
// 啟動 exePath 指向的 pi 可執行檔為子程序。它會：
//  1. 建立臨時目錄（os.MkdirTemp）
//  2. 寫入 models.json（{"providers":{"openai":{"baseUrl":"...","apiKey":"..."}}}）；
//     limits 提供 context 與/或 output 有效值時，另寫出以所選模型為鍵的
//     providers.openai.modelOverrides["<model>"]（contextWindow/maxTokens，
//     未設定欄位的鍵不寫出）
//  3. MaxOutputTokens 有效時寫入 settings.json（{"compaction":{"reserveTokens":N}}），
//     為 pi 輸出量保留回應 headroom；未設定時不建立 settings.json
//  4. 以 BuildPiEnv 組裝子程序環境
//  5. 將 --model <model（呼叫端已解析的單一模型字串）> 附加到 extraArgs 前端
//  6. 啟動子程序，連接 stdin/stdout/stderr
//  7. defer os.RemoveAll 清理臨時目錄
//  8. models.json 與 settings.json 權限皆為 0600
//
// 父程序環境與使用者 pi 設定檔永不被修改。模型解析（候選清單選擇）
// 由呼叫端（cmd/launch 層）完成。
func buildPiArgs(model, effort string, extraArgs []string) []string {
	args := []string{"--model", model}
	if effort != "" {
		args = append(args, "--thinking", effort)
	}
	return append(args, extraArgs...)
}

// piModelOverrides 依 limits 組裝 providers.openai.modelOverrides 物件。
// limits 為 nil 或兩欄位皆未設定時回傳 nil（providers.openai 不寫出
// modelOverrides 鍵）；僅設定的欄位會出現，未設定欄位的鍵完全不寫出。
func piModelOverrides(limits *TokenLimits) map[string]any {
	if limits == nil {
		return nil
	}
	var overrides map[string]any
	if limits.ContextWindowTokens != nil {
		if overrides == nil {
			overrides = map[string]any{}
		}
		overrides["contextWindow"] = *limits.ContextWindowTokens
	}
	if limits.MaxOutputTokens != nil {
		if overrides == nil {
			overrides = map[string]any{}
		}
		overrides["maxTokens"] = *limits.MaxOutputTokens
	}
	return overrides
}

// writePiSettings 在 MaxOutputTokens 有效時寫入暫存 settings.json
// （compaction.reserveTokens = output 值），為 pi 輸出量保留回應
// headroom；output 未設定時不建立檔案。
func writePiSettings(tempDir string, limits *TokenLimits) error {
	if limits == nil || limits.MaxOutputTokens == nil {
		return nil
	}
	settings := map[string]any{
		"compaction": map[string]any{
			"reserveTokens": *limits.MaxOutputTokens,
		},
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(tempDir, "settings.json"), data, 0600)
}

func LaunchPi(profile *config.Profile, model string, limits *TokenLimits, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer, telemetry *config.Telemetry, effort ...string) error {
	tempDir, err := os.MkdirTemp("", "byok-pi-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	provider := map[string]any{
		"baseUrl": profile.APIBase,
		"apiKey":  profile.APIKey,
	}
	if overrides := piModelOverrides(limits); overrides != nil {
		provider["modelOverrides"] = map[string]any{model: overrides}
	}
	models := map[string]any{
		"providers": map[string]any{
			"openai": provider,
		},
	}
	data, err := json.Marshal(models)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tempDir, "models.json"), data, 0600); err != nil {
		return err
	}
	if err := writePiSettings(tempDir, limits); err != nil {
		return err
	}

	e := ""
	if len(effort) > 0 {
		e = effort[0]
	}
	args := buildPiArgs(model, e, extraArgs)

	cmd := exec.Command(exePath, args...)
	cmd.Env = BuildPiEnv(profile, tempDir, telemetry)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
