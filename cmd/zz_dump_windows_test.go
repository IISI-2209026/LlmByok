package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// TestZZDumpWindowsDryRun 僅供人工驗證 Windows 輸出（臨時測試，驗證後刪除）。
func TestZZDumpWindowsDryRun(t *testing.T) {
	dryRunIsWindows = true
	defer func() { dryRunIsWindows = false }()

	p := &config.Profile{APIBase: "https://example.test/v1", APIKey: "sk-real", Provider: "openai", Models: []config.Model{{Name: "gpt-5"}}}
	limits := dryRunLimits(ptrToken(1000000), ptrToken(128000))
	tel := &config.Telemetry{
		Enabled:     true,
		ServiceName: "byok",
		Headers:     map[string]string{"authorization": "Bearer tok"},
		HTTP:        &config.TelemetryHTTP{Endpoint: "https://otel.test/v1/metrics", Protocol: "protobuf"},
		GRPC:        &config.TelemetryGRPC{Endpoint: "https://otel.test:4317"},
	}

	out := os.Getenv("BYOK_DUMP_DIR")
	if out == "" {
		out = "/tmp/byok-dryrun"
	}
	_ = os.MkdirAll(out, 0o755)

	scenarios := []struct {
		name   string
		target string
		opt    launchOptions
		limits *resolvedTokenLimits
		tel    *config.Telemetry
	}{
		{"copilot_full", "copilot", launchOptions{effort: "high"}, limits, tel},
		{"copilot_plain", "copilot", launchOptions{}, nil, nil},
		{"codex_full", "codex", launchOptions{effort: "high"}, limits, tel},
		{"codex_plain", "codex", launchOptions{}, nil, nil},
		{"codexapp_full", "codex-app", launchOptions{effort: "high"}, limits, tel},
		{"claude_full", "claude", launchOptions{effort: "high", subModel: "claude-haiku-4-5"}, limits, tel},
		{"claude_plain", "claude", launchOptions{}, nil, nil},
		{"pi_full", "pi", launchOptions{effort: "high"}, limits, tel},
		{"pi_plain", "pi", launchOptions{}, nil, nil},
	}
	for _, s := range scenarios {
		got := renderLaunchDryRun(s.target, p, "gpt-5", s.opt, []string{"--yolo"}, s.tel, s.limits)
		path := out + "/" + s.name + ".ps1"
		if err := os.WriteFile(path, []byte(got+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		fmt.Printf("=== %s ===\n%s\n", s.name, got)
	}
}