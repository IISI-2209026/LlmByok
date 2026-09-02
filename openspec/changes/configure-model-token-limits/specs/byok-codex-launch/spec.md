## ADDED Requirements

### Requirement: Codex model token limit mapping

For `codex`, byok SHALL map an effective `context_window_tokens` value to the top-level child-process arguments `--config model_context_window=<value>`. The token override SHALL appear with the existing BYOK config overrides before yolo and passthrough arguments. When context is unset, byok SHALL NOT provide `model_context_window`.

Codex does not expose a supported byok configuration for maximum output tokens. When `max_output_tokens` is effective, byok SHALL follow the common unsupported-parameter behavior: write a warning to stderr, omit an output-token override, and continue launch.

#### Scenario: Codex receives context window override

- **WHEN** effective `context_window_tokens` is `272000`
- **THEN** the Codex child arguments contain `--config` followed by `model_context_window=272000` before yolo and passthrough arguments

#### Scenario: Codex omits unset context window

- **WHEN** effective `context_window_tokens` is unset
- **THEN** the Codex child arguments contain no byok-provided `model_context_window`

#### Scenario: Codex ignores maximum output with warning

- **WHEN** effective `max_output_tokens` is `32768`
- **THEN** Codex receives no output-token override, stderr identifies the ignored target, parameter, value, and source, and the child process still starts

#### Scenario: Codex receives context while ignoring output

- **WHEN** effective context is `1000000` and effective output is `128000`
- **THEN** Codex receives `model_context_window=1000000`, receives no output-token override, and launch writes exactly one unsupported output warning
