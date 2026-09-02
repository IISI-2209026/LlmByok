## ADDED Requirements

### Requirement: Pi model token limit overrides

For `pi`, byok SHALL write effective model token limits into the temporary `models.json` under `providers.openai.modelOverrides.<selected-model>`. Effective `context_window_tokens` SHALL map to `contextWindow`, and effective `max_output_tokens` SHALL map to `maxTokens`. An unset field SHALL be omitted. When both fields are unset, byok SHALL omit the model override and preserve the existing temporary provider override containing only BYOK connection data.

The temporary file SHALL retain mode `0600`, SHALL contain the selected model as an exact JSON object key, and SHALL be removed with the temporary directory after Pi exits.

#### Scenario: Pi receives both model limits

- **WHEN** selected model is `gpt-5.4`, effective context is `1000000`, and effective output is `128000`
- **THEN** temporary `models.json` contains `providers.openai.modelOverrides.gpt-5.4.contextWindow=1000000` and `providers.openai.modelOverrides.gpt-5.4.maxTokens=128000`

#### Scenario: Pi context-only override omits maxTokens

- **WHEN** selected model is `gpt-5.4-mini`, effective context is `272000`, and effective output is unset
- **THEN** its temporary model override contains `contextWindow=272000` and no `maxTokens`

#### Scenario: Pi fully unset limits preserve provider-only file

- **WHEN** both effective token limits are unset
- **THEN** temporary `models.json` contains the existing `baseUrl` and `apiKey` provider override and contains no byok-provided `modelOverrides`

#### Scenario: Pi temporary token configuration is removed

- **WHEN** Pi exits after launch with effective token limits
- **THEN** the temporary directory containing `models.json` and any byok-created settings is removed regardless of the Pi exit status

### Requirement: Pi output limit reserves response headroom

When `max_output_tokens` is effective, byok SHALL write a temporary `settings.json` in `PI_CODING_AGENT_DIR` with `compaction.reserveTokens` equal to the effective output value. The temporary setting SHALL NOT expose a user-configurable compaction threshold and SHALL NOT modify `~/.pi/agent/settings.json` or project-local Pi settings. When output is unset, byok SHALL NOT create a settings override for compaction.

#### Scenario: Pi output limit creates equal headroom

- **WHEN** effective `max_output_tokens` is `128000`
- **THEN** temporary `settings.json` contains `compaction.reserveTokens=128000` and has mode `0600`

#### Scenario: Pi output-only configuration updates model and headroom

- **WHEN** effective output is `32768` and context is unset
- **THEN** temporary `models.json` contains `maxTokens=32768`, temporary `settings.json` contains `reserveTokens=32768`, and no `contextWindow` is present

#### Scenario: Pi unset output uses native compaction default

- **WHEN** effective `max_output_tokens` is unset
- **THEN** byok creates no compaction settings override and Pi uses its native reserve behavior
