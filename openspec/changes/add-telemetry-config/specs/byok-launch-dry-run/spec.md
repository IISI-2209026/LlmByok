## MODIFIED Requirements

### Requirement: Render platform-specific equivalent commands

The system SHALL render dry-run output as PowerShell syntax on Windows and POSIX shell syntax on non-Windows platforms. For Copilot, Codex, Codex App, and Claude, the output SHALL include the target command, target-specific arguments, and environment assignments required by the normal launch mapping. For pi, the output SHALL be a complete shell fragment that creates a unique temporary directory, writes a `models.json` containing the profile API base and a masked API key placeholder, invokes pi with `PI_CODING_AGENT_DIR` set to that directory and the resolved arguments, and removes the temporary directory on completion. The output SHALL preserve yolo and passthrough argument order.

When telemetry is enabled and a compatible endpoint exists for the target, the dry-run output SHALL include the telemetry-related environment variables or config flags that would be injected during a real launch. Header values in dry-run output SHALL be masked as `***`. When telemetry is disabled or no compatible endpoint exists, dry-run output SHALL NOT include any telemetry-related assignments.

#### Scenario: Windows renders a PowerShell Claude command

- **WHEN** the user runs `byok launch claude --model claude-sonnet-4-5 --sub-model claude-haiku-4-5 --dry-run` on Windows
- **THEN** stdout SHALL contain PowerShell environment assignments for `ANTHROPIC_BASE_URL`, `ANTHROPIC_API_KEY='***'`, `ANTHROPIC_MODEL`, and `CLAUDE_CODE_SUBAGENT_MODEL`, followed by a `claude` command

#### Scenario: Unix renders a POSIX Codex App command

- **WHEN** the user runs `byok launch codex-app --model gpt-5 --effort high --dry-run` on a non-Windows platform
- **THEN** stdout SHALL contain a POSIX environment assignment for `BYOK_CODEX_API_KEY='***'` followed by `codex app`, the existing BYOK `--config` pairs, and `model_reasoning_effort="high"`

#### Scenario: Pi dry run includes temporary configuration lifecycle

- **WHEN** the user runs `byok launch pi --model gpt-5 --effort high --dry-run`
- **THEN** stdout SHALL contain commands that create a temporary directory, write masked `models.json`, set `PI_CODING_AGENT_DIR`, run `pi --model gpt-5 --thinking high`, and remove the temporary directory

#### Scenario: Dry run with telemetry includes OTEL assignments

- **WHEN** the user runs `byok launch copilot --model gpt-5 --dry-run` with telemetry enabled (http endpoint `http://localhost:4318`, headers containing `Authorization`)
- **THEN** stdout SHALL include environment assignments for `COPILOT_OTEL_ENABLED=true`, `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318`, `OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf`, `OTEL_EXPORTER_OTLP_HEADERS=Authorization=***`, and `OTEL_SERVICE_NAME` (if service_name is set)

#### Scenario: Dry run without telemetry omits OTEL assignments

- **WHEN** the user runs `byok launch claude --model claude-sonnet-4-5 --dry-run` with no telemetry section in config
- **THEN** stdout SHALL NOT contain any OTEL-related environment assignments
