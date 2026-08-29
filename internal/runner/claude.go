package runner

import (
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// claudeByokKeys 是 BuildClaudeEnv 會從現行程序環境中覆寫的
// Claude Code BYOK 環境變數鍵名。token 兩鍵亦由 byok 管理：父環境的值
// 會被移除，僅在有效值存在時以 byok 值重填（環境去重）。
var claudeByokKeys = map[string]struct{}{
	"ANTHROPIC_BASE_URL":               {},
	"ANTHROPIC_API_KEY":                {},
	"ANTHROPIC_MODEL":                  {},
	"CLAUDE_CODE_ALWAYS_ENABLE_EFFORT": {},
	"CLAUDE_CODE_EFFORT_LEVEL":         {},
	"CLAUDE_CODE_SUBAGENT_MODEL":       {},
	"CLAUDE_CODE_MAX_CONTEXT_TOKENS":   {},
	"CLAUDE_CODE_MAX_OUTPUT_TOKENS":    {},
}

// BuildClaudeEnv 回傳環境切片（os.Environ() 形式的 "KEY=VALUE"
// 字串），適合指定給 exec.Cmd.Env。它以現行程序環境（os.Environ()）
// 為起點，並覆寫下列 Claude Code BYOK 變數：
//
//	ANTHROPIC_BASE_URL = profile.APIBase
//	ANTHROPIC_API_KEY  = profile.APIKey
//	ANTHROPIC_MODEL    = model（原樣傳遞，不自動附加 [1m] 後綴）
//
// limits 為呼叫端解析的有效 token 限制（nil 指標欄位表示未設定），
// Claude model token limit mapping 直接映射：
//
//	ContextWindowTokens → CLAUDE_CODE_MAX_CONTEXT_TOKENS
//	MaxOutputTokens     → CLAUDE_CODE_MAX_OUTPUT_TOKENS
//
// token 環境變數在建置子程序環境時去重：父環境的同名鍵先被移除，
// 僅在有效值存在時以 byok 值回填。父程序環境保持不變。
func BuildClaudeEnv(profile *config.Profile, model string, limits *TokenLimits, telemetry *config.Telemetry, options ...string) []string {
	effort, subModel := "", ""
	if len(options) > 0 {
		effort = options[0]
	}
	if len(options) > 1 {
		subModel = options[1]
	}
	env := make([]string, 0, len(os.Environ())+6)

	for _, entry := range os.Environ() {
		key := entry
		if i := strings.IndexByte(entry, '='); i >= 0 {
			key = entry[:i]
		}
		if _, isByok := claudeByokKeys[key]; isByok {
			continue
		}
		env = append(env, entry)
	}

	// Claude Code 透過 ANTHROPIC_MODEL 直接指定模型，模型字串原樣
	// 傳遞（不自動附加 [1m]；呼叫端如需 1M context 可自行帶後綴）。
	env = append(env,
		"ANTHROPIC_BASE_URL="+profile.APIBase,
		"ANTHROPIC_API_KEY="+profile.APIKey,
		"ANTHROPIC_MODEL="+model,
	)
	// Token mapping：直接映射官方欄位，不做算術；unset 欄位不注入。
	if limits != nil {
		if limits.ContextWindowTokens != nil {
			env = append(env, "CLAUDE_CODE_MAX_CONTEXT_TOKENS="+strconv.FormatInt(*limits.ContextWindowTokens, 10))
		}
		if limits.MaxOutputTokens != nil {
			env = append(env, "CLAUDE_CODE_MAX_OUTPUT_TOKENS="+strconv.FormatInt(*limits.MaxOutputTokens, 10))
		}
	}
	if effort != "" {
		env = append(env, "CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1", "CLAUDE_CODE_EFFORT_LEVEL="+effort)
	}
	if subModel != "" {
		env = append(env, "CLAUDE_CODE_SUBAGENT_MODEL="+subModel)
	}

	// Telemetry injection: Claude 支援 gRPC（優先）與 HTTP。
	if telemetry != nil && telemetry.Enabled {
		endpoint := ""
		protocol := ""
		if telemetry.GRPC != nil && telemetry.GRPC.Endpoint != "" {
			endpoint = telemetry.GRPC.Endpoint
			protocol = "grpc"
		} else if telemetry.HTTP != nil && telemetry.HTTP.Endpoint != "" {
			endpoint = telemetry.HTTP.Endpoint
			p := telemetry.HTTP.Protocol
			if p == "" {
				p = "protobuf"
			}
			protocol = "http/" + p
		}
		if endpoint != "" {
			env = append(env,
				"CLAUDE_CODE_ENABLE_TELEMETRY=1",
				"OTEL_METRICS_EXPORTER=otlp",
				"OTEL_LOGS_EXPORTER=otlp",
				"OTEL_EXPORTER_OTLP_ENDPOINT="+endpoint,
				"OTEL_EXPORTER_OTLP_PROTOCOL="+protocol,
			)
			if len(telemetry.Headers) > 0 {
				env = append(env, "OTEL_EXPORTER_OTLP_HEADERS="+formatHeaders(telemetry.Headers))
			}
			if sn := config.ComposeServiceName(telemetry.ServiceName, "claude"); sn != "" {
				env = append(env, "OTEL_SERVICE_NAME="+sn)
			}
		}
	}

	return env
}

// LaunchClaude 以 BuildClaudeEnv 組裝的環境啟動 exePath 指向的
// claude 可執行檔為子程序。extraArgs 會原樣附加為子程序的命令列
// 參數。stdin、stdout 與 stderr 透明連接。父程序環境永不被修改 —
// 僅子程序接收覆寫後的變數。
func LaunchClaude(profile *config.Profile, model string, limits *TokenLimits, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer, telemetry *config.Telemetry, options ...string) error {
	cmd := exec.Command(exePath, extraArgs...)
	cmd.Env = BuildClaudeEnv(profile, model, limits, telemetry, options...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
