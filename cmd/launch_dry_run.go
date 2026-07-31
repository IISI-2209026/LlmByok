package cmd

import (
	"fmt"
	"io"
	"os"
	"runtime"
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

func renderLaunchDryRun(target string, profile *config.Profile, model string, opt launchOptions, extraArgs []string, telemetry *config.Telemetry) string {
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
			if opt.effort != "" {
				prefix = append(prefix, "--config", shellQuote(`model_reasoning_effort="`+opt.effort+`"`))
			}
			prefix = append(prefix, dryRunTelemetryCodexArgs(telemetry, true)...)
		case "claude":
			prefix = []string{set("ANTHROPIC_BASE_URL", profile.APIBase), set("ANTHROPIC_API_KEY", "***"), set("ANTHROPIC_MODEL", model)}
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
			return "$tmp = Join-Path $env:TEMP ('byok-pi-' + [guid]::NewGuid().ToString())\nNew-Item -ItemType Directory -Path $tmp | Out-Null\ntry {\n  ('{\"providers\":{\"openai\":{\"baseUrl\":' + " + base + " + ',\"apiKey\":' + " + key + " + '}}}') | Set-Content (Join-Path $tmp 'models.json')\n  $env:PI_CODING_AGENT_DIR=$tmp" + piTelStr + "\n  pi --model " + m + func() string {
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
		if opt.effort != "" {
			prefix = append(prefix, "--config", shellQuote(`model_reasoning_effort="`+opt.effort+`"`))
		}
		prefix = append(prefix, dryRunTelemetryCodexArgs(telemetry, false)...)
	case "claude":
		prefix = []string{"ANTHROPIC_BASE_URL=" + base, "ANTHROPIC_API_KEY=" + key, "ANTHROPIC_MODEL=" + m}
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
		return "tmp=\"$(mktemp -d -t byok-pi-XXXXXX)\"\ntrap 'rm -rf \"$tmp\"' EXIT\nprintf '%s' '{\"providers\":{\"openai\":{\"baseUrl\":" + base + ",\"apiKey\":" + key + "}}}' > \"$tmp/models.json\"\nPI_CODING_AGENT_DIR=\"$tmp\"" + piTelStr + " pi --model " + m + func() string {
			if opt.effort != "" {
				return " --thinking " + shellQuote(opt.effort)
			}
			return ""
		}() + " " + join(args)
	}
	return join(append(prefix, args...))
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
	fmt.Fprintln(stdout, renderLaunchDryRun(target, profile, resolvedModel, opt, extraArgs, telemetry))
	return nil
}
