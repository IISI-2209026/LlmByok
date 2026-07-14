package cmd

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

func resolveProfileMetadata(cfgPath, profileName string, stderr io.Writer) (*config.Profile, error) {
	path, err := configPath(cfgPath)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(stderr, "錯誤：讀取設定檔 %q 失敗: %v\n", path, err)
		return nil, errExit
	}
	selected := profileName
	if selected == "" {
		selected = cfg.DefaultProfile
	}
	if selected == "" {
		fmt.Fprintf(stderr, "錯誤：未指定 profile 且 %q 中未設定 default_profile\n", path)
		return nil, errExit
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
				return nil, errExit
			}
			return p, nil
		}
	}
	fmt.Fprintf(stderr, "錯誤：在 %q 找不到 profile %q\n", path, selected)
	return nil, errExit
}

func shellQuote(value string) string {
	if runtime.GOOS == "windows" {
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func renderLaunchDryRun(target string, profile *config.Profile, model string, opt launchOptions, extraArgs []string) string {
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
			if opt.effort != "" {
				prefix = append(prefix, "copilot", "--reasoning-effort", shellQuote(opt.effort))
			} else {
				prefix = append(prefix, "copilot")
			}
		case "codex", "codex-app":
			prefix = []string{set("BYOK_CODEX_API_KEY", "***")}
			if target == "codex-app" {
				prefix = append(prefix, "codex", "app")
			} else {
				prefix = append(prefix, "codex")
			}
			prefix = append(prefix, "--config", shellQuote(`model="`+model+`"`), "--config", shellQuote(`model_provider="byok"`), "--config", shellQuote(`model_providers.byok.base_url="`+profile.APIBase+`"`), "--config", shellQuote(`model_providers.byok.env_key="BYOK_CODEX_API_KEY"`))
			if opt.effort != "" {
				prefix = append(prefix, "--config", shellQuote(`model_reasoning_effort="`+opt.effort+`"`))
			}
		case "claude":
			prefix = []string{set("ANTHROPIC_BASE_URL", profile.APIBase), set("ANTHROPIC_API_KEY", "***"), set("ANTHROPIC_MODEL", model)}
			if opt.effort != "" {
				prefix = append(prefix, set("CLAUDE_CODE_ALWAYS_ENABLE_EFFORT", "1"), set("CLAUDE_CODE_EFFORT_LEVEL", opt.effort))
			}
			if opt.subModel != "" {
				prefix = append(prefix, set("CLAUDE_CODE_SUBAGENT_MODEL", opt.subModel))
			}
			prefix = append(prefix, "claude")
		case "pi":
			return "$tmp = Join-Path $env:TEMP ('byok-pi-' + [guid]::NewGuid().ToString())\nNew-Item -ItemType Directory -Path $tmp | Out-Null\ntry {\n  ('{\"providers\":{\"openai\":{\"baseUrl\":' + " + base + " + ',\"apiKey\":' + " + key + " + '}}}') | Set-Content (Join-Path $tmp 'models.json')\n  $env:PI_CODING_AGENT_DIR=$tmp\n  pi --model " + m + func() string {
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
		prefix = []string{"COPILOT_PROVIDER_BASE_URL=" + base, "COPILOT_PROVIDER_TYPE='openai'", "COPILOT_PROVIDER_API_KEY=" + key, "COPILOT_MODEL=" + m, "copilot"}
		if opt.effort != "" {
			prefix = append(prefix, "--reasoning-effort", shellQuote(opt.effort))
		}
	case "codex", "codex-app":
		prefix = []string{"BYOK_CODEX_API_KEY=" + key}
		if target == "codex-app" {
			prefix = append(prefix, "codex", "app")
		} else {
			prefix = append(prefix, "codex")
		}
		prefix = append(prefix, "--config", shellQuote(`model="`+model+`"`), "--config", shellQuote(`model_provider="byok"`), "--config", shellQuote(`model_providers.byok.base_url="`+profile.APIBase+`"`), "--config", shellQuote(`model_providers.byok.env_key="BYOK_CODEX_API_KEY"`))
		if opt.effort != "" {
			prefix = append(prefix, "--config", shellQuote(`model_reasoning_effort="`+opt.effort+`"`))
		}
	case "claude":
		prefix = []string{"ANTHROPIC_BASE_URL=" + base, "ANTHROPIC_API_KEY=" + key, "ANTHROPIC_MODEL=" + m}
		if opt.effort != "" {
			prefix = append(prefix, "CLAUDE_CODE_ALWAYS_ENABLE_EFFORT='1'", "CLAUDE_CODE_EFFORT_LEVEL="+shellQuote(opt.effort))
		}
		if opt.subModel != "" {
			prefix = append(prefix, "CLAUDE_CODE_SUBAGENT_MODEL="+shellQuote(opt.subModel))
		}
		prefix = append(prefix, "claude")
	case "pi":
		return "tmp=\"$(mktemp -d -t byok-pi-XXXXXX)\"\ntrap 'rm -rf \"$tmp\"' EXIT\nprintf '%s' '{\"providers\":{\"openai\":{\"baseUrl\":" + base + ",\"apiKey\":" + key + "}}}' > \"$tmp/models.json\"\nPI_CODING_AGENT_DIR=\"$tmp\" pi --model " + m + func() string {
			if opt.effort != "" {
				return " --thinking " + shellQuote(opt.effort)
			}
			return ""
		}() + " " + join(args)
	}
	return join(append(prefix, args...))
}

func runLaunchDryRun(cfgPath, profileName, target, model string, opt launchOptions, extraArgs []string, stdout, stderr io.Writer) error {
	profile, err := resolveProfileMetadata(cfgPath, profileName, stderr)
	if err != nil {
		return err
	}
	resolvedModel, err := resolveModelForLaunch(profile, model, os.Stdin, stdout, stderr)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, renderLaunchDryRun(target, profile, resolvedModel, opt, extraArgs))
	return nil
}
