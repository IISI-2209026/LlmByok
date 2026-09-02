## MODIFIED Requirements

### Requirement: Target tool selection and dispatch

The `byok launch` command SHALL accept a target tool name as its first positional argument. The command SHALL dispatch to the `copilot` launch flow when the target is `copilot`, to the `codex` launch flow when the target is `codex`, to the `codex-app` launch flow when the target is `codex-app`, to the `claude` launch flow when the target is `claude`, and to the `pi` launch flow when the target is `pi`. When the target is omitted, the command SHALL write the same launch help content as `byok launch --help` to stdout, SHALL print an error message stating that a target tool is required to stderr, and SHALL exit with code 1 without reading configuration or launching a child process. When the target is any value other than `copilot`, `codex`, `codex-app`, `claude`, or `pi`, the command SHALL print an error message listing the supported target tools and exit with code 1 without additionally printing launch help.

After resolving the profile and model, the launch flow SHALL read the `telemetry` section from the loaded config. When telemetry is enabled and a compatible endpoint is available for the target, the launch flow SHALL pass the Telemetry configuration to the runner, which SHALL inject target-specific OTEL settings into the child process. When telemetry is disabled or no compatible endpoint exists, the launch flow SHALL proceed without telemetry injection.

#### Scenario: Launch copilot dispatches to copilot flow

- **WHEN** user runs `byok launch copilot`
- **THEN** the command SHALL dispatch to the copilot launch flow and behave identically to the existing copilot launch behavior

#### Scenario: Launch codex dispatches to codex flow

- **WHEN** user runs `byok launch codex`
- **THEN** the command SHALL dispatch to the codex launch flow

#### Scenario: Launch codex-app dispatches to codex-app flow

- **WHEN** user runs `byok launch codex-app`
- **THEN** the command SHALL dispatch to the codex-app launch flow which starts `codex app` as a child process

#### Scenario: Launch claude dispatches to claude flow

- **WHEN** user runs `byok launch claude`
- **THEN** the command SHALL dispatch to the claude launch flow

#### Scenario: Launch pi dispatches to pi flow

- **WHEN** user runs `byok launch pi`
- **THEN** the command SHALL dispatch to the pi launch flow

#### Scenario: Omitted target tool displays launch help

- **WHEN** user runs `byok launch` with no positional argument
- **THEN** stdout SHALL include the launch command Usage, Targets, Flags, and Examples, stderr SHALL state that a target tool is required, and the command SHALL exit with code 1 without loading a profile or starting a child process

#### Scenario: Unsupported target tool rejected without help

- **WHEN** user runs `byok launch gemini`
- **THEN** the command SHALL print an error message listing `copilot`, `codex`, `codex-app`, `claude`, and `pi` as supported target tools, SHALL not print the launch help, and SHALL exit with code 1

#### Scenario: Launch with telemetry enabled injects OTEL settings

- **WHEN** user runs `byok launch copilot` with telemetry enabled and http endpoint configured
- **THEN** the copilot child process SHALL receive OTEL environment variables as specified by the byok-telemetry-config spec

#### Scenario: Launch with telemetry disabled does not inject OTEL settings

- **WHEN** user runs `byok launch claude` with telemetry section absent or enabled false
- **THEN** the claude child process SHALL NOT receive any OTEL-related environment variables
