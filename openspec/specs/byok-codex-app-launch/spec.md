# byok-codex-app-launch Specification

## Purpose

TBD - created by archiving change 'add-codex-app-launch'. Update Purpose after archive.

## Requirements

### Requirement: Launch Codex desktop app with BYOK profile

The `byok launch codex-app` command SHALL read the selected profile from the byok config file and start the `codex` executable as a child process with the `app` subcommand and BYOK settings injected from the profile. The injection SHALL NOT write to `~/.codex/config.toml` or any other file. The API key SHALL be carried to the codex child process via a `BYOK_CODEX_API_KEY` environment variable set only in the child process environment. The model, model provider, provider base URL, and env key reference SHALL be injected as codex `--config` flag overrides on the command line: `model_provider` set to a fixed custom provider id `byok`, `model_providers.byok.name` set to a fixed display name, `model_providers.byok.base_url` set to the profile `api_base`, `model_providers.byok.env_key` set to `BYOK_CODEX_API_KEY`, and `model` set to the `--model` override when provided otherwise the profile `default_model`. The command-line argument order SHALL be: `codex app` followed by `--config` overrides, followed by yolo flag and passthrough arguments. When no `--profile` flag is provided, the default profile SHALL be used. The child process stdin, stdout, and stderr SHALL be transparently connected to the parent process.

#### Scenario: Launch with default profile

- **WHEN** user runs `byok launch codex-app` with a config file containing a default profile named `openai-official` whose `api_base` is `https://api.openai.com/v1`, `api_key` is `sk-xxxx`, and `default_model` is `gpt-4o`
- **THEN** the `codex` child process is started with `app` as the first command-line argument, `BYOK_CODEX_API_KEY=sk-xxxx` in its environment, and `--config` overrides setting `model="gpt-4o"`, `model_provider="byok"`, `model_providers.byok.name="BYOK"`, `model_providers.byok.base_url="https://api.openai.com/v1"`, and `model_providers.byok.env_key="BYOK_CODEX_API_KEY"`

#### Scenario: Override model with --model flag

- **WHEN** user runs `byok launch codex-app --model gemma4` using a profile whose `default_model` is `gpt-4o`
- **THEN** the `codex` child process receives `app` as the first argument and a `--config` override setting `model="gemma4"` instead of the profile default

#### Scenario: Select profile with --profile flag

- **WHEN** user runs `byok launch codex-app --profile local-ollama`
- **THEN** the `codex` child process is started with `app` as the first argument using the `local-ollama` profile settings instead of the default profile

#### Scenario: App subcommand precedes config flags

- **WHEN** user runs `byok launch codex-app` with any profile
- **THEN** the `codex` child process command-line arguments are ordered as: `app` first, then all `--config` override pairs, then yolo flag and passthrough arguments

<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch_codex_app.go
  - cmd/launch.go
  - internal/runner/codex.go
  - AGENTS.md
  - README.md
tests:
  - cmd/launch_codex_app_test.go
  - cmd/launch_dispatch_test.go
  - internal/runner/codex_app_test.go
-->


<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch.go
  - AGENTS.md
  - cmd/launch_codex.go
  - README.md
  - internal/runner/codex.go
  - cmd/launch_codex_app.go
tests:
  - cmd/launch_codex_app_test.go
  - cmd/launch_dispatch_test.go
  - internal/runner/codex_app_test.go
-->

---
### Requirement: Parent process environment unchanged for codex-app

The `byok` parent process SHALL inject the `BYOK_CODEX_API_KEY` environment variable only into the `codex` child process environment launched by `byok launch codex-app`. The parent process environment and the user shell environment SHALL NOT be modified before, during, or after the launch. The `~/.codex/config.toml` file SHALL NOT be created or modified by `byok`.

#### Scenario: Environment isolation

- **WHEN** user runs `byok launch codex-app` and the launch completes
- **THEN** the parent `byok` process environment variables remain identical to their values before the command ran and `~/.codex/config.toml` is not modified

<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch_codex_app.go
  - internal/runner/codex.go
tests:
  - cmd/launch_codex_app_test.go
  - internal/runner/codex_app_test.go
