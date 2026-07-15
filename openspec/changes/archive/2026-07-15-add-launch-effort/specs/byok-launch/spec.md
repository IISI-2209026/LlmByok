## MODIFIED Requirements

### Requirement: Target tool selection and dispatch

The `byok launch` command SHALL accept a target tool name as its first positional argument. The command SHALL dispatch to the `copilot` launch flow when the target is `copilot`, to the `codex` launch flow when the target is `codex`, to the `codex-app` launch flow when the target is `codex-app`, to the `claude` launch flow when the target is `claude`, and to the `pi` launch flow when the target is `pi`. When the target is omitted, the command SHALL write the same launch help content as `byok launch --help` to stdout, SHALL print an error message stating that a target tool is required to stderr, and SHALL exit with code 1 without reading configuration or launching a child process. When the target is any value other than `copilot`, `codex`, `codex-app`, `claude`, or `pi`, the command SHALL print an error message listing the supported target tools and exit with code 1 without additionally printing launch help.

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

## ADDED Requirements

### Requirement: Dispatch optional reasoning effort

The `byok launch` command SHALL accept an optional `--effort <level>` flag and SHALL pass its validated value to the selected `copilot`, `codex`, `codex-app`, `claude`, or `pi` launch flow. The command SHALL preserve existing model, profile, yolo, and passthrough argument behavior. When the effort value is empty, the command SHALL pass no target-specific effort override.

#### Scenario: Copilot receives selected effort

- **WHEN** the user runs `byok launch copilot --effort medium`
- **THEN** the command SHALL dispatch `medium` to the Copilot launch flow together with the resolved profile, model, and existing extra arguments

### Requirement: Dispatch optional subagent model

The `byok launch` command SHALL accept an optional `--sub-model <model>` flag and SHALL pass its value to the Claude launch flow when the target is `claude`. When the target is `copilot`, `codex`, `codex-app`, or `pi`, the command SHALL accept the flag but SHALL NOT pass a sub-model override to that target launch flow. The command SHALL preserve existing model, profile, effort, yolo, and passthrough argument behavior.

#### Scenario: Non-Claude dispatch ignores subagent model

- **WHEN** the user runs `byok launch pi --sub-model claude-haiku-4-5`
- **THEN** the command SHALL dispatch the pi launch flow without a sub-model override and without an error

### Requirement: Dispatch dry-run command rendering

The `byok launch` command SHALL accept a `--dry-run` flag. When the flag is provided, the command SHALL resolve the selected profile and model, validate the optional effort, and write the platform-specific masked equivalent target command to stdout instead of invoking any target launch flow. The command SHALL use the selected target to choose the equivalent command mapping, SHALL preserve yolo and passthrough argument order, and SHALL not require the target executable to exist on PATH.

#### Scenario: Dry run bypasses target launch flow

- **WHEN** the user runs `byok launch copilot --dry-run`
- **THEN** the command SHALL write the masked Copilot equivalent command to stdout and SHALL NOT invoke the Copilot launch flow or target executable
