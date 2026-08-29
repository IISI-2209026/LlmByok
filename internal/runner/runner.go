// Package runner 建置以 BYOK（Bring Your Own Key）provider 設定
// 啟動 Copilot CLI 所需的環境。
package runner

import (
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// TokenLimits 承載已解析的有效 token 限制（呼叫端完成優先序解析）。
// nil 指標欄位表示該欄位未設定 → runner 不得注入對應 override；
// runner 不決定設定優先序，也不產生預設值。
type TokenLimits struct {
	ContextWindowTokens *int64
	MaxOutputTokens     *int64
}

// byokKeys 是 BuildEnv 會從現行程序環境中覆寫的 Copilot BYOK 環境變數鍵名。
// token 兩鍵亦由 byok 管理：父環境的值會被移除，僅在有效值存在時以 byok
// 值重填（環境去重）。
var byokKeys = map[string]struct{}{
	"COPILOT_PROVIDER_BASE_URL":          {},
	"COPILOT_PROVIDER_TYPE":              {},
	"COPILOT_PROVIDER_API_KEY":           {},
	"COPILOT_MODEL":                      {},
	"COPILOT_PROVIDER_MAX_PROMPT_TOKENS": {},
	"COPILOT_PROVIDER_MAX_OUTPUT_TOKENS": {},
}

// formatHeaders 將 headers map 格式化為 OTEL_EXPORTER_OTLP_HEADERS 值
// （comma-separated key=value）。輸出依鍵名排序以確保確定性。
func formatHeaders(headers map[string]string) string {
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+headers[k])
	}
	return strings.Join(parts, ",")
}

// BuildEnv 回傳環境切片（os.Environ() 形式的 "KEY=VALUE"
// 字串），適合指定給 exec.Cmd.Env。它以現行程序環境
// （os.Environ()）為起點，並覆寫下列 Copilot BYOK 變數：
//
//	COPILOT_PROVIDER_BASE_URL = profile.APIBase
//	COPILOT_PROVIDER_TYPE     = profile.Provider（空字串時回退為 "openai"）
//	COPILOT_PROVIDER_API_KEY  = profile.APIKey
//	COPILOT_MODEL             = model（呼叫端已解析的單一模型字串）
//
// token 限制映射（limits 欄位 nil 時不注入對應變數，不做算術）：
//
//	ContextWindowTokens → COPILOT_PROVIDER_MAX_PROMPT_TOKENS
//	MaxOutputTokens     → COPILOT_PROVIDER_MAX_OUTPUT_TOKENS
//
// token 環境變數在建置子程序環境時去重：父環境的同名鍵先被移除，
// 僅在有效值存在時以 byok 值回填。父程序環境保持不變。
//
// 當 telemetry 非 nil 且 enabled 且 HTTP endpoint 存在時，額外注入
// OTEL 相關環境變數。
func BuildEnv(profile *config.Profile, model string, limits *TokenLimits, telemetry *config.Telemetry) []string {
	env := make([]string, 0, len(os.Environ())+4)

	// 複製現有環境，略過既存的 BYOK 鍵，使下方的覆寫成為
	// 這些鍵的唯一資料來源。
	for _, entry := range os.Environ() {
		key := entry
		if i := strings.IndexByte(entry, '='); i >= 0 {
			key = entry[:i]
		}
		if _, isByok := byokKeys[key]; isByok {
			continue
		}
		env = append(env, entry)
	}

	// 附上覆寫後的 BYOK 項目。
	provider := profile.Provider
	if provider == "" {
		provider = "openai"
	}

	env = append(env,
		"COPILOT_PROVIDER_BASE_URL="+profile.APIBase,
		"COPILOT_PROVIDER_TYPE="+provider,
		"COPILOT_PROVIDER_API_KEY="+profile.APIKey,
		"COPILOT_MODEL="+model,
	)
	// Token mapping：直接映射官方欄位，不做 context-output 算術；
	// unset 欄位不注入（不再使用固定 1048576/131072 假設值）。
	if limits != nil {
		if limits.ContextWindowTokens != nil {
			env = append(env, "COPILOT_PROVIDER_MAX_PROMPT_TOKENS="+strconv.FormatInt(*limits.ContextWindowTokens, 10))
		}
		if limits.MaxOutputTokens != nil {
			env = append(env, "COPILOT_PROVIDER_MAX_OUTPUT_TOKENS="+strconv.FormatInt(*limits.MaxOutputTokens, 10))
		}
	}

	// Telemetry injection: Copilot 僅支援 HTTP。
	if telemetry != nil && telemetry.Enabled && telemetry.HTTP != nil && telemetry.HTTP.Endpoint != "" {
		protocol := telemetry.HTTP.Protocol
		if protocol == "" {
			protocol = "protobuf"
		}
		env = append(env,
			"COPILOT_OTEL_ENABLED=true",
			"OTEL_EXPORTER_OTLP_ENDPOINT="+telemetry.HTTP.Endpoint,
			"OTEL_EXPORTER_OTLP_PROTOCOL=http/"+protocol,
		)
		if len(telemetry.Headers) > 0 {
			env = append(env, "OTEL_EXPORTER_OTLP_HEADERS="+formatHeaders(telemetry.Headers))
		}
		if sn := config.ComposeServiceName(telemetry.ServiceName, "copilot"); sn != "" {
			env = append(env, "OTEL_SERVICE_NAME="+sn)
		}
	}

	return env
}

// Launch 以 profile（及可選的 modelOverride）建置的 BYOK 環境變數，
// 將 exePath 指向的可執行檔啟動為子程序。limits 為呼叫端解析的有效 token
// 限制（nil 表示無）。stdin、stdout 與 stderr 透明
// 連接，讓使用者如常與子程序互動。父程序環境永不被修改 — 僅子程序
// 接收覆寫後的變數。
//
// exePath 必須為絕對路徑或可於 PATH 中解析的名稱；呼叫端通常在
// 呼叫 Launch 前先以 exec.LookPath 解析之。
//
// extraArgs 會原樣附加為子程序的命令列參數；傳入 nil 或空切片
// 時不附加任何參數（與舊版行為一致）。
func buildCopilotArgs(effort string, extraArgs []string) []string {
	args := make([]string, 0, len(extraArgs)+2)
	if effort != "" {
		args = append(args, "--reasoning-effort", effort)
	}
	return append(args, extraArgs...)
}

func Launch(profile *config.Profile, model string, limits *TokenLimits, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer, telemetry *config.Telemetry, effort ...string) error {
	e := ""
	if len(effort) > 0 {
		e = effort[0]
	}
	cmd := exec.Command(exePath, buildCopilotArgs(e, extraArgs)...)
	cmd.Env = BuildEnv(profile, model, limits, telemetry)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
