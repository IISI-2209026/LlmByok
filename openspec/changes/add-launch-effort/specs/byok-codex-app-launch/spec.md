## ADDED Requirements

### Requirement: Launch Codex App with an optional reasoning effort

The `byok launch codex-app` command SHALL add `--config model_reasoning_effort="<level>"` to the Codex child process only when a validated `--effort <level>` is provided. The `app` subcommand SHALL remain the first child-process argument, the effort override SHALL follow the existing BYOK `--config` overrides, and yolo and passthrough arguments SHALL remain last. When `--effort` is omitted, the command SHALL NOT add `model_reasoning_effort`.

#### Scenario: Codex App effort override keeps app first

- **WHEN** the user runs `byok launch codex-app --effort xhigh`
- **THEN** the child process arguments SHALL start with `app`, include `--config` followed by `model_reasoning_effort="xhigh"`, and place any yolo or passthrough argument after all config overrides
