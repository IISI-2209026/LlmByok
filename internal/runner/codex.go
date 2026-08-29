package runner

import (
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// codexAPIKeyEnv 為承載 Codex BYOK API key 的內部環境變數名稱。
// 僅設定於 codex 子程序環境，父程序與 shell 不受影響。
const codexAPIKeyEnv = "BYOK_CODEX_API_KEY"

// codexProviderID 為 byok 用於 codex --config 覆寫的自訂 provider id。
// 採用不易衝突的固定 id，避開 codex 保留字（openai/ollama/lmstudio）。
const codexProviderID = "byok"

// BuildCodexArgs 以 profile 與可選的 modelOverride 建置 codex 子程序
// 所需的環境切片與 --config 旗標切片。
//
// env 以現行程序環境（os.Environ()）為起點，過濾掉既存的
// BYOK_CODEX_API_KEY 後附加 BYOK_CODEX_API_KEY=<profile.APIKey>；
// 其餘現有環境變數保持不變。
//
// configArgs 為成對的 ["--config", "<key>=<value>", ...] 切片，覆寫：
//
//	model                                  = model（呼叫端已解析的單一模型字串）
//	model_provider                         = "byok"
//	model_providers.byok.name              = "BYOK"
//	model_providers.byok.base_url          = profile.APIBase
//	model_providers.byok.env_key           = "BYOK_CODEX_API_KEY"
//
// TOML 字串值以雙引號包裹（不經過 shell，故不需外層 shell quoting）。
// 模型解析（候選清單選擇）由呼叫端（cmd/launch 層）完成。
//
// limits 為呼叫端解析的有效 token 限制（nil 表示無）：
//
//	ContextWindowTokens → --config model_context_window=<value>
//	  （未加引號的整數，位於五個 provider 覆寫之後、effort 之前）
//	MaxOutputTokens     → 不支援，由呼叫端的共用 warning contract 提示、
//	  本函式一律忽略，絕不注入任何 output override。
func BuildCodexArgs(profile *config.Profile, model string, limits *TokenLimits, telemetry *config.Telemetry, effort ...string) (env []string, configArgs []string) {
	env = make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		key := entry
		if i := strings.IndexByte(entry, '='); i >= 0 {
			key = entry[:i]
		}
		if key == codexAPIKeyEnv {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, codexAPIKeyEnv+"="+profile.APIKey)

	configArgs = []string{
		"--config", `model="` + model + `"`,
		"--config", `model_provider="` + codexProviderID + `"`,
		"--config", `model_providers.` + codexProviderID + `.name="BYOK"`,
		"--config", `model_providers.` + codexProviderID + `.base_url="` + profile.APIBase + `"`,
		"--config", `model_providers.` + codexProviderID + `.env_key="` + codexAPIKeyEnv + `"`,
	}
	// Codex token limit mapping：effective context 以未加引號的整數注入
	// 頂層 model_context_window；MaxOutputTokens 不支援 → 一律忽略
	// （提示由呼叫端的共用 warning contract 負責）。
	if limits != nil && limits.ContextWindowTokens != nil {
		configArgs = append(configArgs,
			"--config", "model_context_window="+strconv.FormatInt(*limits.ContextWindowTokens, 10))
	}
	if len(effort) > 0 && effort[0] != "" {
		configArgs = append(configArgs, "--config", `model_reasoning_effort="`+effort[0]+`"`)
	}

	// Telemetry injection: Codex 支援 gRPC（優先）與 HTTP。
	if telemetry != nil && telemetry.Enabled {
		if telemetry.GRPC != nil && telemetry.GRPC.Endpoint != "" {
			configArgs = append(configArgs,
				"--config", `otel.trace_exporter="otlp-grpc"`,
				"--config", `otel.trace_exporter.endpoint="`+telemetry.GRPC.Endpoint+`"`,
				"--config", `otel.exporter="otlp-grpc"`,
				"--config", `otel.exporter.endpoint="`+telemetry.GRPC.Endpoint+`"`,
			)
			for k, v := range telemetry.Headers {
				configArgs = append(configArgs,
					"--config", `otel.trace_exporter.headers.`+k+`="`+v+`"`,
					"--config", `otel.exporter.headers.`+k+`="`+v+`"`,
				)
			}
			if sn := config.ComposeServiceName(telemetry.ServiceName, "codex"); sn != "" {
				env = append(env, "OTEL_SERVICE_NAME="+sn)
			}
		} else if telemetry.HTTP != nil && telemetry.HTTP.Endpoint != "" {
			protocol := telemetry.HTTP.Protocol
			if protocol == "" {
				protocol = "protobuf"
			}
			codexProto := "binary"
			if protocol == "json" {
				codexProto = "json"
			}
			configArgs = append(configArgs,
				"--config", `otel.trace_exporter="otlp-http"`,
				"--config", `otel.trace_exporter.endpoint="`+telemetry.HTTP.Endpoint+`"`,
				"--config", `otel.trace_exporter.protocol="`+codexProto+`"`,
				"--config", `otel.exporter="otlp-http"`,
				"--config", `otel.exporter.endpoint="`+telemetry.HTTP.Endpoint+`"`,
				"--config", `otel.exporter.protocol="`+codexProto+`"`,
			)
			for k, v := range telemetry.Headers {
				configArgs = append(configArgs,
					"--config", `otel.trace_exporter.headers.`+k+`="`+v+`"`,
					"--config", `otel.exporter.headers.`+k+`="`+v+`"`,
				)
			}
			if sn := config.ComposeServiceName(telemetry.ServiceName, "codex"); sn != "" {
				env = append(env, "OTEL_SERVICE_NAME="+sn)
			}
		}
	}

	return env, configArgs
}

// LaunchCodex 以 BuildCodexArgs 組裝的環境與 --config 旗標啟動 codex
// 可執行檔為子程序。extraArgs 會插入於 --config 旗標之後、原樣附加為
// 子程序的命令列參數（呼叫端負責安排 --yolo 與透傳順序）。stdin、stdout
// 與 stderr 透明連接。父程序環境永不被修改 — 僅子程序接收覆寫後的變數。
//
// 命令列順序：codex [<--config ...>] [<extraArgs...>]。
func LaunchCodex(profile *config.Profile, model string, limits *TokenLimits, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer, telemetry *config.Telemetry, effort ...string) error {
	env, configArgs := BuildCodexArgs(profile, model, limits, telemetry, effort...)
	args := append([]string(nil), configArgs...)
	args = append(args, extraArgs...)

	cmd := exec.Command(exePath, args...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// LaunchCodexApp 以 BuildCodexArgs 組裝的環境與 --config 旗標啟動
// codex app 子程序（Codex 桌面版）。app 子命令插入為命令列第一個參數，
// 接著是 --config 旗標，最後是透傳參數。stdin、stdout 與 stderr 透明
// 連接。父程序環境永不被修改 — 僅子程序接收覆寫後的變數。
//
// 命令列順序：codex app [--config ...] [<extraArgs...>]。
func LaunchCodexApp(profile *config.Profile, model string, limits *TokenLimits, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer, telemetry *config.Telemetry, effort ...string) error {
	env, configArgs := BuildCodexArgs(profile, model, limits, telemetry, effort...)
	args := []string{"app"}
	args = append(args, configArgs...)
	args = append(args, extraArgs...)

	cmd := exec.Command(exePath, args...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
