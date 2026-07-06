## ADDED Requirements

### Requirement: Launch Pi with BYOK profile

The `byok launch pi` command SHALL read the selected profile from the byok config file and start the `pi` executable as a child process with BYOK settings injected from the profile. The injection SHALL NOT write to `~/.pi/agent/models.json` or any other pi configuration file. The injection SHALL create a temporary directory containing a `models.json` file that overrides the `openai` provider with `baseUrl` set to the profile `api_base` and `apiKey` set to the profile api key, and SHALL set the `PI_CODING_AGENT_DIR` environment variable to the temporary directory path in the child process environment only. The model SHALL be passed via the `--model` CLI flag using the `--model` override when provided, otherwise the profile `default_model`. When no `--profile` flag is provided, the default profile SHALL be used. The child process stdin, stdout, and stderr SHALL be transparently connected to the parent process. The temporary directory SHALL be removed after the child process exits, regardless of success or failure.

#### Scenario: Launch with default profile

- **WHEN** user runs `byok launch pi` with a config file containing a default profile named `openai-official` whose `api_base` is `https://api.openai.com/v1`, `api_key` is `sk-xxxx`, and `default_model` is `gpt-4o`
- **THEN** the `pi` child process is started with `PI_CODING_AGENT_DIR` set to a temporary directory containing a `models.json` whose `providers.openai.baseUrl` is `https://api.openai.com/v1` and `providers.openai.apiKey` is `sk-xxxx`, and the command-line argument `--model gpt-4o`

#### Scenario: Override model with --model flag

- **WHEN** user runs `byok launch pi --model claude-sonnet-4-5` using a profile whose `default_model` is `gpt-4o`
- **THEN** the `pi` child process receives the command-line argument `--model claude-sonnet-4-5` instead of the profile default

#### Scenario: Select profile with --profile flag

- **WHEN** user runs `byok launch pi --profile local-ollama`
- **THEN** the `pi` child process is started using the `local-ollama` profile settings instead of the default profile

#### Scenario: Temporary directory cleaned up after exit

- **WHEN** user runs `byok launch pi` and the pi child process exits
- **THEN** the temporary directory created for `PI_CODING_AGENT_DIR` is removed from the filesystem

#### Scenario: User pi config not modified

- **WHEN** user runs `byok launch pi` and the launch completes
- **THEN** `~/.pi/agent/models.json` is not created or modified by byok

---
### Requirement: Parent process environment unchanged for pi

The `byok` parent process SHALL set the `PI_CODING_AGENT_DIR` environment variable only in the `pi` child process environment. The parent process environment and the user shell environment SHALL NOT be modified before, during, or after the launch. The `~/.pi/agent/models.json` file SHALL NOT be created or modified by `byok`.

#### Scenario: Environment isolation

- **WHEN** user runs `byok launch pi` and the launch completes
- **THEN** the parent `byok` process environment variables remain identical to their values before the command ran and `~/.pi/agent/models.json` is not modified

---
### Requirement: Pi executable presence check

Before launching, the `byok launch pi` command SHALL verify the `pi` executable is resolvable on PATH via `exec.LookPath`. When the executable is not found, the command SHALL print an error message instructing the user to install pi CLI and exit with code 1.

#### Scenario: Pi not installed

- **WHEN** user runs `byok launch pi` and `pi` is not on PATH
- **THEN** the command prints an error message mentioning pi CLI installation and exits with code 1

---
### Requirement: Pi missing profile error

When the resolved profile (default or named via `--profile`) does not exist in the config file, the `byok launch pi` command SHALL print an error message listing available profile names and exit with code 1.

#### Scenario: Named profile missing

- **WHEN** user runs `byok launch pi --profile nonexistent` and the config file contains profiles `openai-official` and `local-ollama`
- **THEN** the command prints an error listing `openai-official` and `local-ollama` as available profiles and exits with code 1

---
### Requirement: Pi missing config file error

