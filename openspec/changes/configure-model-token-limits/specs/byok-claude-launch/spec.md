## MODIFIED Requirements

### Requirement: Launch Claude with BYOK profile

The `byok launch claude` command SHALL read the selected profile from the byok config file and start the `claude` executable as a child process with BYOK settings injected from the profile. The injection SHALL NOT write to `~/.claude/settings.json` or any other Claude Code configuration file. The API key, provider base URL, and model SHALL be carried to the `claude` child process via environment variables set only in the child process environment: `ANTHROPIC_BASE_URL` set to the profile `api_base`, `ANTHROPIC_API_KEY` set to the profile api key, and `ANTHROPIC_MODEL` set to the selected model exactly as resolved. Byok SHALL NOT append, remove, or rewrite a `[1m]` suffix. When no `--profile` flag is provided, the default profile SHALL be used. The child process stdin, stdout, and stderr SHALL be transparently connected to the parent process.

#### Scenario: Launch with default profile

- **WHEN** user runs `byok launch claude` with a config file containing a default profile named `openai-official` whose `api_base` is `https://api.openai.com/v1`, `api_key` is `sk-xxxx`, and selected model is `gpt-4o`
- **THEN** the `claude` child process is started with `ANTHROPIC_BASE_URL=https://api.openai.com/v1`, `ANTHROPIC_API_KEY=sk-xxxx`, and `ANTHROPIC_MODEL=gpt-4o` in its environment and zero command-line arguments

#### Scenario: Override model remains unchanged

- **WHEN** user runs `byok launch claude --model claude-sonnet-4-5`
- **THEN** the `claude` child process is started with `ANTHROPIC_MODEL=claude-sonnet-4-5`

#### Scenario: Explicit extended-context suffix remains unchanged

- **WHEN** user runs `byok launch claude --model claude-sonnet-4-5[1m]`
- **THEN** the `claude` child process is started with `ANTHROPIC_MODEL=claude-sonnet-4-5[1m]`

#### Scenario: Select profile with --profile flag

- **WHEN** user runs `byok launch claude --profile local-ollama`
- **THEN** the `claude` child process is started using the `local-ollama` profile settings instead of the default profile

## ADDED Requirements

### Requirement: Claude model token limit mapping

For `claude`, byok SHALL map effective `context_window_tokens` to `CLAUDE_CODE_MAX_CONTEXT_TOKENS` and effective `max_output_tokens` to `CLAUDE_CODE_MAX_OUTPUT_TOKENS` in the child environment. Each unset field SHALL omit the corresponding byok-provided variable. These variables SHALL NOT be written to the parent environment or any Claude Code configuration file.

#### Scenario: Claude receives both token limits

- **WHEN** effective context is `1000000` and effective output is `128000`
- **THEN** the Claude child environment contains `CLAUDE_CODE_MAX_CONTEXT_TOKENS=1000000` and `CLAUDE_CODE_MAX_OUTPUT_TOKENS=128000`

#### Scenario: Claude receives one configured limit

- **WHEN** effective context is `272000` and effective output is unset
- **THEN** the Claude child environment contains `CLAUDE_CODE_MAX_CONTEXT_TOKENS=272000` and no byok-provided `CLAUDE_CODE_MAX_OUTPUT_TOKENS`

#### Scenario: Claude omits all token overrides when unset

- **WHEN** both effective token limits are unset
- **THEN** the Claude child environment contains no byok-provided `CLAUDE_CODE_MAX_CONTEXT_TOKENS` or `CLAUDE_CODE_MAX_OUTPUT_TOKENS`
