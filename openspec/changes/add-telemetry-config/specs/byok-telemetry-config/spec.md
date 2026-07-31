## ADDED Requirements

### Requirement: Telemetry configuration structure

The config file (`~/.byok/config.yaml`) SHALL support a top-level `telemetry` section with the following fields:
- `enabled` (boolean): controls whether telemetry injection is active
- `service_name` (string, optional): custom service name prefix
- `headers` (map[string]string, optional): key-value pairs for authentication headers
- `grpc` (object, optional): contains `endpoint` (string) for gRPC OTLP endpoint
- `http` (object, optional): contains `endpoint` (string) and `protocol` (string, default `protobuf`)

The `http.protocol` field SHALL accept only `protobuf` or `json`. Any other value SHALL cause config loading to fail with a descriptive error and exit code 1.

#### Scenario: Valid telemetry config with both endpoints

- **WHEN** the config file contains a `telemetry` section with `enabled: true`, `grpc.endpoint: "http://localhost:4317"`, and `http.endpoint: "http://localhost:4318"`
- **THEN** config loading SHALL succeed and the Telemetry struct SHALL have both GRPC and HTTP populated

#### Scenario: Invalid protocol value rejected

- **WHEN** the config file contains `telemetry.http.protocol: "grpc"`
- **THEN** config loading SHALL fail with an error message naming the invalid protocol value and the valid options (`protobuf`, `json`), and exit with code 1

#### Scenario: Telemetry section absent

- **WHEN** the config file has no `telemetry` section
- **THEN** config loading SHALL succeed and the Telemetry field SHALL be nil

##### Example: minimal telemetry config

- **GIVEN** config YAML:
  ```yaml
  telemetry:
    enabled: true
    http:
      endpoint: "http://localhost:4318"
  ```
- **WHEN** config is loaded
- **THEN** Telemetry.Enabled is true, Telemetry.HTTP.Endpoint is "http://localhost:4318", Telemetry.HTTP.Protocol defaults to "protobuf", Telemetry.GRPC is nil

### Requirement: Service name composition

When `telemetry.service_name` is set, the system SHALL compose the injected service name as `<service_name>-<agent-name>` where agent-name is target-specific:
- copilot: `github-copilot`
- codex / codex-app: `codex-cli`
- claude: `claude-code`
- pi: `pi-coding-agent`

When `telemetry.service_name` is not set (empty or omitted), the system SHALL NOT inject any service name, allowing each target to use its native default.

#### Scenario: Service name set produces composite name

- **WHEN** `telemetry.service_name` is `"my-team"` and target is `copilot`
- **THEN** the injected OTEL_SERVICE_NAME SHALL be `my-team-github-copilot`

#### Scenario: Service name unset means no injection

- **WHEN** `telemetry.service_name` is empty and target is `claude`
- **THEN** no OTEL_SERVICE_NAME environment variable SHALL be injected

##### Example: all targets

| service_name | target     | Injected value            |
|-------------|------------|---------------------------|
| "my-team"   | copilot    | my-team-github-copilot    |
| "my-team"   | codex      | my-team-codex-cli         |
| "my-team"   | codex-app  | my-team-codex-cli         |
| "my-team"   | claude     | my-team-claude-code       |
| "my-team"   | pi         | my-team-pi-coding-agent   |
| ""          | copilot    | (not injected)            |

### Requirement: Target endpoint selection

The system SHALL select the endpoint for each target based on native protocol support:
- Targets that support gRPC (codex, codex-app, claude): SHALL use `telemetry.grpc.endpoint` when available; SHALL fall back to `telemetry.http.endpoint` when grpc is not configured
- Targets that only support HTTP (copilot, pi): SHALL use `telemetry.http.endpoint` only

When the required endpoint for a target is not configured (e.g., only grpc is set but target only supports HTTP), the system SHALL silently skip telemetry injection for that target without error.

#### Scenario: gRPC-capable target uses gRPC when both set

- **WHEN** both `telemetry.grpc.endpoint` and `telemetry.http.endpoint` are configured, and target is `codex`
- **THEN** the system SHALL inject using the gRPC endpoint

#### Scenario: gRPC-capable target falls back to HTTP

- **WHEN** only `telemetry.http.endpoint` is configured (grpc is nil), and target is `claude`
- **THEN** the system SHALL inject using the HTTP endpoint with the configured protocol

#### Scenario: HTTP-only target skips when only gRPC set

- **WHEN** only `telemetry.grpc.endpoint` is configured (http is nil), and target is `copilot`
- **THEN** the system SHALL NOT inject any telemetry settings and SHALL NOT report an error

### Requirement: Copilot CLI telemetry injection