When the config file does not exist, the `byok launch pi` command SHALL print an error message instructing the user to run `byok config add` first and exit with code 1.

#### Scenario: No config file

- **WHEN** user runs `byok launch pi` and `~/.byok/config.yaml` does not exist
- **THEN** the command prints an error suggesting `byok config add` and exits with code 1

---
### Requirement: Pi provider validation

The `byok launch pi` command SHALL only accept profiles whose `provider` field is `openai` (empty `provider` defaults to `openai`). When a profile has any other provider value, the command SHALL print an error message stating that only the openai provider is supported and exit with code 1.

#### Scenario: Non-openai provider rejected

- **WHEN** user runs `byok launch pi --profile azure-prod` and the `azure-prod` profile has `provider: azure`
- **THEN** the command prints an error stating only the openai provider is supported and exits with code 1

---
### Requirement: Pi YOLO mode flag

The `byok launch pi` command SHALL accept a `-y` / `--yolo` boolean flag. When the flag is set, the command SHALL append the string `--approve` (the pi CLI auto-trust flag equivalent to copilot/codex `--yolo`) to the `pi` executable arguments before any passthrough arguments.

#### Scenario: YOLO flag appends --approve

- **WHEN** user runs `byok launch pi --yolo`
- **THEN** the `pi` child process receives the argument `--approve`

#### Scenario: Short form -y alias

- **WHEN** user runs `byok launch pi -y`
- **THEN** the `pi` child process receives the argument `--approve`

#### Scenario: YOLO flag combined with passthrough

- **WHEN** user runs `byok launch pi -y -- --continue`
- **THEN** the `pi` child process receives the arguments `--approve --continue` in that order

---
### Requirement: Pi argument passthrough via double dash

The `byok launch pi` command SHALL accept a `--` separator followed by arbitrary arguments. All arguments after the `--` SHALL be forwarded verbatim to the `pi` executable as command-line arguments, without parsing or validation by `byok`.

#### Scenario: Single passthrough argument

- **WHEN** user runs `byok launch pi -- --continue`
- **THEN** the `pi` child process receives the argument `--continue`

#### Scenario: No passthrough arguments after dash

- **WHEN** user runs `byok launch pi --`
- **THEN** the `pi` child process receives zero passthrough arguments (yolo flag still applies if set)

---
### Requirement: Pi launch documentation in README

The `README.md` SHALL document `pi` as a supported `byok launch` target alongside `copilot`, `codex`, `codex-app`, and `claude`. The README SHALL include a pi BYOK section in "運作原理" describing that `byok launch pi` creates a temporary directory with `models.json` overriding the `openai` provider `baseUrl` and `apiKey`, sets `PI_CODING_AGENT_DIR` in the child process environment, and passes `--model` as a CLI flag, without writing `~/.pi/agent/models.json`. The README targets table, launch examples, prerequisites, official documentation links, and troubleshooting section SHALL be updated to include `pi`. Introductory text that describes `byok` as specific to Copilot, Codex, or Claude SHALL be generalized to reflect that `byok` supports `copilot`, `codex`, `codex-app`, `claude`, and `pi`.

#### Scenario: Pi appears in targets table

- **WHEN** reader views the `byok launch <target>` Targets table in `README.md`
- **THEN** the table lists `copilot`, `codex`, `codex-app`, `claude`, and `pi` with descriptions

#### Scenario: Pi BYOK section in README

- **WHEN** reader views the "運作原理" section of `README.md`
- **THEN** a "Pi BYOK" subsection describes the `PI_CODING_AGENT_DIR` environment variable, the temporary `models.json` with `providers.openai.baseUrl` and `providers.openai.apiKey`, and the `--model` CLI flag, and states that `~/.pi/agent/models.json` is not modified

#### Scenario: Pi in official documentation links

- **WHEN** reader views the "官方文件" section of `README.md`
- **THEN** a link to the pi.dev providers documentation is listed alongside the Copilot, Codex, and Claude BYOK documentation links
