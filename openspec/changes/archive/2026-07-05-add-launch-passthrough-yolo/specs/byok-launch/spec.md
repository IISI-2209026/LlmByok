## MODIFIED Requirements

### Requirement: Launch Copilot with BYOK profile

The `byok launch copilot` command SHALL read the specified profile from the config file and start the `copilot` executable as a child process with BYOK environment variables injected from the profile settings. When no `--profile` flag is provided, the default profile SHALL be used. The child process stdin, stdout, and stderr SHALL be transparently connected to the parent process so the user interacts with Copilot normally. The command SHALL forward extra arguments (from the `--yolo`/`-y` flag and the `--` passthrough) to the `copilot` executable in the order: yolo flag arguments first, then passthrough arguments.

#### Scenario: Launch with default profile

- **WHEN** user runs `byok launch copilot` with a config file containing a default profile named `openai-official` whose `api_base` is `https://api.openai.com/v1` and `default_model` is `gpt-4o`
- **THEN** the `copilot` child process is started with `COPILOT_PROVIDER_BASE_URL=https://api.openai.com/v1`, `COPILOT_PROVIDER_TYPE=openai`, `COPILOT_PROVIDER_API_KEY=<profile api_key>`, and `COPILOT_MODEL=gpt-4o` in its environment and zero command-line arguments

#### Scenario: Override model with --model flag

- **WHEN** user runs `byok launch copilot --model gemma4` using a profile whose `default_model` is `gpt-4o`
- **THEN** the `copilot` child process is started with `COPILOT_MODEL=gemma4` overriding the profile default and zero command-line arguments

#### Scenario: Select profile with --profile flag

- **WHEN** user runs `byok launch copilot --profile local-ollama`
- **THEN** the `copilot` child process is started using the `local-ollama` profile settings instead of the default profile and zero command-line arguments

#### Scenario: Launch with no extra arguments

- **WHEN** user runs `byok launch copilot` without `-y`/`--yolo` or `--` passthrough
- **THEN** the `copilot` child process receives zero command-line arguments

## ADDED Requirements

### Requirement: YOLO mode flag

The `byok launch copilot` command SHALL accept a `-y` / `--yolo` boolean flag. When the flag is set, the command SHALL append the string `--yolo` to the copilot executable arguments before any passthrough arguments.

#### Scenario: YOLO flag appends --yolo

- **WHEN** user runs `byok launch copilot --yolo`
- **THEN** the `copilot` child process receives the argument `--yolo`

#### Scenario: Short form -y alias

- **WHEN** user runs `byok launch copilot -y`
- **THEN** the `copilot` child process receives the argument `--yolo`

#### Scenario: YOLO flag combined with passthrough

- **WHEN** user runs `byok launch copilot -y -- --continue`
- **THEN** the `copilot` child process receives the arguments `--yolo --continue` in that order

### Requirement: Argument passthrough via double dash

The `byok launch copilot` command SHALL accept a `--` separator followed by arbitrary arguments. All arguments after the `--` SHALL be forwarded verbatim to the `copilot` executable as command-line arguments, without parsing or validation by `byok`.

#### Scenario: Single passthrough argument

- **WHEN** user runs `byok launch copilot -- --continue`
- **THEN** the `copilot` child process receives the argument `--continue`

#### Scenario: Multiple passthrough arguments

- **WHEN** user runs `byok launch copilot -- --continue --model x`
- **THEN** the `copilot` child process receives the arguments `--continue --model x` in that order

#### Scenario: No passthrough arguments after dash

- **WHEN** user runs `byok launch copilot --`
- **THEN** the `copilot` child process receives zero command-line arguments from the passthrough (yolo flag still applies if set)