-->


<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch.go
  - AGENTS.md
  - cmd/launch_codex.go
  - README.md
  - internal/runner/codex.go
  - cmd/launch_codex_app.go
tests:
  - cmd/launch_codex_app_test.go
  - cmd/launch_dispatch_test.go
  - internal/runner/codex_app_test.go
-->

---
### Requirement: Codex executable presence check for codex-app

Before launching, the `byok launch codex-app` command SHALL verify the `codex` executable is resolvable on PATH via `exec.LookPath`. When the executable is not found, the command SHALL print an error message instructing the user to install Codex CLI and exit with code 1.

#### Scenario: Codex not installed

- **WHEN** user runs `byok launch codex-app` and `codex` is not on PATH
- **THEN** the command prints an error message mentioning Codex CLI installation and exits with code 1

<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch_codex_app.go
  - cmd/launch.go
tests:
  - cmd/launch_codex_app_test.go
-->


<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch.go
  - AGENTS.md
  - cmd/launch_codex.go
  - README.md
  - internal/runner/codex.go
  - cmd/launch_codex_app.go
tests:
  - cmd/launch_codex_app_test.go
  - cmd/launch_dispatch_test.go
  - internal/runner/codex_app_test.go
-->

---
### Requirement: Codex-app missing profile error

When the resolved profile (default or named via `--profile`) does not exist in the config file, the `byok launch codex-app` command SHALL print an error message listing available profile names and exit with code 1.

#### Scenario: Named profile missing

- **WHEN** user runs `byok launch codex-app --profile nonexistent` and the config file contains profiles `openai-official` and `local-ollama`
- **THEN** the command prints an error listing `openai-official` and `local-ollama` as available profiles and exits with code 1

<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch_codex_app.go
  - cmd/launch.go
  - internal/config/config.go
tests:
  - cmd/launch_codex_app_test.go
-->


<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch.go
  - AGENTS.md
  - cmd/launch_codex.go
  - README.md
  - internal/runner/codex.go
  - cmd/launch_codex_app.go
tests:
  - cmd/launch_codex_app_test.go
  - cmd/launch_dispatch_test.go
  - internal/runner/codex_app_test.go
-->

---
### Requirement: Codex-app missing config file error

When the config file does not exist, the `byok launch codex-app` command SHALL print an error message instructing the user to run `byok config add` first and exit with code 1.

#### Scenario: No config file

- **WHEN** user runs `byok launch codex-app` and `~/.byok/config.yaml` does not exist
- **THEN** the command prints an error suggesting `byok config add` and exits with code 1

<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch_codex_app.go
  - cmd/launch.go
  - internal/config/config.go
tests:
  - cmd/launch_codex_app_test.go
-->


<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch.go
  - AGENTS.md
  - cmd/launch_codex.go
  - README.md
  - internal/runner/codex.go
  - cmd/launch_codex_app.go
tests:
  - cmd/launch_codex_app_test.go
  - cmd/launch_dispatch_test.go
  - internal/runner/codex_app_test.go
-->

---
### Requirement: Codex-app provider validation

The `byok launch codex-app` command SHALL only accept profiles whose `provider` field is `openai` (empty `provider` defaults to `openai`). When a profile has any other provider value, the command SHALL print an error message stating that only the openai provider is supported and exit with code 1.

#### Scenario: Non-openai provider rejected

- **WHEN** user runs `byok launch codex-app --profile azure-prod` and the `azure-prod` profile has `provider: azure`
- **THEN** the command prints an error stating only the openai provider is supported and exits with code 1

<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch_codex_app.go
  - cmd/launch.go
  - internal/config/config.go
tests:
  - cmd/launch_codex_app_test.go
-->


<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch.go
  - AGENTS.md
  - cmd/launch_codex.go
  - README.md
  - internal/runner/codex.go
  - cmd/launch_codex_app.go
tests:
  - cmd/launch_codex_app_test.go
  - cmd/launch_dispatch_test.go
  - internal/runner/codex_app_test.go
-->

---
### Requirement: Codex-app YOLO mode flag

