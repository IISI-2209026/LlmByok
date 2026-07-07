# byok-config Specification

## Purpose

TBD - created by archiving change 'add-byok-cli'. Update Purpose after archive.

## Requirements

### Requirement: Config file location

The config file SHALL default to `~/.byok/config.yaml`. The `--config` flag SHALL allow overriding this path for all `byok config` subcommands. The config file SHALL be in YAML format and contain a `profiles` list plus a `default_profile` field. Each profile SHALL carry a `models` list of candidate model identifiers instead of a single `default_model` field. When a config file containing a legacy `default_model` field is loaded and the profile's `models` list is empty, the loader SHALL migrate the `default_model` value into a single-element `models` list. When saving, the config writer SHALL NOT emit the legacy `default_model` field.

#### Scenario: Default config path

- **WHEN** user runs `byok config list` without `--config`
- **THEN** the config is read from `~/.byok/config.yaml`

#### Scenario: Legacy default_model migrated on load

- **WHEN** a config file contains a profile `openai-official` with `default_model: gpt-4o` and no `models` list, and the user runs any `byok config` or `byok launch` command that loads the config
- **THEN** the loaded profile exposes a `models` list of `["gpt-4o"]` and no `default_model` is required for subsequent operations

#### Scenario: Saved config omits legacy default_model

- **WHEN** the user runs a command that writes the config file after loading a legacy profile
- **THEN** the written file contains the profile's `models` list and does not contain a `default_model` field


<!-- @trace
source: add-model-selection
updated: 2026-07-07
code:
  - cmd/launch_codex_app.go
  - cmd/config.go
  - cmd/set_models.go
  - internal/config/config.go
  - internal/config/models_windows.go
  - internal/runner/pi.go
  - internal/config/models.go
  - internal/runner/runner.go
  - internal/runner/claude.go
  - cmd/launch.go
  - CLAUDE.md
  - AGENTS.md
  - cmd/launch_codex.go
  - cmd/launch_pi.go
  - README.md
  - cmd/launch_claude.go
  - internal/runner/codex.go
  - internal/config/models_unix.go
tests:
  - internal/runner/launch_integration_test.go
  - internal/runner/runner_test.go
  - cmd/launch_model_test.go
  - internal/config/models_test.go
  - cmd/set_models_test.go
  - internal/runner/codex_test.go
  - internal/runner/claude_test.go
  - internal/runner/codex_launch_test.go
  - internal/config/interactive_test.go
  - internal/runner/pi_test.go
  - internal/runner/codex_app_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
-->

---
### Requirement: Add a profile

The `byok config add <profile name>` command SHALL append a new profile to the config file. The profile name SHALL be supplied as the first positional argument; the legacy `--name` flag SHALL NOT be accepted. The command SHALL accept the flags `--provider`, `--api-base`, `--api-key`, and `--key-storage plaintext|keychain`. Candidate models SHALL NOT be set by `add`; they are managed by `byok config set-models`. The new profile SHALL be created with an empty `models` list. The command SHALL support two modes:

1. **Parameter mode**: accepts the positional profile name plus `--provider`, `--api-base`, `--api-key`, and `--key-storage`. `--api-key` SHALL be optional; when omitted no key is stored and the keychain is left untouched. `--api-base` SHALL be required.
2. **Interactive mode**: when none of `--provider`, `--api-base`, `--api-key` are provided, the command SHALL prompt the user sequentially for provider (default `openai`), api_base, api_key (SHALL accept an empty value), and key storage choice (`keychain` [default] or `plaintext`). The profile name positional argument SHALL be required in both modes. Prompts SHALL use non-echoing input for api_key. Interactive mode SHALL NOT prompt for a model.

When `--key-storage` is `keychain` (the default when an api_key is provided), the command SHALL store the api_key via `internal/secret.Store` under `profile:<name>` and write the profile with an empty `api_key` field. When `--key-storage` is `plaintext`, the command SHALL write the api_key into the config `api_key` field and SHALL delete any existing keychain entry for `profile:<name>`. When the keychain backend is unavailable and `--key-storage keychain` was selected, the command SHALL print a backend-unavailable error and exit with code 1 without writing the config file. When interactive mode is invoked and stdin is not a terminal, the command SHALL print an error directing the user to parameter mode and exit with code 1.

When the config file does not exist, it SHALL be created. When a profile with the same name already exists, the command SHALL print an error and exit with code 1 without modifying the file. When the new profile is the first profile and no `default_profile` is set, `default_profile` SHALL be set to the new profile name.

#### Scenario: Add new profile to empty config via parameters

