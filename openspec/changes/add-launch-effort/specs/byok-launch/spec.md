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
