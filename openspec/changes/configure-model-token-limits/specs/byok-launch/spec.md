## ADDED Requirements

### Requirement: Resolve model token limits per field

After resolving the selected model name, `byok launch <target>` SHALL independently resolve `context_window_tokens` and `max_output_tokens` in this order: an explicitly supplied launch flag, the matching model item's value, the profile's `default_model_limits` value, then unset. An unset field SHALL NOT cause byok to inject a token override.

When `--model <name>` matches a configured model item, that item's limits SHALL participate in resolution. When the override name is not present in the profile's `models` list, launch SHALL continue to accept the override and SHALL resolve token limits only from launch flags and profile defaults.

#### Scenario: CLI overrides model and profile values independently

- **WHEN** profile defaults are context `272000` and output `16384`, selected model `gpt-5.4` sets context `1000000` and output `128000`, and launch specifies `--context-window-tokens 500000`
- **THEN** the effective context is `500000` from the CLI and the effective output is `128000` from model `gpt-5.4`

#### Scenario: Missing model field falls back to profile default

- **WHEN** selected model `gpt-5.4-mini` sets only `max_output_tokens: 32768` and the profile default context is `272000`
- **THEN** the effective context is `272000` and the effective output is `32768`

#### Scenario: Unknown model override uses profile defaults

- **WHEN** user supplies `--model experimental-model`, no configured model has that name, and profile defaults are context `128000` and output `8192`
- **THEN** launch selects `experimental-model` with effective context `128000` and output `8192` without borrowing limits from another model

#### Scenario: Fully unset limits remain unset

- **WHEN** neither launch flags, the selected model, nor profile defaults provide either token field
- **THEN** both effective token limits are unset and byok injects no token override

### Requirement: Validate launch token flags

The `byok launch <target>` command SHALL accept optional `--context-window-tokens <value>` and `--max-output-tokens <value>` flags for every supported target. Each supplied value MUST be a positive 64-bit integer. Invalid values SHALL produce an error naming the flag and SHALL exit with code 1 before resolving an API key, checking the target executable, or launching a child process.

#### Scenario: Positive token flags accepted

- **WHEN** user runs `byok launch copilot --context-window-tokens 1000000 --max-output-tokens 128000`
- **THEN** launch resolves both CLI values and continues through the normal target flow

#### Scenario: Zero context flag rejected

- **WHEN** user runs `byok launch claude --context-window-tokens 0`
- **THEN** launch prints an error naming `--context-window-tokens`, exits with code 1, and does not resolve a key or start Claude

#### Scenario: Negative output flag rejected

- **WHEN** user runs `byok launch pi --max-output-tokens -1`
- **THEN** launch prints an error naming `--max-output-tokens`, exits with code 1, and does not resolve a key or start Pi

### Requirement: Unsupported token limit is warning-only

When a target does not support an effective token parameter, launch SHALL omit that parameter from the child-process mapping, write a warning to stderr naming the target, parameter, ignored value, and resolution source, and continue the target launch. The warning SHALL NOT be written when the parameter is unset. An unsupported parameter SHALL NOT change the process exit status unless the target process fails for another reason.

#### Scenario: Unsupported configured output is ignored

- **WHEN** Codex receives effective `max_output_tokens: 32768` from the profile default
- **THEN** stderr contains a warning naming `codex`, `max_output_tokens`, `32768`, and the profile-default source, no output-token override is sent, and Codex launch continues

#### Scenario: Unset unsupported value is silent

- **WHEN** Codex receives no effective `max_output_tokens`
- **THEN** no unsupported-token warning is written

### Requirement: Copilot token limit mapping

For `copilot`, byok SHALL map effective `context_window_tokens` directly to `COPILOT_PROVIDER_MAX_PROMPT_TOKENS` and effective `max_output_tokens` to `COPILOT_PROVIDER_MAX_OUTPUT_TOKENS` in the child process environment. Byok SHALL NOT subtract the output value from the context value. When a value is unset, byok SHALL NOT provide the corresponding environment variable. The parent process environment SHALL remain unchanged.

#### Scenario: Copilot receives both explicit limits

- **WHEN** effective context is `1000000` and effective output is `128000`
- **THEN** the Copilot child environment contains `COPILOT_PROVIDER_MAX_PROMPT_TOKENS=1000000` and `COPILOT_PROVIDER_MAX_OUTPUT_TOKENS=128000`

#### Scenario: Copilot no longer receives hard-coded limits

- **WHEN** both effective token limits are unset
- **THEN** the Copilot child environment contains no byok-provided `COPILOT_PROVIDER_MAX_PROMPT_TOKENS` or `COPILOT_PROVIDER_MAX_OUTPUT_TOKENS`

#### Scenario: Copilot mapping performs no arithmetic

- **WHEN** effective context is `200000` and effective output is `64000`
- **THEN** `COPILOT_PROVIDER_MAX_PROMPT_TOKENS` is exactly `200000`, not `136000`
