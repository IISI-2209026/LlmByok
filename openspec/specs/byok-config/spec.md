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

The `byok config add` command SHALL append a new profile to the config file with the fields `name`, `provider`, `api_base`, `api_key`, and `default_model`. The `api_key` field SHALL be optional; when omitted the profile is created without a plaintext key and the user is expected to set the key via `byok config set-key`. When the config file does not exist, it SHALL be created. When a profile with the same `name` already exists, the command SHALL print an error and exit with code 1 without modifying the file.

#### Scenario: Add new profile to empty config

- **WHEN** user runs `byok config add --name openai-official --provider openai --api-base https://api.openai.com/v1 --api-key sk-xxxx --default-model gpt-4o` and the config file does not exist
- **THEN** a config file is created at `~/.byok/config.yaml` containing the profile `openai-official` with the given fields

#### Scenario: Add profile without api key

- **WHEN** user runs `byok config add --name openai-official --provider openai --api-base https://api.openai.com/v1 --default-model gpt-4o` without `--api-key`
- **THEN** the profile is created with an empty `api_key` field and the command succeeds

#### Scenario: Duplicate profile name rejected

- **WHEN** user runs `byok config add --name openai-official ...` and a profile named `openai-official` already exists in the config file
- **THEN** the command prints an error stating the profile name already exists and exits with code 1 without modifying the config file


<!-- @trace
source: add-keychain-secret-storage
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
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

---
### Requirement: Set API key in OS keychain

The `byok config set-key <profile>` command SHALL prompt the user to enter an API key via a non-echoing terminal read (`golang.org/x/term.ReadPassword`), store the entered key in the OS keychain under `profile:<profile.Name>`, and when the profile's `api_key` field in the config file is non-empty SHALL clear that field and rewrite the config file. When the entered key is empty, the command SHALL print "金鑰不可為空" and exit with code 1 without writing to the keychain. When the named profile does not exist in the config file, the command SHALL print "profile <name> 不存在" and exit with code 1.

#### Scenario: Set key for existing profile

- **WHEN** user runs `byok config set-key openai-official` and enters `sk-new` and the profile exists with a plaintext `api_key: sk-old`
- **THEN** `sk-new` is stored in the keychain under `profile:openai-official` and the config file is rewritten with `api_key` empty for that profile

#### Scenario: Empty key rejected

- **WHEN** user runs `byok config set-key openai-official` and presses Enter without typing a key
- **THEN** the command prints "金鑰不可為空" and exits with code 1 and the keychain is not modified

#### Scenario: Non-existent profile rejected

- **WHEN** user runs `byok config set-key nonexistent` and no profile named `nonexistent` exists
- **THEN** the command prints "profile nonexistent 不存在" and exits with code 1


<!-- @trace
source: add-keychain-secret-storage
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
-->

---
### Requirement: Delete API key from OS keychain

The `byok config del-key <profile>` command SHALL delete the key stored under `profile:<profile.Name>` from the OS keychain. When the keychain has no entry for the profile, the command SHALL print "profile <name> 未在 keychain 中" and exit with code 1. When the named profile does not exist in the config file, the command SHALL print "profile <name> 不存在" and exit with code 1.

#### Scenario: Delete existing key

- **WHEN** user runs `byok config del-key openai-official` and the keychain contains `profile:openai-official`
- **THEN** the entry is removed and the command prints "已自 keychain 刪除金鑰（profile: openai-official）"

#### Scenario: Delete missing key

- **WHEN** user runs `byok config del-key openai-official` and the keychain has no entry for the profile
- **THEN** the command prints "profile openai-official 未在 keychain 中" and exits with code 1


<!-- @trace
source: add-keychain-secret-storage
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
-->

---
### Requirement: Batch import plaintext keys into keychain

The `byok config import-keys` command SHALL iterate every profile in the config file whose `api_key` field is non-empty, store each key in the OS keychain under `profile:<profile.Name>`, then clear that profile's `api_key` field and rewrite the config file once after all profiles are processed. When a single profile's keychain store fails, the command SHALL record the failure and continue with the remaining profiles. When all stores succeed, the command SHALL print "匯入 N 個金鑰至 keychain" where N is the count of imported profiles. When one or more profiles failed, the command SHALL print the list of failed profile names and exit with code 1. When no profile has a non-empty `api_key`, the command SHALL print "設定檔中無明碼金鑰可匯入" and exit with code 0.

#### Scenario: Import all plaintext keys

- **WHEN** user runs `byok config import-keys` and the config file has two profiles with non-empty `api_key`
- **THEN** both keys are stored in the keychain, both `api_key` fields are cleared in the rewritten config file, and the command prints "匯入 2 個金鑰至 keychain"

#### Scenario: Partial failure continues and reports

- **WHEN** user runs `byok config import-keys` and the first profile stores successfully but the second fails
- **THEN** the first profile's `api_key` is cleared and the second is left as-is, and the command prints the failed profile name and exits with code 1

#### Scenario: No plaintext keys to import

- **WHEN** user runs `byok config import-keys` and no profile has a non-empty `api_key`
- **THEN** the command prints "設定檔中無明碼金鑰可匯入" and exits with code 0 without rewriting the config file


<!-- @trace
source: add-keychain-secret-storage
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
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