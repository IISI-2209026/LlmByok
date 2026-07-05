# byok-config Specification

## Purpose

TBD - created by archiving change 'add-byok-cli'. Update Purpose after archive.

## Requirements

### Requirement: Config file location

The config file SHALL default to `~/.byok/config.yaml`. The `--config` flag SHALL allow overriding this path for all `byok config` subcommands. The config file SHALL be in YAML format and contain a `profiles` list plus a `default_profile` field.

#### Scenario: Default config path

- **WHEN** user runs `byok config list` without `--config`
- **THEN** the config is read from `~/.byok/config.yaml`


<!-- @trace
source: add-byok-cli
updated: 2026-07-05
code:
  - main.go
  - cmd/root.go
  - cmd/config.go
  - cmd/launch.go
  - internal/runner/testdata/stub/main.go
  - go.mod
  - internal/config/config.go
  - README.md
  - Makefile
  - internal/runner/runner.go
  - go.sum
tests:
  - internal/runner/launch_integration_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
  - cmd/launch_test.go
  - internal/runner/runner_test.go
-->

---
### Requirement: Add a profile

The `byok config add` command SHALL append a new profile to the config file with the fields `name`, `provider`, `api_base`, `api_key`, and `default_model`. When the config file does not exist, it SHALL be created. When a profile with the same `name` already exists, the command SHALL print an error and exit with code 1 without modifying the file.

#### Scenario: Add new profile to empty config

- **WHEN** user runs `byok config add --name openai-official --provider openai --api-base https://api.openai.com/v1 --api-key sk-xxxx --default-model gpt-4o` and the config file does not exist
- **THEN** a config file is created at `~/.byok/config.yaml` containing the profile `openai-official` with the given fields

#### Scenario: Duplicate profile name rejected

- **WHEN** user runs `byok config add --name openai-official ...` and a profile named `openai-official` already exists in the config file
- **THEN** the command prints an error stating the profile name already exists and exits with code 1 without modifying the config file


<!-- @trace
source: add-byok-cli
updated: 2026-07-05
code:
  - main.go
  - cmd/root.go
  - cmd/config.go
  - cmd/launch.go
  - internal/runner/testdata/stub/main.go
  - go.mod
  - internal/config/config.go
  - README.md
  - Makefile
  - internal/runner/runner.go
  - go.sum
tests:
  - internal/runner/launch_integration_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
  - cmd/launch_test.go
  - internal/runner/runner_test.go
-->

---
### Requirement: List profiles with masked API key

The `byok config list` command SHALL print every profile in the config file showing `name`, `provider`, `api_base`, `default_model`, and a masked `api_key`. The masked `api_key` SHALL display only the first 4 and last 4 characters separated by an ellipsis, with all middle characters hidden. When the `api_key` is empty, the displayed value SHALL be an empty string.

#### Scenario: List with masked keys

- **WHEN** user runs `byok config list` and the config file contains a profile `openai-official` with `api_key: sk-abcdefghijklmnopqrstuvwxyz1234567890`
- **THEN** the printed output shows the profile name `openai-official` and the masked api key `sk-a...7890`

##### Example: masking boundary cases

| Input api_key | Expected masked output | Notes |
| ----- | ----- | ----- |
| `sk-abcdefghijklmnopqrstuvwxyz1234567890` | `sk-a...7890` | normal long key |
| `sk-1234` | `sk-1...1234` | short key still masked with available chars |
| `` (empty) | `` (empty) | empty key shown as empty |


<!-- @trace
source: add-byok-cli
updated: 2026-07-05
code:
  - main.go
  - cmd/root.go
  - cmd/config.go
  - cmd/launch.go
  - internal/runner/testdata/stub/main.go
  - go.mod
  - internal/config/config.go
  - README.md
  - Makefile
  - internal/runner/runner.go
  - go.sum
tests:
  - internal/runner/launch_integration_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
  - cmd/launch_test.go
  - internal/runner/runner_test.go
-->

---
### Requirement: Remove a profile

The `byok config remove` command SHALL remove the profile identified by `--name` from the config file. When the named profile does not exist, the command SHALL print an error and exit with code 1 without modifying the file. When the removed profile was the `default_profile`, the `default_profile` field SHALL be cleared.

#### Scenario: Remove existing profile

- **WHEN** user runs `byok config remove --name local-ollama` and the config file contains the `local-ollama` profile
- **THEN** the `local-ollama` profile is removed from the config file and the file is rewritten

#### Scenario: Remove non-existent profile

- **WHEN** user runs `byok config remove --name nonexistent`
- **THEN** the command prints an error stating the profile was not found and exits with code 1 without modifying the config file


<!-- @trace
source: add-byok-cli
updated: 2026-07-05
code:
  - main.go
  - cmd/root.go
  - cmd/config.go
  - cmd/launch.go
  - internal/runner/testdata/stub/main.go
  - go.mod
  - internal/config/config.go
  - README.md
  - Makefile
  - internal/runner/runner.go
  - go.sum
tests:
  - internal/runner/launch_integration_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
  - cmd/launch_test.go
  - internal/runner/runner_test.go
-->

---
### Requirement: Set default profile

The `byok config add` command SHALL set the new profile as `default_profile` when no `default_profile` is currently set in the config file. The `byok config` family SHALL provide a `set-default --name <name>` subcommand that updates the `default_profile` field to the named profile, and SHALL error with exit code 1 when the named profile does not exist.

#### Scenario: First profile becomes default

- **WHEN** user runs `byok config add --name openai-official ...` on a config file with no `default_profile`
- **THEN** the `default_profile` field is set to `openai-official`

#### Scenario: Change default via set-default

- **WHEN** user runs `byok config set-default --name local-ollama` and the `local-ollama` profile exists
- **THEN** the `default_profile` field is updated to `local-ollama`


<!-- @trace
source: add-byok-cli
updated: 2026-07-05
code:
  - main.go
  - cmd/root.go
  - cmd/config.go
  - cmd/launch.go
  - internal/runner/testdata/stub/main.go
  - go.mod
  - internal/config/config.go
  - README.md
  - Makefile
  - internal/runner/runner.go
  - go.sum
tests:
  - internal/runner/launch_integration_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
  - cmd/launch_test.go
  - internal/runner/runner_test.go
-->

---
### Requirement: Config file parse error reporting

When the config file exists but cannot be parsed as valid YAML matching the expected schema, every `byok config` subcommand SHALL print an error message containing the file path and the parse error detail, then exit with code 1.

#### Scenario: Malformed config file

- **WHEN** user runs `byok config list` and `~/.byok/config.yaml` contains invalid YAML
- **THEN** the command prints an error containing the config file path and the YAML parse error and exits with code 1

<!-- @trace
source: add-byok-cli
updated: 2026-07-05
code:
  - main.go
  - cmd/root.go
  - cmd/config.go
  - cmd/launch.go
  - internal/runner/testdata/stub/main.go
  - go.mod
  - internal/config/config.go
  - README.md
  - Makefile
  - internal/runner/runner.go
  - go.sum
tests:
  - internal/runner/launch_integration_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
  - cmd/launch_test.go
  - internal/runner/runner_test.go
-->