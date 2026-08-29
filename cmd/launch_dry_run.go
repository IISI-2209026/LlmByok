package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

func resolveProfileMetadata(cfgPath, profileName string, stderr io.Writer) (*config.Profile, *config.Telemetry, error) {
	path, err := configPath(cfgPath)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(stderr, "錯誤：讀取設定檔 %q 失敗: %v\n", path, err)
		return nil, nil, errExit
	}
	selected := profileName
	if selected == "" {
		selected = cfg.DefaultProfile
	}
	if selected == "" {
		fmt.Fprintf(stderr, "錯誤：未指定 profile 且 %q 中未設定 default_profile\n", path)
		return nil, nil, errExit
	}
	for i := range cfg.Profiles {
		if cfg.Profiles[i].Name == selected {
			p := &cfg.Profiles[i]
			provider := p.Provider
			if provider == "" {
				provider = "openai"
			}
			if provider != "openai" {
				fmt.Fprintf(stderr, "錯誤：profile %q 使用 provider %q；byok 首版僅支援 openai provider\n", p.Name, provider)
				return nil, nil, errExit
			}
			return p, cfg.Telemetry, nil
		}
	}
	fmt.Fprintf(stderr, "錯誤：在 %q 找不到 profile %q\n", path, selected)
	return nil, nil, errExit
}

