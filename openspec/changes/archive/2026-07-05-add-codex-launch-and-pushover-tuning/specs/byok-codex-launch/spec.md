## ADDED Requirements

### Requirement: Launch Codex with BYOK profile

The `byok launch codex` command SHALL read the selected profile from the byok config file and start the `codex` executable as a child process with BYOK settings injected from the profile. The injection SHALL NOT write to `~/.codex/config.toml` or any other file. The API key SHALL be carried to the codex child process via a `BYOK_CODEX_API_KEY` environment variable set only in the child process environment. The model, model provider, provider base URL, and env key reference SHALL be injected as codex `--config` flag overrides on the command line: `model_provider` set to a fixed custom provider id `byok`, `model_providers.byok.name` set to a fixed display name, `model_providers.byok.base_url` set to the profile `api_base`, `model_providers.byok.env_key` set to `BYOK_CODEX_API_KEY`, and `model` set to the `--model` override when provided otherwise the profile `default_model`. When no `--profile` flag is provided, the default profile SHALL be used. The child process stdin, stdout, and stderr SHALL be transparently connected to the parent process.

#### Scenario: Launch with default profile

- **WHEN** user runs `byok launch codex` with a config file containing a default profile named `openai-official` whose `api_base` is `https://api.openai.com/v1`, `api_key` is `sk-xxxx`, and `default_model` is `gpt-4o`
- **THEN** the `codex` child process is started with `BYOK_CODEX_API_KEY=sk-xxxx` in its environment and command-line `--config` overrides setting `model="gpt-4o"`, `model_provider="byok"`, `model_providers.byok.name="BYOK"`, `model_providers.byok.base_url="https://api.openai.com/v1"`, and `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`

#### Scenario: Override model with --model flag

- **WHEN** user runs `byok launch codex --model gemma4` using a profile whose `default_model` is `gpt-4o`
- **THEN** the `codex` child process receives a `--config` override setting `model="gemma4"` instead of the profile default

#### Scenario: Select profile with --profile flag

- **WHEN** user runs `byok launch codex --profile local-ollama`
- **THEN** the `codex` child process is started using the `local-ollama` profile settings instead of the default profile

### Requirement: Parent process environment unchanged for codex

The `byok` parent process SHALL inject the `BYOK_CODEX_API_KEY` environment variable only into the `codex` child process environment. The parent process environment and the user shell environment SHALL NOT be modified before, during, or after the launch. The `~/.codex/config.toml` file SHALL NOT be created or modified by `byok`.

#### Scenario: Environment isolation

- **WHEN** user runs `byok launch codex` and the launch completes
- **THEN** the parent `byok` process environment variables remain identical to their values before the command ran and `~/.codex/config.toml` is not modified

### Requirement: Codex executable presence check

Before launching, the `byok launch codex` command SHALL verify the `codex` executable is resolvable on PATH via `exec.LookPath`. When the executable is not found, the command SHALL print an error message instructing the user to install Codex CLI and exit with code 1.

#### Scenario: Codex not installed

- **WHEN** user runs `byok launch codex` and `codex` is not on PATH
- **THEN** the command prints an error message mentioning Codex CLI installation and exits with code 1

### Requirement: Codex missing profile error

When the resolved profile (default or named via `--profile`) does not exist in the config file, the `byok launch codex` command SHALL print an error message listing available profile names and exit with code 1.

#### Scenario: Named profile missing

- **WHEN** user runs `byok launch codex --profile nonexistent` and the config file contains profiles `openai-official` and `local-ollama`
- **THEN** the command prints an error listing `openai-official` and `local-ollama` as available profiles and exits with code 1

### Requirement: Codex missing config file error

When the config file does not exist, the `byok launch codex` command SHALL print an error message instructing the user to run `byok config add` first and exit with code 1.

#### Scenario: No config file

- **WHEN** user runs `byok launch codex` and `~/.byok/config.yaml` does not exist
- **THEN** the command prints an error suggesting `byok config add` and exits with code 1

### Requirement: Codex provider validation

The `byok launch codex` command SHALL only accept profiles whose `provider` field is `openai` (empty `provider` defaults to `openai`). When a profile has any other provider value, the command SHALL print an error message stating that only the openai provider is supported and exit with code 1.

#### Scenario: Non-openai provider rejected

- **WHEN** user runs `byok launch codex --profile azure-prod` and the `azure-prod` profile has `provider: azure`
- **THEN** the command prints an error stating only the openai provider is supported and exits with code 1

### Requirement: Codex YOLO mode flag

The `byok launch codex` command SHALL accept a `-y` / `--yolo` boolean flag. When the flag is set, the command SHALL append the string `--yolo` to the codex executable arguments before any passthrough arguments. Note: codex `--yolo` is the documented alias of `--dangerously-bypass-approvals-and-sandbox` (no sandbox, no approvals / full access), per Codex [Sandbox & approvals docs](https://developers.openai.com/codex/agent-approvals-security); byok does not compose `--sandbox danger-full-access --ask-for-approval never` and does not alter codex sandbox/approval behavior beyond forwarding this flag.

#### Scenario: YOLO flag appends --yolo

- **WHEN** user runs `byok launch codex --yolo`
- **THEN** the `codex` child process receives the argument `--yolo`

#### Scenario: Short form -y alias

- **WHEN** user runs `byok launch codex -y`
- **THEN** the `codex` child process receives the argument `--yolo`

#### Scenario: YOLO flag combined with passthrough

- **WHEN** user runs `byok launch codex -y -- exec`
- **THEN** the `codex` child process receives the arguments `--yolo exec` in that order

### Requirement: Codex argument passthrough via double dash

The `byok launch codex` command SHALL accept a `--` separator followed by arbitrary arguments. All arguments after the `--` SHALL be forwarded verbatim to the `codex` executable as command-line arguments, without parsing or validation by `byok`.

#### Scenario: Single passthrough argument

- **WHEN** user runs `byok launch codex -- exec "review this"`
- **THEN** the `codex` child process receives the arguments `exec "review this"` verbatim

#### Scenario: No passthrough arguments after dash

- **WHEN** user runs `byok launch codex --`
- **THEN** the `codex` child process receives zero passthrough arguments (yolo flag still applies if set)