- **WHEN** user runs `byok config add openai-official --provider openai --api-base https://api.openai.com/v1 --api-key sk-xxxx` and the config file does not exist
- **THEN** a config file is created at `~/.byok/config.yaml` containing the profile `openai-official` with an empty `models` list, `sk-xxxx` is stored in the keychain under `profile:openai-official`, and the profile `api_key` field is empty

#### Scenario: Add profile with plaintext storage

- **WHEN** user runs `byok config add local --provider openai --api-base http://localhost:11434 --api-key sk-local --key-storage plaintext`
- **THEN** the profile is written with `api_key: sk-local` in the config file, an empty `models` list, and any prior keychain entry for `profile:local` is deleted

#### Scenario: Add profile without api key

- **WHEN** user runs `byok config add openai-official --provider openai --api-base https://api.openai.com/v1` without `--api-key`
- **THEN** the profile is created with an empty `api_key` field, an empty `models` list, no keychain write occurs, and the command succeeds

#### Scenario: Interactive mode prompts for all fields

- **WHEN** user runs `byok config add openai-official` in a terminal and enters `openai`, `https://api.openai.com/v1`, `sk-xxxx`, and accepts the default `keychain` storage
- **THEN** the profile is added with `sk-xxxx` stored in the keychain, the `api_key` field empty in the config file, and an empty `models` list

#### Scenario: Interactive mode rejected on non-tty stdin

- **WHEN** user pipes input into `byok config add openai-official` (stdin is not a terminal) and provides no field flags
- **THEN** the command prints an error directing the user to use parameter flags and exits with code 1

#### Scenario: Missing positional profile name rejected

- **WHEN** user runs `byok config add` without a positional profile name
- **THEN** the command prints an error stating a profile name is required and exits with code 1

#### Scenario: Keychain backend unavailable with keychain storage

- **WHEN** user runs `byok config add p --api-key sk-x --key-storage keychain` and `internal/secret.Store` returns a backend-unavailable error
- **THEN** the command prints a backend-unavailable error, suggests `--key-storage plaintext`, and exits with code 1 without writing the config file

#### Scenario: Duplicate profile name rejected

- **WHEN** user runs `byok config add openai-official ...` and a profile named `openai-official` already exists in the config file
- **THEN** the command prints an error stating the profile name already exists and exits with code 1 without modifying the config file


<!-- @trace
source: add-model-selection
updated: 2026-07-07
code:
  - cmd/launch_codex_app.go
  - cmd/config.go
  - cmd/set_models.go
  - internal/config/config.go
  - internal/config/models_windows.go
  - internal/runner/pi.go
  - internal/config/models.go
  - internal/runner/runner.go
  - internal/runner/claude.go
  - cmd/launch.go
  - CLAUDE.md
  - AGENTS.md
  - cmd/launch_codex.go
  - cmd/launch_pi.go
  - README.md
  - cmd/launch_claude.go
  - internal/runner/codex.go
  - internal/config/models_unix.go
tests:
  - internal/runner/launch_integration_test.go
  - internal/runner/runner_test.go
  - cmd/launch_model_test.go
  - internal/config/models_test.go
  - cmd/set_models_test.go
  - internal/runner/codex_test.go
  - internal/runner/claude_test.go
  - internal/runner/codex_launch_test.go
  - internal/config/interactive_test.go
  - internal/runner/pi_test.go
  - internal/runner/codex_app_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
-->

---
### Requirement: List profiles with masked API key

The `byok config list` command SHALL print every profile in the config file showing `name`, `provider`, `api_base`, the candidate `models` list, and a masked `api_key`. The `models` list SHALL be rendered as a comma-separated string (e.g. `gpt-4o, gpt-4o-mini`); an empty list SHALL be rendered as an empty string. The masked `api_key` SHALL display only the first 4 and last 4 characters separated by an ellipsis, with all middle characters hidden. When the `api_key` is empty, the displayed value SHALL be an empty string.

The output SHALL be a column-aligned table with a header row followed by one row per profile. Column widths SHALL be computed dynamically as the maximum display width of the header and every row's value for that column, with a fixed gap of two spaces between columns. Display width SHALL account for double-width (CJK/full-width) characters counting as 2 columns, so that headers containing Chinese characters (e.g. 名稱, 模型, 來源) align with ASCII row values. Values longer than any fixed width SHALL expand their column rather than overflowing into the next column.

#### Scenario: List with masked keys and models

