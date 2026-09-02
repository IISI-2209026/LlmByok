## ADDED Requirements

### Requirement: Dry-run renders effective token limits

The `--dry-run` flow SHALL use the same resolved model token limits and target mapping as normal launch. Supported effective values SHALL appear in the platform-native equivalent command on stdout. Unset values SHALL be omitted. Unsupported effective values SHALL be omitted from stdout and SHALL produce the same warning on stderr as normal launch. Token-limit rendering SHALL NOT expose an API key or telemetry header.

#### Scenario: Copilot dry-run renders token environment variables

- **WHEN** user runs a Copilot dry-run with effective context `1000000` and output `128000`
- **THEN** stdout contains platform-native assignments for `COPILOT_PROVIDER_MAX_PROMPT_TOKENS=1000000` and `COPILOT_PROVIDER_MAX_OUTPUT_TOKENS=128000`

#### Scenario: Codex dry-run separates warning from command

- **WHEN** user runs a Codex dry-run with effective context `272000` and output `32768`
- **THEN** stdout contains `--config model_context_window=272000`, stdout contains no output-token override or warning text, and stderr contains the unsupported `max_output_tokens` warning

#### Scenario: Unset token limits are absent

- **WHEN** a dry-run has no effective context or output values
- **THEN** stdout contains no token-limit environment variable, Codex token config, or Pi model limit override, and stderr contains no token-limit warning

### Requirement: Pi dry-run renders token configuration lifecycle

When Pi has effective token limits, dry-run stdout SHALL render a complete platform-native temporary-directory lifecycle whose masked `models.json` includes only the effective `contextWindow` and `maxTokens` model override fields. When `max_output_tokens` is effective, the fragment SHALL also create a temporary `settings.json` with `compaction.reserveTokens` equal to that value. Cleanup SHALL cover every temporary file through removal of the temporary directory.

#### Scenario: Pi dry-run renders context output and headroom

- **WHEN** Pi dry-run receives model `gpt-5.4`, context `1000000`, and output `128000`
- **THEN** stdout contains masked temporary configuration with `modelOverrides.gpt-5.4.contextWindow=1000000`, `modelOverrides.gpt-5.4.maxTokens=128000`, `compaction.reserveTokens=128000`, the Pi invocation, and temporary-directory cleanup

#### Scenario: Pi context-only dry-run omits settings file

- **WHEN** Pi dry-run receives context `272000` and no effective output value
- **THEN** stdout includes `contextWindow=272000` in the temporary model override and does not create a byok-provided `settings.json`
