## ADDED Requirements

### Requirement: Codex App model token limit mapping

For `codex-app`, byok SHALL map an effective `context_window_tokens` value to `--config model_context_window=<value>` while keeping `app` as the first child-process argument. The context override SHALL follow the existing BYOK config overrides and precede yolo and passthrough arguments. When context is unset, byok SHALL NOT provide `model_context_window`.

Codex App uses the same Codex configuration contract and does not expose a supported byok configuration for maximum output tokens. When `max_output_tokens` is effective, byok SHALL write a warning to stderr, omit an output-token override, and continue opening the app.

#### Scenario: Codex App receives context after app subcommand

- **WHEN** effective `context_window_tokens` is `272000`
- **THEN** the child arguments begin with `app` and later contain `--config` followed by `model_context_window=272000`

#### Scenario: Codex App omits unset context window

- **WHEN** effective `context_window_tokens` is unset
- **THEN** the Codex App child arguments contain no byok-provided `model_context_window`

#### Scenario: Codex App ignores maximum output with warning

- **WHEN** effective `max_output_tokens` is `32768`
- **THEN** Codex App receives no output-token override, stderr identifies the ignored target, parameter, value, and source, and the app launch continues
