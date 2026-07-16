## ADDED Requirements

### Requirement: Optional target-specific reasoning effort

The system SHALL provide an optional `--effort <level>` flag on `byok launch <target>`. When the flag is omitted, the system SHALL NOT add an effort-specific argument or environment variable to the target child process. When the flag is provided, the system SHALL validate the value for the selected target before starting the child process. For `copilot`, valid levels SHALL be `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, and `max`. For `codex` and `codex-app`, valid levels SHALL be `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, and `max`. For `claude`, valid levels SHALL be `low`, `medium`, `high`, `xhigh`, and `max`. For `pi`, valid levels SHALL be `off`, `minimal`, `low`, `medium`, `high`, `xhigh`, and `max`.

#### Scenario: Omitted effort preserves native defaults

- **WHEN** the user runs `byok launch copilot` without `--effort`
- **THEN** the system SHALL start the Copilot child process without an effort-specific command-line argument or environment variable

#### Scenario: Invalid target-specific effort is rejected

- **WHEN** the user runs `byok launch claude --effort none`
- **THEN** the system SHALL print an error naming `claude`, `none`, and the valid Claude levels, exit with code 1, and SHALL NOT start a child process

#### Scenario: Valid effort dispatches to the target flow

- **WHEN** the user runs `byok launch pi --effort high`
- **THEN** the system SHALL validate `high` for `pi` and dispatch the resolved effort to the pi launch flow

### Requirement: Optional Claude subagent model selection

The system SHALL provide an optional `--sub-model <model>` flag on `byok launch <target>`. The system SHALL treat the model string as opaque and SHALL NOT validate it. When the selected target is `claude`, the system SHALL pass the value to the Claude launch flow. When the selected target is `copilot`, `codex`, `codex-app`, or `pi`, the system SHALL accept the flag, SHALL NOT report an error, and SHALL NOT add a sub-model argument or environment variable to the child process. When the flag is omitted, the system SHALL not add a subagent-model override.

#### Scenario: Claude receives an opaque subagent model

- **WHEN** the user runs `byok launch claude --sub-model claude-haiku-4-5`
- **THEN** the system SHALL dispatch `claude-haiku-4-5` to the Claude launch flow without validating the model identifier

#### Scenario: Non-Claude target ignores subagent model

- **WHEN** the user runs `byok launch codex --sub-model claude-haiku-4-5`
- **THEN** the system SHALL start the Codex child process without a subagent-model override and without an error