- **WHEN** user runs `byok config list` and the config file contains a profile `openai-official` with `models: [gpt-4o, gpt-4o-mini]` and `api_key: sk-abcdefghijklmnopqrstuvwxyz1234567890`
- **THEN** the printed output shows the profile name `openai-official`, the models `gpt-4o, gpt-4o-mini`, and the masked api key `sk-a...7890`

#### Scenario: List profile with empty models

- **WHEN** user runs `byok config list` and a profile has an empty `models` list
- **THEN** the printed output shows an empty models value for that profile

#### Scenario: List table columns align despite CJK headers and long model values

- **WHEN** user runs `byok config list` and the config file contains a profile `litellm` with `models: [glm-5.2, kimi-k2.7-code]` (a value longer than a fixed column width) and `api_base: https://llm.homeplus.i234.me`, and the header row uses Chinese column titles (名稱, 模型, 來源)
- **THEN** every column's value starts at the same display-width position in the header row and in the data row, so the long model value expands the model column and the Chinese headers align with the ASCII row values without shifting subsequent columns

##### Example: masking boundary cases

| Input api_key | Expected masked output | Notes |
| ----- | ----- | ----- |
| `sk-abcdefghijklmnopqrstuvwxyz1234567890` | `sk-a...7890` | normal long key |
| `sk-1234` | `sk-1...1234` | short key still masked with available chars |
| `` (empty) | `` (empty) | empty key shown as empty |


<!-- @trace
source: add-model-selection
updated: 2026-07-07
code:
  - cmd/launch_codex_app.go
  - cmd/config.go
  - cmd/set_models.go
  - internal/config/config.go
  - internal/config/models_windows.go
  - internal/runner/pi.go
  - internal/config/models.go
  - internal/runner/runner.go
  - internal/runner/claude.go
  - cmd/launch.go
  - CLAUDE.md
  - AGENTS.md
  - cmd/launch_codex.go
  - cmd/launch_pi.go
  - README.md
  - cmd/launch_claude.go
  - internal/runner/codex.go
  - internal/config/models_unix.go
tests:
  - internal/runner/launch_integration_test.go
  - internal/runner/runner_test.go
  - cmd/launch_model_test.go
  - internal/config/models_test.go
  - cmd/set_models_test.go
  - internal/runner/codex_test.go
  - internal/runner/claude_test.go
  - internal/runner/codex_launch_test.go
  - internal/config/interactive_test.go
  - internal/runner/pi_test.go
  - internal/runner/codex_app_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
-->

---
### Requirement: Remove a profile

The `byok config delete <profile name>` command SHALL remove the profile identified by the positional profile name argument from the config file; the legacy `--name` flag SHALL NOT be accepted. After removing the profile, the command SHALL attempt to delete the keychain entry `profile:<name>` via `internal/secret.Delete`; the profile deletion SHALL succeed regardless of whether the keychain entry existed. When the keychain delete fails for any reason other than not-found, the command SHALL print a warning naming the profile and the failure but SHALL still exit with code 0. When the named profile does not exist in the config file, the command SHALL print an error and exit with code 1 without modifying the file or touching the keychain. When the removed profile was the `default_profile`, the `default_profile` field SHALL be cleared.

#### Scenario: Delete existing profile with keychain key

- **WHEN** user runs `byok config delete local-ollama`, the config file contains the `local-ollama` profile, and the keychain contains `profile:local-ollama`
- **THEN** the `local-ollama` profile is removed from the config file, the keychain entry is deleted, and the command exits with code 0

#### Scenario: Delete existing profile without keychain key

- **WHEN** user runs `byok config delete local-ollama` and the config file contains the `local-ollama` profile but the keychain has no entry for it
- **THEN** the `local-ollama` profile is removed from the config file and the command exits with code 0

#### Scenario: Delete non-existent profile

- **WHEN** user runs `byok config delete nonexistent`
- **THEN** the command prints an error stating the profile was not found and exits with code 1 without modifying the config file

#### Scenario: Keychain delete failure warns but succeeds

- **WHEN** user runs `byok config delete local-ollama` and `internal/secret.Delete` returns a backend-unavailable error after the profile was removed
- **THEN** the command prints a warning naming `local-ollama` and the failure, and exits with code 0


<!-- @trace
source: add-model-selection
updated: 2026-07-07
code:
  - cmd/launch_codex_app.go
  - cmd/config.go
  - cmd/set_models.go
  - internal/config/config.go
  - internal/config/models_windows.go
  - internal/runner/pi.go
  - internal/config/models.go
  - internal/runner/runner.go
  - internal/runner/claude.go
  - cmd/launch.go
  - CLAUDE.md
  - AGENTS.md
  - cmd/launch_codex.go
  - cmd/launch_pi.go
  - README.md
  - cmd/launch_claude.go
  - internal/runner/codex.go
  - internal/config/models_unix.go