The `byok launch codex-app` command SHALL accept a `-y` / `--yolo` boolean flag. When the flag is set, the command SHALL append the string `--yolo` to the codex executable arguments after the `--config` overrides and before any passthrough arguments.

#### Scenario: YOLO flag appends --yolo

- **WHEN** user runs `byok launch codex-app --yolo`
- **THEN** the `codex` child process receives the arguments `app`, followed by `--config` overrides, followed by `--yolo`

#### Scenario: Short form -y alias

- **WHEN** user runs `byok launch codex-app -y`
- **THEN** the `codex` child process receives the argument `--yolo` after the `--config` overrides

#### Scenario: YOLO flag combined with passthrough

- **WHEN** user runs `byok launch codex-app -y -- exec`
- **THEN** the `codex` child process receives the arguments `--yolo exec` after the `--config` overrides

<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch_codex_app.go
  - cmd/launch.go
  - internal/runner/codex.go
tests:
  - cmd/launch_codex_app_test.go
  - internal/runner/codex_app_test.go
-->


<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch.go
  - AGENTS.md
  - cmd/launch_codex.go
  - README.md
  - internal/runner/codex.go
  - cmd/launch_codex_app.go
tests:
  - cmd/launch_codex_app_test.go
  - cmd/launch_dispatch_test.go
  - internal/runner/codex_app_test.go
-->

---
### Requirement: Codex-app argument passthrough via double dash

The `byok launch codex-app` command SHALL accept a `--` separator followed by arbitrary arguments. All arguments after the `--` SHALL be forwarded verbatim to the `codex` executable as command-line arguments after the `app` subcommand and `--config` overrides, without parsing or validation by `byok`.

#### Scenario: Single passthrough argument

- **WHEN** user runs `byok launch codex-app -- exec "review this"`
- **THEN** the `codex` child process receives the arguments `app`, `--config` overrides, then `exec "review this"` verbatim

#### Scenario: No passthrough arguments after dash

- **WHEN** user runs `byok launch codex-app --`
- **THEN** the `codex` child process receives zero passthrough arguments (yolo flag still applies if set)

<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch_codex_app.go
  - cmd/launch.go
  - internal/runner/codex.go
tests:
  - cmd/launch_codex_app_test.go
  - internal/runner/codex_app_test.go
-->

<!-- @trace
source: add-codex-app-launch
updated: 2026-07-06
code:
  - cmd/launch.go
  - AGENTS.md
  - cmd/launch_codex.go
  - README.md
  - internal/runner/codex.go
  - cmd/launch_codex_app.go
tests:
  - cmd/launch_codex_app_test.go
  - cmd/launch_dispatch_test.go
  - internal/runner/codex_app_test.go
-->

---
### Requirement: Launch Codex App with an optional reasoning effort

The `byok launch codex-app` command SHALL add `--config model_reasoning_effort="<level>"` to the Codex child process only when a validated `--effort <level>` is provided. The `app` subcommand SHALL remain the first child-process argument, the effort override SHALL follow the existing BYOK `--config` overrides, and yolo and passthrough arguments SHALL remain last. When `--effort` is omitted, the command SHALL NOT add `model_reasoning_effort`.

#### Scenario: Codex App effort override keeps app first

- **WHEN** the user runs `byok launch codex-app --effort xhigh`
- **THEN** the child process arguments SHALL start with `app`, include `--config` followed by `model_reasoning_effort="xhigh"`, and place any yolo or passthrough argument after all config overrides

<!-- @trace
source: add-launch-effort
updated: 2026-07-15
code:
  - cmd/launch_claude.go
  - internal/runner/runner.go
  - cmd/launch_codex.go
  - cmd/launch_dry_run.go
  - internal/runner/claude.go
  - README.md
  - internal/runner/pi.go
  - cmd/launch_pi.go
  - AGENTS.md
  - cmd/launch.go
  - internal/version/version.go
  - internal/runner/codex.go
  - cmd/launch_effort.go
  - cmd/launch_codex_app.go
tests:
  - cmd/launch_dry_run_test.go
  - cmd/launch_effort_test.go
  - internal/runner/launch_effort_test.go
  - cmd/launch_dispatch_test.go
-->