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

The `byok config add` command SHALL append a new profile to the config file with the fields `name`, `provider`, `api_base`, `default_model`, and an api key. The command SHALL support two modes:

1. **Parameter mode**: accepts `--name`, `--provider`, `--api-base`, `--default-model`, `--api-key`, and `--key-storage plaintext|keychain`. `--api-key` SHALL be optional; when omitted no key is stored and the keychain is left untouched.
2. **Interactive mode**: when none of `--name`, `--provider`, `--api-base`, `--default-model`, `--api-key` are provided, the command SHALL prompt the user sequentially for name, provider (default `openai`), api_base, default_model, api_key (SHALL accept an empty value), and key storage choice (`keychain` [default] or `plaintext`). Prompts SHALL use non-echoing input for api_key.

When `--key-storage` is `keychain` (the default when an api_key is provided), the command SHALL store the api_key via `internal/secret.Store` under `profile:<name>` and write the profile with an empty `api_key` field. When `--key-storage` is `plaintext`, the command SHALL write the api_key into the config `api_key` field and SHALL delete any existing keychain entry for `profile:<name>`. When the keychain backend is unavailable and `--key-storage keychain` was selected, the command SHALL print a backend-unavailable error and exit with code 1 without writing the config file. When interactive mode is invoked and stdin is not a terminal, the command SHALL print an error directing the user to parameter mode and exit with code 1.

When the config file does not exist, it SHALL be created. When a profile with the same `name` already exists, the command SHALL print an error and exit with code 1 without modifying the file. When the new profile is the first profile and no `default_profile` is set, `default_profile` SHALL be set to the new profile name.

#### Scenario: Add new profile to empty config via parameters

- **WHEN** user runs `byok config add --name openai-official --provider openai --api-base https://api.openai.com/v1 --api-key sk-xxxx --default-model gpt-4o` and the config file does not exist
- **THEN** a config file is created at `~/.byok/config.yaml` containing the profile `openai-official`, `sk-xxxx` is stored in the keychain under `profile:openai-official`, and the profile `api_key` field is empty

#### Scenario: Add profile with plaintext storage

- **WHEN** user runs `byok config add --name local --provider openai --api-base http://localhost:11434 --default-model llama3 --api-key sk-local --key-storage plaintext`
- **THEN** the profile is written with `api_key: sk-local` in the config file and any prior keychain entry for `profile:local` is deleted

#### Scenario: Add profile without api key

- **WHEN** user runs `byok config add --name openai-official --provider openai --api-base https://api.openai.com/v1 --default-model gpt-4o` without `--api-key`
- **THEN** the profile is created with an empty `api_key` field, no keychain write occurs, and the command succeeds

#### Scenario: Interactive mode prompts for all fields

- **WHEN** user runs `byok config add` in a terminal and enters `openai-official`, `openai`, `https://api.openai.com/v1`, `gpt-4o`, `sk-xxxx`, and accepts the default `keychain` storage
- **THEN** the profile is added with `sk-xxxx` stored in the keychain and the `api_key` field empty in the config file

#### Scenario: Interactive mode rejected on non-tty stdin

- **WHEN** user pipes input into `byok config add` (stdin is not a terminal) and provides no flags
- **THEN** the command prints an error directing the user to use parameter flags and exits with code 1

#### Scenario: Keychain backend unavailable with keychain storage

- **WHEN** user runs `byok config add --name p --api-key sk-x --key-storage keychain` and `internal/secret.Store` returns a backend-unavailable error
- **THEN** the command prints a backend-unavailable error, suggests `--key-storage plaintext`, and exits with code 1 without writing the config file

#### Scenario: Duplicate profile name rejected

- **WHEN** user runs `byok config add --name openai-official ...` and a profile named `openai-official` already exists in the config file
- **THEN** the command prints an error stating the profile name already exists and exits with code 1 without modifying the config file


<!-- @trace
source: simplify-config-cli
updated: 2026-07-06
code:
  - internal/config/interactive.go
  - README.md
  - cmd/config.go
  - AGENTS.md
  - internal/config/config.go
tests:
  - internal/config/interactive_test.go
  - cmd/config_test.go
  - cmd/config_key_test.go
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