func shellQuote(value string) string {
	if runtime.GOOS == "windows" {
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// dryRunTelemetryEnvs 回傳 dry-run 中 telemetry 相關的環境變數設定字串。
// isWindows 決定格式（PowerShell vs POSIX）。headers 值以 *** mask。
func dryRunTelemetryEnvs(target string, telemetry *config.Telemetry, isWindows bool) []string {
	if telemetry == nil || !telemetry.Enabled {
		return nil
	}
	set := func(name, value string) string {
		if isWindows {
			return "$env:" + name + "=" + shellQuote(value)
		}
		return name + "=" + shellQuote(value)
	}

	var envs []string
	switch target {
	case "copilot":
		if telemetry.HTTP == nil || telemetry.HTTP.Endpoint == "" {
			return nil
		}
		protocol := telemetry.HTTP.Protocol
		if protocol == "" {
			protocol = "protobuf"
		}
		envs = append(envs, set("COPILOT_OTEL_ENABLED", "true"))
		envs = append(envs, set("OTEL_EXPORTER_OTLP_ENDPOINT", telemetry.HTTP.Endpoint))
		envs = append(envs, set("OTEL_EXPORTER_OTLP_PROTOCOL", "http/"+protocol))
		if len(telemetry.Headers) > 0 {
			envs = append(envs, set("OTEL_EXPORTER_OTLP_HEADERS", maskHeaders(telemetry.Headers)))
		}
		if sn := config.ComposeServiceName(telemetry.ServiceName, "copilot"); sn != "" {
			envs = append(envs, set("OTEL_SERVICE_NAME", sn))
		}
	case "codex", "codex-app":
		if sn := config.ComposeServiceName(telemetry.ServiceName, "codex"); sn != "" {
			envs = append(envs, set("OTEL_SERVICE_NAME", sn))
		}
	case "claude":
		endpoint, protocol := "", ""
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
		if endpoint == "" {
			return nil
		}
		envs = append(envs, set("CLAUDE_CODE_ENABLE_TELEMETRY", "1"))
		envs = append(envs, set("OTEL_METRICS_EXPORTER", "otlp"))
		envs = append(envs, set("OTEL_LOGS_EXPORTER", "otlp"))
		envs = append(envs, set("OTEL_EXPORTER_OTLP_ENDPOINT", endpoint))
		envs = append(envs, set("OTEL_EXPORTER_OTLP_PROTOCOL", protocol))
		if len(telemetry.Headers) > 0 {
			envs = append(envs, set("OTEL_EXPORTER_OTLP_HEADERS", maskHeaders(telemetry.Headers)))
		}
		if sn := config.ComposeServiceName(telemetry.ServiceName, "claude"); sn != "" {
			envs = append(envs, set("OTEL_SERVICE_NAME", sn))
		}
	case "pi":
		if telemetry.HTTP == nil || telemetry.HTTP.Endpoint == "" {
			return nil
		}
		envs = append(envs, set("PI_OTEL_ENABLED", "true"))
		envs = append(envs, set("OTEL_EXPORTER_OTLP_ENDPOINT", telemetry.HTTP.Endpoint))
		if sn := config.ComposeServiceName(telemetry.ServiceName, "pi"); sn != "" {
			envs = append(envs, set("OTEL_SERVICE_NAME", sn))
		}
	}
	return envs
}

// maskHeaders 回傳 headers 的 masked 格式（key=***,key=***）。
func maskHeaders(headers map[string]string) string {
	parts := make([]string, 0, len(headers))
	for k := range headers {
		parts = append(parts, k+"=***")
	}
	return strings.Join(parts, ",")
}

// dryRunTelemetryCodexArgs 回傳 codex telemetry 的 --config 旗標（dry-run 用）。
// headers 值以 *** mask。
func dryRunTelemetryCodexArgs(telemetry *config.Telemetry, isWindows bool) []string {
	if telemetry == nil || !telemetry.Enabled {
		return nil
	}
	q := func(s string) string {
		if isWindows {
			return shellQuote(s)
		}
		return shellQuote(s)
	}
	var args []string
	if telemetry.GRPC != nil && telemetry.GRPC.Endpoint != "" {
		args = append(args, "--config", q(`otel.trace_exporter="otlp-grpc"`))
		args = append(args, "--config", q(`otel.trace_exporter.endpoint="`+telemetry.GRPC.Endpoint+`"`))
		args = append(args, "--config", q(`otel.exporter="otlp-grpc"`))
		args = append(args, "--config", q(`otel.exporter.endpoint="`+telemetry.GRPC.Endpoint+`"`))
		for k := range telemetry.Headers {
			args = append(args, "--config", q(`otel.trace_exporter.headers.`+k+`="***"`))
			args = append(args, "--config", q(`otel.exporter.headers.`+k+`="***"`))
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
		args = append(args, "--config", q(`otel.trace_exporter="otlp-http"`))
		args = append(args, "--config", q(`otel.trace_exporter.endpoint="`+telemetry.HTTP.Endpoint+`"`))
		args = append(args, "--config", q(`otel.trace_exporter.protocol="`+codexProto+`"`))
		args = append(args, "--config", q(`otel.exporter="otlp-http"`))
		args = append(args, "--config", q(`otel.exporter.endpoint="`+telemetry.HTTP.Endpoint+`"`))
		args = append(args, "--config", q(`otel.exporter.protocol="`+codexProto+`"`))
		for k := range telemetry.Headers {
			args = append(args, "--config", q(`otel.trace_exporter.headers.`+k+`="***"`))
			args = append(args, "--config", q(`otel.exporter.headers.`+k+`="***"`))
		}
	}
	return args
}

func renderLaunchDryRun(target string, profile *config.Profile, model string, opt launchOptions, extraArgs []string, telemetry *config.Telemetry, limits *resolvedTokenLimits) string {
	key := shellQuote("***")
	base := shellQuote(profile.APIBase)
	m := shellQuote(model)
	args := make([]string, 0, len(extraArgs))
	for _, arg := range extraArgs {
		args = append(args, shellQuote(arg))
	}
	join := func(values []string) string { return strings.Join(values, " ") }
	if runtime.GOOS == "windows" {
		set := func(name, value string) string { return "$env:" + name + "=" + shellQuote(value) }
		prefix := []string{}
		switch target {
		case "copilot":
			prefix = []string{set("COPILOT_PROVIDER_BASE_URL", profile.APIBase), set("COPILOT_PROVIDER_TYPE", "openai"), set("COPILOT_PROVIDER_API_KEY", "***"), set("COPILOT_MODEL", model)}
			prefix = append(prefix, dryRunTokenEnvs("copilot", limits, true)...)
			prefix = append(prefix, dryRunTelemetryEnvs("copilot", telemetry, true)...)
			if opt.effort != "" {
				prefix = append(prefix, "copilot", "--reasoning-effort", shellQuote(opt.effort))
			} else {
				prefix = append(prefix, "copilot")
			}
		case "codex", "codex-app":
			prefix = []string{set("BYOK_CODEX_API_KEY", "***")}
			prefix = append(prefix, dryRunTelemetryEnvs(target, telemetry, true)...)
			if target == "codex-app" {
				prefix = append(prefix, "codex", "app")
			} else {
				prefix = append(prefix, "codex")
			}
			prefix = append(prefix, "--config", shellQuote(`model="`+model+`"`), "--config", shellQuote(`model_provider="byok"`), "--config", shellQuote(`model_providers.byok.base_url="`+profile.APIBase+`"`), "--config", shellQuote(`model_providers.byok.env_key="BYOK_CODEX_API_KEY"`))
			if limits != nil && limits.ContextWindow != nil {
				prefix = append(prefix, "--config", shellQuote("model_context_window="+strconv.FormatInt(*limits.ContextWindow, 10)))
			}
			if opt.effort != "" {
				prefix = append(prefix, "--config", shellQuote(`model_reasoning_effort="`+opt.effort+`"`))
			}
			prefix = append(prefix, dryRunTelemetryCodexArgs(telemetry, true)...)
		case "claude":
			prefix = []string{set("ANTHROPIC_BASE_URL", profile.APIBase), set("ANTHROPIC_API_KEY", "***"), set("ANTHROPIC_MODEL", model)}
			prefix = append(prefix, dryRunTokenEnvs("claude", limits, true)...)
			if opt.effort != "" {
				prefix = append(prefix, set("CLAUDE_CODE_ALWAYS_ENABLE_EFFORT", "1"), set("CLAUDE_CODE_EFFORT_LEVEL", opt.effort))
			}
			if opt.subModel != "" {
				prefix = append(prefix, set("CLAUDE_CODE_SUBAGENT_MODEL", opt.subModel))
			}
			prefix = append(prefix, dryRunTelemetryEnvs("claude", telemetry, true)...)
			prefix = append(prefix, "claude")
		case "pi":
			piTelEnvs := dryRunTelemetryEnvs("pi", telemetry, true)
			piTelStr := ""
			if len(piTelEnvs) > 0 {
				piTelStr = "\n  " + strings.Join(piTelEnvs, "\n  ")
			}
			return "$tmp = Join-Path $env:TEMP ('byok-pi-' + [guid]::NewGuid().ToString())\nNew-Item -ItemType Directory -Path $tmp | Out-Null\ntry {\n  ('" + sqEmbed(piDryRunModelsJSON(profile.APIBase, model, limits), true) + "') | Set-Content (Join-Path $tmp 'models.json')" + piDryRunSettingsFragment(limits, true) + "\n  $env:PI_CODING_AGENT_DIR=$tmp" + piTelStr + "\n  pi --model " + m + func() string {
				if opt.effort != "" {
					return " --thinking " + shellQuote(opt.effort)
				}
				return ""
			}() + " " + join(args) + "\n} finally { Remove-Item -Recurse -Force $tmp }"
		}
		return join(append(prefix, args...))
	}
	prefix := []string{}
	switch target {
	case "copilot":
		prefix = []string{"COPILOT_PROVIDER_BASE_URL=" + base, "COPILOT_PROVIDER_TYPE='openai'", "COPILOT_PROVIDER_API_KEY=" + key, "COPILOT_MODEL=" + m}
		prefix = append(prefix, dryRunTokenEnvs("copilot", limits, false)...)
		prefix = append(prefix, dryRunTelemetryEnvs("copilot", telemetry, false)...)
		prefix = append(prefix, "copilot")
		if opt.effort != "" {
			prefix = append(prefix, "--reasoning-effort", shellQuote(opt.effort))
		}
	case "codex", "codex-app":
		prefix = []string{"BYOK_CODEX_API_KEY=" + key}
		prefix = append(prefix, dryRunTelemetryEnvs(target, telemetry, false)...)
		if target == "codex-app" {
			prefix = append(prefix, "codex", "app")
		} else {
			prefix = append(prefix, "codex")
		}
		prefix = append(prefix, "--config", shellQuote(`model="`+model+`"`), "--config", shellQuote(`model_provider="byok"`), "--config", shellQuote(`model_providers.byok.base_url="`+profile.APIBase+`"`), "--config", shellQuote(`model_providers.byok.env_key="BYOK_CODEX_API_KEY"`))
		if limits != nil && limits.ContextWindow != nil {
			prefix = append(prefix, "--config", shellQuote("model_context_window="+strconv.FormatInt(*limits.ContextWindow, 10)))
		}
		if opt.effort != "" {
			prefix = append(prefix, "--config", shellQuote(`model_reasoning_effort="`+opt.effort+`"`))
		}
		prefix = append(prefix, dryRunTelemetryCodexArgs(telemetry, false)...)
	case "claude":
		prefix = []string{"ANTHROPIC_BASE_URL=" + base, "ANTHROPIC_API_KEY=" + key, "ANTHROPIC_MODEL=" + m}
		prefix = append(prefix, dryRunTokenEnvs("claude", limits, false)...)
		if opt.effort != "" {
			prefix = append(prefix, "CLAUDE_CODE_ALWAYS_ENABLE_EFFORT='1'", "CLAUDE_CODE_EFFORT_LEVEL="+shellQuote(opt.effort))
		}
		if opt.subModel != "" {
			prefix = append(prefix, "CLAUDE_CODE_SUBAGENT_MODEL="+shellQuote(opt.subModel))
		}
		prefix = append(prefix, dryRunTelemetryEnvs("claude", telemetry, false)...)
		prefix = append(prefix, "claude")
	case "pi":
		piTelEnvs := dryRunTelemetryEnvs("pi", telemetry, false)
		piTelStr := ""
		if len(piTelEnvs) > 0 {
			piTelStr = " " + strings.Join(piTelEnvs, " ")
		}
		return "tmp=\"$(mktemp -d -t byok-pi-XXXXXX)\"\ntrap 'rm -rf \"$tmp\"' EXIT\nprintf '%s' '" + sqEmbed(piDryRunModelsJSON(profile.APIBase, model, limits), false) + "' > \"$tmp/models.json\"" + piDryRunSettingsFragment(limits, false) + "\nPI_CODING_AGENT_DIR=\"$tmp\"" + piTelStr + " pi --model " + m + func() string {
			if opt.effort != "" {
				return " --thinking " + shellQuote(opt.effort)
			}
			return ""
		}() + " " + join(args)
	}
	return join(append(prefix, args...))
}

// dryRunTokenEnvs 回傳 dry-run 中環境變數型 target（Copilot / Claude）的
// token 設定字串。isWindows 決定格式（PowerShell vs POSIX）。
// Copilot 使用官方 prompt/output 變數名；Claude 使用 CLAUDE_CODE_MAX_*。
// unset 欄位不輸出；Codex 的 context 以 --config 渲染（於 renderLaunchDryRun
// 分支內），不經此函式。
func dryRunTokenEnvs(target string, limits *resolvedTokenLimits, isWindows bool) []string {
	if limits == nil {
		return nil
	}
	ctxVar, outVar := "COPILOT_PROVIDER_MAX_PROMPT_TOKENS", "COPILOT_PROVIDER_MAX_OUTPUT_TOKENS"
	if target == "claude" {
		ctxVar, outVar = "CLAUDE_CODE_MAX_CONTEXT_TOKENS", "CLAUDE_CODE_MAX_OUTPUT_TOKENS"
	}
	set := func(name string, v int64) string {
		value := strconv.FormatInt(v, 10)
		if isWindows {
			return "$env:" + name + "=" + shellQuote(value)
		}
		return name + "=" + shellQuote(value)
	}
	var envs []string
	if limits.ContextWindow != nil {
		envs = append(envs, set(ctxVar, *limits.ContextWindow))
	}
	if limits.MaxOutput != nil {
		envs = append(envs, set(outVar, *limits.MaxOutput))
	}
	return envs
}

// piDryRunModelsJSON 組出 masked Pi models.json 的 JSON 內容字串（嵌入
// dry-run 指令；以 json.Marshal 保證輸出為有效 JSON 且模型名稱為精確鍵）。
// 有效 context/output 時輸出 modelOverrides.<model>.{contextWindow,maxTokens}；
// 皆未設定時維持 provider-only 內容（無 modelOverrides 鍵）。API key 一律 `"***"`。
func piDryRunModelsJSON(apiBase, model string, limits *resolvedTokenLimits) string {
	provider := map[string]any{"baseUrl": apiBase, "apiKey": "***"}
	if limits != nil && (limits.ContextWindow != nil || limits.MaxOutput != nil) {
		override := map[string]any{}
		if limits.ContextWindow != nil {
			override["contextWindow"] = *limits.ContextWindow
		}
		if limits.MaxOutput != nil {
			override["maxTokens"] = *limits.MaxOutput
		}
		provider["modelOverrides"] = map[string]any{model: override}
	}
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(map[string]any{"providers": map[string]any{"openai": provider}})
	return strings.TrimRight(b.String(), "\n")
}

// piDryRunSettingsJSON 回傳 masked 暫存 settings.json 的 JSON（有效
// max_output_tokens 時 reserveTokens 等於該值）。
func piDryRunSettingsJSON(limits *resolvedTokenLimits) string {
	if limits == nil || limits.MaxOutput == nil {
		return ""
	}
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(map[string]any{"compaction": map[string]any{"reserveTokens": *limits.MaxOutput}})
	return strings.TrimRight(b.String(), "\n")
}

// piWindowSettingsFragment returns the Windows Set-Content fragment for the
// temporary settings.json, or "" when unset.
// piDryRunSettingsFragment 回傳 platform-native 暫存 settings.json 寫入片段
// （有效 max_output_tokens 時 reserveTokens 等於該值）；unset 時為空字串。
func piDryRunSettingsFragment(limits *resolvedTokenLimits, isWindows bool) string {
	settings := piDryRunSettingsJSON(limits)
	if settings == "" {
		return ""
	}
	if isWindows {
		return "\n  ('" + sqEmbed(settings, true) + "') | Set-Content (Join-Path $tmp 'settings.json')"
	}
	return "\nprintf '%s' '" + sqEmbed(settings, false) + "' > \"$tmp/settings.json\""
}

// sqEmbed 將字串安全嵌入 single-quoted（POSIX）或 PowerShell 單引號字串。
func sqEmbed(s string, isWindows bool) string {
	if isWindows {
		return strings.ReplaceAll(s, "'", "''")
	}
	return strings.ReplaceAll(s, "'", `'\''`)
}

func runLaunchDryRun(cfgPath, profileName, target, model string, opt launchOptions, extraArgs []string, stdout, stderr io.Writer) error {
	profile, telemetry, err := resolveProfileMetadata(cfgPath, profileName, stderr)
	if err != nil {
		return err
	}
	resolvedModel, err := resolveModelForLaunch(profile, model, os.Stdin, stdout, stderr)
	if err != nil {
		return err
	}
	// normal launch 與 dry-run 共用同一份有效 token 限制；
	// 不支援的參數 warning 只寫 stderr，stdout 保持可執行命令。
	limits := resolveTokenLimits(profile, resolvedModel, opt.cliContextTokens, opt.cliMaxOutputTokens)
	warnUnsupportedTokenLimits(target, limits, stderr)
	fmt.Fprintln(stdout, renderLaunchDryRun(target, profile, resolvedModel, opt, extraArgs, telemetry, limits))
	return nil
}
