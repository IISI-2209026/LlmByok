package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTelemetry_Validate_ValidConfig(t *testing.T) {
	tel := &Telemetry{
		Enabled:     true,
		ServiceName: "my-team",
		Headers:     map[string]string{"Authorization": "Bearer token"},
		GRPC:        &TelemetryGRPC{Endpoint: "http://localhost:4317"},
		HTTP:        &TelemetryHTTP{Endpoint: "http://localhost:4318", Protocol: "protobuf"},
	}
	if err := tel.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestTelemetry_Validate_InvalidProtocol(t *testing.T) {
	tel := &Telemetry{
		Enabled: true,
		HTTP:    &TelemetryHTTP{Endpoint: "http://localhost:4318", Protocol: "grpc"},
	}
	err := tel.Validate()
	if err == nil {
		t.Fatal("expected error for invalid protocol")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestTelemetry_Validate_NilTelemetry(t *testing.T) {
	var tel *Telemetry
	if err := tel.Validate(); err != nil {
		t.Errorf("expected nil telemetry to be valid, got %v", err)
	}
}

func TestTelemetry_Validate_EmptyProtocolIsValid(t *testing.T) {
	tel := &Telemetry{
		Enabled: true,
		HTTP:    &TelemetryHTTP{Endpoint: "http://localhost:4318", Protocol: ""},
	}
	if err := tel.Validate(); err != nil {
		t.Errorf("empty protocol should be valid (defaults to protobuf), got %v", err)
	}
}

func TestTelemetry_Validate_JsonProtocol(t *testing.T) {
	tel := &Telemetry{
		Enabled: true,
		HTTP:    &TelemetryHTTP{Endpoint: "http://localhost:4318", Protocol: "json"},
	}
	if err := tel.Validate(); err != nil {
		t.Errorf("json protocol should be valid, got %v", err)
	}
}

func TestLoad_WithTelemetry(t *testing.T) {
	src := `profiles:
  - name: test
    provider: openai
    api_base: https://api.openai.com/v1
    models:
      - gpt-4o
default_profile: test
telemetry:
  enabled: true
  service_name: my-team
  headers:
    Authorization: "Bearer token"
  grpc:
    endpoint: "http://localhost:4317"
  http:
    endpoint: "http://localhost:4318"
    protocol: protobuf
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Telemetry == nil {
		t.Fatal("expected Telemetry to be non-nil")
	}
	if !cfg.Telemetry.Enabled {
		t.Error("expected Enabled=true")
	}
	if cfg.Telemetry.ServiceName != "my-team" {
		t.Errorf("ServiceName = %q, want %q", cfg.Telemetry.ServiceName, "my-team")
	}
	if cfg.Telemetry.GRPC == nil || cfg.Telemetry.GRPC.Endpoint != "http://localhost:4317" {
		t.Errorf("GRPC endpoint mismatch")
	}
	if cfg.Telemetry.HTTP == nil || cfg.Telemetry.HTTP.Endpoint != "http://localhost:4318" {
		t.Errorf("HTTP endpoint mismatch")
	}
	if cfg.Telemetry.HTTP.Protocol != "protobuf" {
		t.Errorf("HTTP protocol = %q, want protobuf", cfg.Telemetry.HTTP.Protocol)
	}
}

func TestLoad_WithoutTelemetry(t *testing.T) {
	src := `profiles:
  - name: test
    provider: openai
    api_base: https://api.openai.com/v1
    models:
      - gpt-4o
default_profile: test
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Telemetry != nil {
		t.Errorf("expected Telemetry to be nil, got %+v", cfg.Telemetry)
	}
}

func TestLoad_InvalidTelemetryProtocol(t *testing.T) {
	src := `profiles:
  - name: test
    provider: openai
    api_base: https://api.openai.com/v1
    models:
      - gpt-4o
default_profile: test
telemetry:
  enabled: true
  http:
    endpoint: "http://localhost:4318"
    protocol: invalid
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid protocol")
	}
}

func TestComposeServiceName(t *testing.T) {
	tests := []struct {
		serviceName string
		target      string
		want        string
	}{
		{"my-team", "copilot", "my-team-github-copilot"},
		{"my-team", "codex", "my-team-codex-cli"},
		{"my-team", "codex-app", "my-team-codex-cli"},
		{"my-team", "claude", "my-team-claude-code"},
		{"my-team", "pi", "my-team-pi-coding-agent"},
		{"", "copilot", ""},
		{"", "claude", ""},
	}
	for _, tt := range tests {
		t.Run(tt.serviceName+"_"+tt.target, func(t *testing.T) {
			got := ComposeServiceName(tt.serviceName, tt.target)
			if got != tt.want {
				t.Errorf("ComposeServiceName(%q, %q) = %q, want %q", tt.serviceName, tt.target, got, tt.want)
			}
		})
	}
}