The `byok config delete --name <name>` command SHALL remove the profile identified by `--name` from the config file. After removing the profile, the command SHALL attempt to delete the keychain entry `profile:<name>` via `internal/secret.Delete`; the profile deletion SHALL succeed regardless of whether the keychain entry existed. When the keychain delete fails for any reason other than not-found, the command SHALL print a warning naming the profile and the failure but SHALL still exit with code 0. When the named profile does not exist in the config file, the command SHALL print an error and exit with code 1 without modifying the file or touching the keychain. When the removed profile was the `default_profile`, the `default_profile` field SHALL be cleared.

#### Scenario: Delete existing profile with keychain key

- **WHEN** user runs `byok config delete --name local-ollama`, the config file contains the `local-ollama` profile, and the keychain contains `profile:local-ollama`
- **THEN** the `local-ollama` profile is removed from the config file, the keychain entry is deleted, and the command exits with code 0

#### Scenario: Delete existing profile without keychain key

- **WHEN** user runs `byok config delete --name local-ollama` and the config file contains the `local-ollama` profile but the keychain has no entry for it
- **THEN** the `local-ollama` profile is removed from the config file and the command exits with code 0

#### Scenario: Delete non-existent profile

- **WHEN** user runs `byok config delete --name nonexistent`
- **THEN** the command prints an error stating the profile was not found and exits with code 1 without modifying the config file

#### Scenario: Keychain delete failure warns but succeeds

- **WHEN** user runs `byok config delete --name local-ollama` and `internal/secret.Delete` returns a backend-unavailable error after the profile was removed
- **THEN** the command prints a warning naming `local-ollama` and the failure, and exits with code 0


<!-- @trace
source: simplify-config-cli
updated: 2026-07-06
code:
  - internal/config/interactive.go
  - README.md
  - cmd/config.go
  - AGENTS.md
  - internal/config/config.go
tests:
  - internal/config/interactive_test.go
  - cmd/config_test.go
  - cmd/config_key_test.go
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

---
### Requirement: List profiles with key source indicator

The `byok config list` command SHALL, for each profile, display the key source alongside the existing masked key column. The source indicator SHALL be `keychain` when the key was resolved from the OS keychain, `plaintext` when resolved from the config file `api_key` field, and `missing` when neither source yielded a key. The masked key display SHALL use the resolved key value regardless of source.

#### Scenario: Mixed key sources in list

- **WHEN** user runs `byok config list` and profile `openai-official` has its key in the keychain while profile `local-ollama` has a plaintext `api_key` and profile `empty-profile` has no key in either source
- **THEN** the printed output shows `openai-official` with source `keychain`, `local-ollama` with source `plaintext`, and `empty-profile` with source `missing`

<!-- @trace
source: add-keychain-secret-storage
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
-->

---
### Requirement: Update an existing profile

The `byok config update --name <name>` command SHALL update an existing profile in the config file. The command SHALL accept the flags `--provider`, `--api-base`, `--default-model`, `--api-key`, and `--key-storage plaintext|keychain`. When a flag is omitted, the corresponding profile field SHALL retain its current value; `--api-key` omitted means the key is left untouched (no keychain rewrite). When `--api-key` is provided, the command SHALL apply the same key-storage handling as `byok config add` (default `keychain`). The command SHALL support terminal interactive mode identical to `byok config add` when no profile-shaping flags are supplied. When the named profile does not exist, the command SHALL print an error and exit with code 1 without modifying the file. When `--name` is omitted in parameter mode, the command SHALL error and exit with code 1.

#### Scenario: Update model and keep key

- **WHEN** user runs `byok config update --name openai-official --default-model gpt-4o-mini` and the profile exists
- **THEN** only the `default_model` field is updated and the key storage state is unchanged

#### Scenario: Update api key into keychain

- **WHEN** user runs `byok config update --name openai-official --api-key sk-new --key-storage keychain`
- **THEN** `sk-new` is stored in the keychain under `profile:openai-official`, the profile `api_key` field in the config file is cleared, and the command succeeds

#### Scenario: Non-existent profile rejected

- **WHEN** user runs `byok config update --name nonexistent`
- **THEN** the command prints an error stating the profile was not found and exits with code 1 without modifying the config file

<!-- @trace
source: simplify-config-cli
updated: 2026-07-06
code:
  - internal/config/interactive.go
  - README.md
  - cmd/config.go
  - AGENTS.md
  - internal/config/config.go
tests:
  - internal/config/interactive_test.go
  - cmd/config_test.go
  - cmd/config_key_test.go
-->