## ADDED Requirements

### Requirement: Launch Copilot with BYOK profile

The `byok launch copilot` command SHALL read the specified profile from the config file and start the `copilot` executable as a child process with BYOK environment variables injected from the profile settings. When no `--profile` flag is provided, the default profile SHALL be used. The child process stdin, stdout, and stderr SHALL be transparently connected to the parent process so the user interacts with Copilot normally.

#### Scenario: Launch with default profile

- **WHEN** user runs `byok launch copilot` with a config file containing a default profile named `openai-official` whose `api_base` is `https://api.openai.com/v1` and `default_model` is `gpt-4o`
- **THEN** the `copilot` child process is started with `COPILOT_PROVIDER_BASE_URL=https://api.openai.com/v1`, `COPILOT_PROVIDER_TYPE=openai`, `COPILOT_PROVIDER_API_KEY=<profile api_key>`, and `COPILOT_MODEL=gpt-4o` in its environment

#### Scenario: Override model with --model flag

- **WHEN** user runs `byok launch copilot --model gemma4` using a profile whose `default_model` is `gpt-4o`
- **THEN** the `copilot` child process is started with `COPILOT_MODEL=gemma4` overriding the profile default

#### Scenario: Select profile with --profile flag

- **WHEN** user runs `byok launch copilot --profile local-ollama`
- **THEN** the `copilot` child process is started using the `local-ollama` profile settings instead of the default profile

### Requirement: Parent process environment unchanged

The `byok` parent process SHALL inject BYOK environment variables only into the `copilot` child process environment. The parent process environment and the user shell environment SHALL NOT be modified before, during, or after the launch.

#### Scenario: Environment isolation

- **WHEN** user runs `byok launch copilot` and the launch completes
- **THEN** the parent `byok` process environment variables remain identical to their values before the command ran

### Requirement: Config file path override

The `--config` flag SHALL allow the user to specify an alternate config file path. When omitted, the default path `~/.byok/config.yaml` SHALL be used.

#### Scenario: Custom config path

- **WHEN** user runs `byok launch copilot --config /tmp/my-config.yaml`
- **THEN** the config is read from `/tmp/my-config.yaml` instead of the default path

### Requirement: Copilot executable presence check

Before launching, the `byok launch copilot` command SHALL verify the `copilot` executable is resolvable on PATH via `exec.LookPath`. When the executable is not found, the command SHALL print an error message instructing the user to install Copilot CLI and exit with code 1.

#### Scenario: Copilot not installed

- **WHEN** user runs `byok launch copilot` and `copilot` is not on PATH
- **THEN** the command prints an error message mentioning Copilot CLI installation and exits with code 1

### Requirement: Missing profile error

When the resolved profile (default or named via `--profile`) does not exist in the config file, the `byok launch copilot` command SHALL print an error message listing available profile names and exit with code 1.

#### Scenario: Named profile missing

- **WHEN** user runs `byok launch copilot --profile nonexistent` and the config file contains profiles `openai-official` and `local-ollama`
- **THEN** the command prints an error listing `openai-official` and `local-ollama` as available profiles and exits with code 1

### Requirement: Missing config file error

When the config file does not exist, the `byok launch copilot` command SHALL print an error message instructing the user to run `byok config add` first and exit with code 1.

#### Scenario: No config file

- **WHEN** user runs `byok launch copilot` and `~/.byok/config.yaml` does not exist
- **THEN** the command prints an error suggesting `byok config add` and exits with code 1

### Requirement: Provider validation

The `byok launch copilot` command SHALL only accept profiles whose `provider` field is `openai`. When a profile has any other provider value, the command SHALL print an error message stating that the first version only supports the openai provider and exit with code 1.

#### Scenario: Non-openai provider rejected

- **WHEN** user runs `byok launch copilot --profile azure-prod` and the `azure-prod` profile has `provider: azure`
- **THEN** the command prints an error stating only the openai provider is supported and exits with code 1
