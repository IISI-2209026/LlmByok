## MODIFIED Requirements

### Requirement: Launch Claude with BYOK profile

The `byok launch claude` command SHALL read the selected profile from the byok config file and start the `claude` executable as a child process with BYOK settings injected from the profile. The injection SHALL NOT write to `~/.claude/settings.json` or any other Claude Code configuration file. The API key, provider base URL, and model SHALL be carried to the `claude` child process via environment variables set only in the child process environment: `ANTHROPIC_BASE_URL` set to the profile `api_base`, `ANTHROPIC_API_KEY` set to the profile api key, and `ANTHROPIC_MODEL` set to the selected model appended with the suffix `[1m]`. When no `--profile` flag is provided, the default profile SHALL be used. The child process stdin, stdout, and stderr SHALL be transparently connected to the parent process.

#### Scenario: Launch with default profile

- **WHEN** user runs `byok launch claude` with a config file containing a default profile named `openai-official` whose `api_base` is `https://api.openai.com/v1`, `api_key` is `sk-xxxx`, and selected model is `gpt-4o`
- **THEN** the `claude` child process is started with `ANTHROPIC_BASE_URL=https://api.openai.com/v1`, `ANTHROPIC_API_KEY=sk-xxxx`, and `ANTHROPIC_MODEL=gpt-4o[1m]` in its environment and zero command-line arguments

#### Scenario: Override model with --model flag

- **WHEN** user runs `byok launch claude --model claude-sonnet-4-5`
- **THEN** the `claude` child process is started with `ANTHROPIC_MODEL=claude-sonnet-4-5[1m]`

#### Scenario: Select profile with --profile flag

- **WHEN** user runs `byok launch claude --profile local-ollama`
- **THEN** the `claude` child process is started using the `local-ollama` profile settings instead of the default profile