tests:
  - internal/runner/launch_integration_test.go
  - internal/runner/runner_test.go
  - cmd/launch_model_test.go
  - internal/config/models_test.go
  - cmd/set_models_test.go
  - internal/runner/codex_test.go
  - internal/runner/claude_test.go
  - internal/runner/codex_launch_test.go
  - internal/config/interactive_test.go
  - internal/runner/pi_test.go
  - internal/runner/codex_app_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
-->

---
### Requirement: Set default profile

The `byok config add` command SHALL set the new profile as `default_profile` when no `default_profile` is currently set in the config file. The `byok config` family SHALL provide a `set-default <profile name>` subcommand that updates the `default_profile` field to the named profile supplied as a positional argument; the legacy `--name` flag SHALL NOT be accepted. The command SHALL error with exit code 1 when the named profile does not exist.

#### Scenario: First profile becomes default

- **WHEN** user runs `byok config add openai-official ...` on a config file with no `default_profile`
- **THEN** the `default_profile` field is set to `openai-official`

#### Scenario: Change default via set-default

- **WHEN** user runs `byok config set-default local-ollama` and the `local-ollama` profile exists
- **THEN** the `default_profile` field is updated to `local-ollama`


<!-- @trace
source: add-model-selection
updated: 2026-07-07
code:
  - cmd/launch_codex_app.go
  - cmd/config.go
  - cmd/set_models.go
  - internal/config/config.go
  - internal/config/models_windows.go
  - internal/runner/pi.go
  - internal/config/models.go
  - internal/runner/runner.go
  - internal/runner/claude.go
  - cmd/launch.go
  - CLAUDE.md
  - AGENTS.md
  - cmd/launch_codex.go
  - cmd/launch_pi.go
  - README.md
  - cmd/launch_claude.go
  - internal/runner/codex.go
  - internal/config/models_unix.go
tests:
  - internal/runner/launch_integration_test.go
  - internal/runner/runner_test.go
  - cmd/launch_model_test.go
  - internal/config/models_test.go
  - cmd/set_models_test.go
  - internal/runner/codex_test.go
  - internal/runner/claude_test.go
  - internal/runner/codex_launch_test.go
  - internal/config/interactive_test.go
  - internal/runner/pi_test.go
  - internal/runner/codex_app_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
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

The `byok config update <profile name>` command SHALL update an existing profile in the config file. The profile name SHALL be supplied as the first positional argument; the legacy `--name` flag SHALL NOT be accepted. The command SHALL accept the flags `--provider`, `--api-base`, `--api-key`, and `--key-storage plaintext|keychain`. Candidate models SHALL NOT be editable via `update`; they are managed by `byok config set-models`. When a flag is omitted, the corresponding profile field SHALL retain its current value; `--api-key` omitted means the key is left untouched (no keychain rewrite). When `--api-key` is provided, the command SHALL apply the same key-storage handling as `byok config add` (default `keychain`). The command SHALL support terminal interactive mode identical to `byok config add` when no profile-shaping flags (`--provider`, `--api-base`, `--api-key`) are supplied. The positional profile name SHALL be required in both modes. When the named profile does not exist, the command SHALL print an error and exit with code 1 without modifying the file.

#### Scenario: Update api base and keep key

- **WHEN** user runs `byok config update openai-official --api-base https://api.openai.com/v2` and the profile exists
- **THEN** only the `api_base` field is updated and the key storage state and models list are unchanged

#### Scenario: Update api key into keychain

- **WHEN** user runs `byok config update openai-official --api-key sk-new --key-storage keychain`
- **THEN** `sk-new` is stored in the keychain under `profile:openai-official`, the profile `api_key` field in the config file is cleared, the models list is unchanged, and the command succeeds

#### Scenario: Non-existent profile rejected

- **WHEN** user runs `byok config update nonexistent`
- **THEN** the command prints an error stating the profile was not found and exits with code 1 without modifying the config file


<!-- @trace
source: add-model-selection
updated: 2026-07-07
code:
  - cmd/launch_codex_app.go
  - cmd/config.go
  - cmd/set_models.go
  - internal/config/config.go
  - internal/config/models_windows.go
  - internal/runner/pi.go
  - internal/config/models.go
  - internal/runner/runner.go
  - internal/runner/claude.go
  - cmd/launch.go
  - CLAUDE.md
  - AGENTS.md
  - cmd/launch_codex.go
  - cmd/launch_pi.go
  - README.md
  - cmd/launch_claude.go
  - internal/runner/codex.go
  - internal/config/models_unix.go