When telemetry is enabled and HTTP endpoint is configured, the Copilot runner SHALL inject these environment variables into the child process:
- `COPILOT_OTEL_ENABLED=true`
- `OTEL_EXPORTER_OTLP_ENDPOINT=<http.endpoint>`
- `OTEL_EXPORTER_OTLP_PROTOCOL=http/<http.protocol>` (i.e., `http/protobuf` or `http/json`)
- `OTEL_EXPORTER_OTLP_HEADERS=<key>=<value>,<key>=<value>` (comma-separated, only if headers non-empty)
- `OTEL_SERVICE_NAME=<composed-name>` (only if service_name is set)

#### Scenario: Full Copilot telemetry injection

- **WHEN** telemetry is enabled with http endpoint `http://localhost:4318`, protocol `protobuf`, service_name `my-team`, and headers `Authorization=Bearer xxx`
- **THEN** child process env SHALL contain `COPILOT_OTEL_ENABLED=true`, `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318`, `OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf`, `OTEL_EXPORTER_OTLP_HEADERS=Authorization=Bearer xxx`, `OTEL_SERVICE_NAME=my-team-github-copilot`

### Requirement: Codex telemetry injection

When telemetry is enabled, the Codex runner SHALL inject OTEL config via `--config` flags:
- Using gRPC endpoint: `--config otel.trace_exporter="otlp-grpc"` with `--config otel.trace_exporter.endpoint="<grpc.endpoint>"`
- Using HTTP endpoint: `--config otel.trace_exporter="otlp-http"` with `--config otel.trace_exporter.endpoint="<http.endpoint>"` and `--config otel.trace_exporter.protocol="<binary|json>"`
- Headers: `--config otel.trace_exporter.headers.<key>="<value>"` for each header
- Service name: injected as `OTEL_SERVICE_NAME` environment variable (not via --config)

The same pattern SHALL apply to `otel.exporter` (log exporter) with identical endpoint/protocol values.

#### Scenario: Codex gRPC injection

- **WHEN** telemetry is enabled with grpc endpoint `http://localhost:4317` and service_name `ops`
- **THEN** codex args SHALL include `--config otel.trace_exporter="otlp-grpc"`, `--config otel.trace_exporter.endpoint="http://localhost:4317"`, `--config otel.exporter="otlp-grpc"`, `--config otel.exporter.endpoint="http://localhost:4317"`, and env SHALL contain `OTEL_SERVICE_NAME=ops-codex-cli`

### Requirement: Claude telemetry injection

When telemetry is enabled, the Claude runner SHALL inject these environment variables:
- `CLAUDE_CODE_ENABLE_TELEMETRY=1`
- `OTEL_METRICS_EXPORTER=otlp`
- `OTEL_LOGS_EXPORTER=otlp`
- `OTEL_EXPORTER_OTLP_ENDPOINT=<endpoint>` (gRPC preferred, fallback HTTP)
- `OTEL_EXPORTER_OTLP_PROTOCOL=<grpc|http/protobuf|http/json>` (depending on selected endpoint)
- `OTEL_EXPORTER_OTLP_HEADERS=<key>=<value>,<key>=<value>` (only if headers non-empty)
- `OTEL_SERVICE_NAME=<composed-name>` (only if service_name is set)

#### Scenario: Claude gRPC injection

- **WHEN** telemetry is enabled with grpc endpoint `http://localhost:4317`
- **THEN** claude env SHALL contain `CLAUDE_CODE_ENABLE_TELEMETRY=1`, `OTEL_METRICS_EXPORTER=otlp`, `OTEL_LOGS_EXPORTER=otlp`, `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317`, `OTEL_EXPORTER_OTLP_PROTOCOL=grpc`

### Requirement: Pi telemetry injection

When telemetry is enabled and HTTP endpoint is configured, the Pi runner SHALL inject these environment variables:
- `PI_OTEL_ENABLED=true`
- `OTEL_EXPORTER_OTLP_ENDPOINT=<http.endpoint>`
- `OTEL_SERVICE_NAME=<composed-name>` (only if service_name is set)

Pi does not support headers injection via environment variable; headers SHALL NOT be injected for Pi.

#### Scenario: Pi telemetry injection

- **WHEN** telemetry is enabled with http endpoint `http://localhost:4318` and service_name `team`
- **THEN** pi env SHALL contain `PI_OTEL_ENABLED=true`, `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318`, `OTEL_SERVICE_NAME=team-pi-coding-agent`

### Requirement: Telemetry disabled means no injection

When `telemetry.enabled` is false, or when the telemetry section is absent, the system SHALL NOT inject any telemetry-related environment variables or config flags for any target. The launch behavior SHALL be identical to a config without the telemetry section.

#### Scenario: Enabled false suppresses all injection

- **WHEN** telemetry section exists with `enabled: false` and all endpoints configured
- **THEN** no OTEL-related environment variables or config flags SHALL be present in the child process environment
