## ADDED Requirements

### Requirement: Launch Codex with an optional reasoning effort

The `byok launch codex` command SHALL add `--config model_reasoning_effort="<level>"` to the Codex child process only when a validated `--effort <level>` is provided. This override SHALL be a top-level Codex configuration key and SHALL appear with the existing BYOK `--config` overrides before yolo and passthrough arguments. When `--effort` is omitted, the command SHALL NOT add `model_reasoning_effort`.

#### Scenario: Codex effort override

- **WHEN** the user runs `byok launch codex --effort high`
- **THEN** the Codex child process SHALL receive `--config` followed by `model_reasoning_effort="high"` before any yolo or passthrough argument

#### Scenario: Codex omits effort override by default

- **WHEN** the user runs `byok launch codex` without `--effort`
- **THEN** the Codex child process SHALL not receive a `model_reasoning_effort` configuration override