tests:
  - internal/runner/launch_integration_test.go
  - internal/runner/runner_test.go
  - cmd/launch_model_test.go
  - internal/config/models_test.go
  - cmd/set_models_test.go
  - internal/runner/codex_test.go
  - internal/runner/claude_test.go
  - internal/runner/codex_launch_test.go
  - internal/config/interactive_test.go
  - internal/runner/pi_test.go
  - internal/runner/codex_app_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
-->

---
### Requirement: Set candidate models for a profile

The `byok config set-models <profile name>` command SHALL set the candidate model list for an existing profile, fully replacing the profile's `models` list. The command SHALL be registered as a subcommand of `byok config` (alongside `add`/`delete`/`set-default`/`update`/`list`), not as a top-level command. The profile name SHALL be supplied as the first positional argument. The command SHALL accept a repeatable `--model` flag; each occurrence appends one candidate model identifier to the new list, in the order supplied. The command SHALL support two modes:

1. **Parameter mode**: one or more `--model` flags supply the new candidate list, which fully overwrites the existing `models` list.
2. **Interactive mode**: when no `--model` flag is provided and stdin is a terminal, the command SHALL prompt the user to enter model identifiers one per line until an empty line is submitted, then SHALL store the collected non-empty entries as the new `models` list.

When the resulting `models` list is empty, the command SHALL print an error stating that at least one model is required and exit with code 1 without modifying the config file. When the named profile does not exist, the command SHALL print an error listing available profile names and exit with code 1 without modifying the file. When interactive mode is invoked and stdin is not a terminal, the command SHALL print an error directing the user to parameter mode and exit with code 1.

#### Scenario: Set multiple models via flags

- **WHEN** user runs `byok config set-models openai-official --model gpt-4o --model gpt-4o-mini` and the profile exists
- **THEN** the profile's `models` list is set to `["gpt-4o", "gpt-4o-mini"]` in that order and the command succeeds

#### Scenario: Replace existing model list

- **WHEN** user runs `byok config set-models openai-official --model gpt-4o` and the profile's existing `models` list is `["gpt-4o", "gpt-4o-mini"]`
- **THEN** the profile's `models` list is set to `["gpt-4o"]`, fully replacing the prior list

#### Scenario: Interactive mode collects models until empty line

- **WHEN** user runs `byok config set-models openai-official` in a terminal and enters `gpt-4o`, `gpt-4o-mini`, then an empty line
- **THEN** the profile's `models` list is set to `["gpt-4o", "gpt-4o-mini"]`

#### Scenario: Empty model list rejected

- **WHEN** user runs `byok config set-models openai-official` and the resulting model list is empty
- **THEN** the command prints an error stating at least one model is required and exits with code 1 without modifying the config file

#### Scenario: Non-existent profile rejected

- **WHEN** user runs `byok config set-models nonexistent --model gpt-4o` and the config file contains profiles `openai-official` and `local-ollama`
- **THEN** the command prints an error listing `openai-official` and `local-ollama` as available profiles and exits with code 1 without modifying the config file

#### Scenario: Interactive mode rejected on non-tty stdin

- **WHEN** user pipes input into `byok config set-models openai-official` (stdin is not a terminal) and provides no `--model` flag
- **THEN** the command prints an error directing the user to use the `--model` flag and exits with code 1

<!-- @trace
source: add-model-selection
updated: 2026-07-07
code:
  - cmd/launch_codex_app.go
  - cmd/config.go
  - cmd/set_models.go
  - internal/config/config.go
  - internal/config/models_windows.go
  - internal/runner/pi.go
  - internal/config/models.go
  - internal/runner/runner.go
  - internal/runner/claude.go
  - cmd/launch.go
  - CLAUDE.md
  - AGENTS.md
  - cmd/launch_codex.go
  - cmd/launch_pi.go
  - README.md
  - cmd/launch_claude.go
  - internal/runner/codex.go
  - internal/config/models_unix.go
tests:
  - internal/runner/launch_integration_test.go
  - internal/runner/runner_test.go
  - cmd/launch_model_test.go
  - internal/config/models_test.go
  - cmd/set_models_test.go
  - internal/runner/codex_test.go
  - internal/runner/claude_test.go
  - internal/runner/codex_launch_test.go
  - internal/config/interactive_test.go
  - internal/runner/pi_test.go
  - internal/runner/codex_app_test.go
  - internal/config/config_test.go
  - cmd/config_test.go
-->