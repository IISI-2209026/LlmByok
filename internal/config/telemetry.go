package config

import "fmt"

// TelemetryGRPC 承載 gRPC OTLP endpoint 設定。
type TelemetryGRPC struct {
	Endpoint string `yaml:"endpoint"`
}

// TelemetryHTTP 承載 HTTP OTLP endpoint 設定。
type TelemetryHTTP struct {
	Endpoint string `yaml:"endpoint"`
	Protocol string `yaml:"protocol"` // "protobuf" | "json"
}

// Telemetry 承載頂層 telemetry 設定區段。
type Telemetry struct {
	Enabled     bool              `yaml:"enabled"`
	ServiceName string            `yaml:"service_name,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty"`
	GRPC        *TelemetryGRPC    `yaml:"grpc,omitempty"`
	HTTP        *TelemetryHTTP    `yaml:"http,omitempty"`
}

// Validate 驗證 Telemetry 設定。nil receiver 視為合法（未設定 telemetry）。
func (t *Telemetry) Validate() error {
	if t == nil {
		return nil
	}
	if t.HTTP != nil && t.HTTP.Protocol != "" {
		if t.HTTP.Protocol != "protobuf" && t.HTTP.Protocol != "json" {
			return fmt.Errorf("telemetry.http.protocol 值 %q 無效，僅接受 \"protobuf\" 或 \"json\"", t.HTTP.Protocol)
		}
	}
	return nil
}

// ComposeServiceName 依據 service_name 與 target 組合 OTEL service name。
// 空 serviceName 回傳空字串。target 接受 copilot、codex、codex-app、claude、pi。
func ComposeServiceName(serviceName, target string) string {
	if serviceName == "" {
		return ""
	}
	suffix := map[string]string{
		"copilot":   "github-copilot",
		"codex":     "codex-cli",
		"codex-app": "codex-cli",
		"claude":    "claude-code",
		"pi":        "pi-coding-agent",
	}
	s, ok := suffix[target]
	if !ok {
		return serviceName
	}
	return serviceName + "-" + s
}
