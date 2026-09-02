## MODIFIED Requirements

### Requirement: Config file location

The config file SHALL default to `~/.byok/config.yaml`. The `--config` flag SHALL allow overriding this path for all `byok config` subcommands. The config file SHALL be in YAML format and contain a `profiles` list plus a `default_profile` field.

Each profile SHALL carry a `models` list. A model item SHALL accept either a legacy scalar model identifier or a mapping containing a required non-empty `name` and optional positive integer `context_window_tokens` and `max_output_tokens`. The loader SHALL normalize a scalar item to a model whose name is the scalar value and whose token limits are unset. A profile SHALL accept an optional `default_model_limits` mapping with optional positive integer `context_window_tokens` and `max_output_tokens`.

When a config file containing a legacy `default_model` field is loaded and the profile's `models` list is empty, the loader SHALL migrate the `default_model` value into a single model item. When saving, the config writer SHALL emit model items in mapping form and SHALL NOT emit the legacy `default_model` field.

#### Scenario: Default config path

- **WHEN** user runs `byok config list` without `--config`
- **THEN** the config is read from `~/.byok/config.yaml`

#### Scenario: Legacy scalar models remain readable

- **WHEN** a profile contains `models: [gpt-4o, gpt-4o-mini]`
- **THEN** the loaded profile contains two models named `gpt-4o` and `gpt-4o-mini`, each with no individual token limits

#### Scenario: Mapping model limits load

- **WHEN** a profile model mapping contains `name: gpt-5.4`, `context_window_tokens: 1000000`, and `max_output_tokens: 128000`
- **THEN** the loaded model exposes the same name and both token values

#### Scenario: Profile default model limits load

- **WHEN** a profile contains `default_model_limits.context_window_tokens: 272000` and `default_model_limits.max_output_tokens: 16384`
- **THEN** the loaded profile exposes both default token values without applying them to the stored individual model objects

#### Scenario: Legacy default_model migrated on load

- **WHEN** a config file contains a profile `openai-official` with `default_model: gpt-4o` and no `models` list, and the user runs any `byok config` or `byok launch` command that loads the config
- **THEN** the loaded profile exposes one model named `gpt-4o` and no `default_model` is required for subsequent operations

#### Scenario: Saved config uses mapping models and omits legacy default_model

- **WHEN** the user runs a command that writes the config file after loading scalar models or a legacy profile
- **THEN** the written file contains model mappings with `name` fields and does not contain a `default_model` field

### Requirement: Set candidate models for a profile

The `byok config set-models <profile name>` command SHALL set the candidate model list for an existing profile, fully replacing the set and order of model names. The command SHALL be registered as a subcommand of `byok config`, not as a top-level command. The profile name SHALL be supplied as the first positional argument. The command SHALL accept a repeatable `--model` flag; each occurrence appends one candidate model identifier to the new list, in the order supplied.

When constructing the replacement list, the command SHALL preserve `context_window_tokens` and `max_output_tokens` from an existing model with the same name. A newly added name SHALL have no individual token limits. A removed name and its individual limits SHALL be removed. The command SHALL NOT modify `default_model_limits`.

The command SHALL support two modes:

1. **Parameter mode**: one or more `--model` flags supply the new candidate name list.
2. **Interactive mode**: when no `--model` flag is provided and stdin is a terminal, the command SHALL prompt the user to enter model identifiers one per line until an empty line is submitted, then SHALL store the collected non-empty entries as the new candidate name list.

When the resulting list is empty, the command SHALL print an error stating that at least one model is required and exit with code 1 without modifying the config file. When the named profile does not exist, the command SHALL print an error listing available profile names and exit with code 1 without modifying the file. When interactive mode is invoked and stdin is not a terminal, the command SHALL print an error directing the user to parameter mode and exit with code 1.

#### Scenario: Set multiple models via flags

- **WHEN** user runs `byok config set-models openai-official --model gpt-4o --model gpt-4o-mini` and the profile exists with neither name configured
- **THEN** the profile's model names are set to `gpt-4o` and `gpt-4o-mini` in that order with no individual limits, and the command succeeds

#### Scenario: Retained model keeps individual limits

- **WHEN** the existing `gpt-4o` model has `context_window_tokens: 128000` and the user runs `byok config set-models openai-official --model gpt-4o --model o3`
- **THEN** `gpt-4o` retains `context_window_tokens: 128000`, `o3` has no individual limits, and all omitted model names are removed

#### Scenario: Default model limits survive replacement

- **WHEN** a profile has `default_model_limits.max_output_tokens: 16384` and the user replaces its model names
- **THEN** `default_model_limits.max_output_tokens` remains `16384`

#### Scenario: Interactive mode collects models until empty line

- **WHEN** user runs `byok config set-models openai-official` in a terminal and enters `gpt-4o`, `gpt-4o-mini`, then an empty line
- **THEN** the profile's model names are set to `gpt-4o` and `gpt-4o-mini` in that order

#### Scenario: Empty model list rejected

- **WHEN** user runs `byok config set-models openai-official` and the resulting model list is empty
- **THEN** the command prints an error stating at least one model is required and exits with code 1 without modifying the config file

#### Scenario: Non-existent profile rejected

- **WHEN** user runs `byok config set-models nonexistent --model gpt-4o` and the config file contains profiles `openai-official` and `local-ollama`
- **THEN** the command prints an error listing `openai-official` and `local-ollama` as available profiles and exits with code 1 without modifying the config file

#### Scenario: Interactive mode rejected on non-tty stdin

- **WHEN** user pipes input into `byok config set-models openai-official` and provides no `--model` flag
- **THEN** the command prints an error directing the user to use the `--model` flag and exits with code 1

## ADDED Requirements

### Requirement: Model token configuration validation

The config loader SHALL reject an empty model name, duplicate model names within one profile, and any configured `context_window_tokens` or `max_output_tokens` value that is zero or negative. The error SHALL identify the profile and the invalid model or field. The loader SHALL NOT impose a cross-field ordering rule between context and output values.

#### Scenario: Duplicate model names rejected across scalar and mapping forms

- **WHEN** one profile contains scalar model `gpt-4o` and mapping model `name: gpt-4o`
- **THEN** config loading fails with an error identifying the profile and duplicate name `gpt-4o`

#### Scenario: Non-positive default limit rejected

- **WHEN** a profile contains `default_model_limits.context_window_tokens: 0`
- **THEN** config loading fails with an error identifying `context_window_tokens`

#### Scenario: Non-positive model limit rejected

- **WHEN** model `gpt-5.4` contains `max_output_tokens: -1`
- **THEN** config loading fails with an error identifying model `gpt-5.4` and `max_output_tokens`

#### Scenario: No cross-field ordering validation

- **WHEN** a model contains positive context and output values in any numeric order
- **THEN** config loading succeeds and target-specific behavior determines whether the downstream tool accepts the values

### Requirement: Model name views omit token metadata

The interactive model selector and `byok config list` SHALL display each model's `name` without rendering its token metadata. Their existing ordering, cancellation, terminal behavior, and table layout SHALL remain unchanged.

#### Scenario: Config list displays names from mapping models

- **WHEN** a profile contains mapping models named `gpt-5.4` and `gpt-5.4-mini`
- **THEN** `byok config list` displays `gpt-5.4, gpt-5.4-mini` in the models column

#### Scenario: Interactive selector displays mapping model names

- **WHEN** launch resolves a profile containing two mapping models
- **THEN** the interactive selector displays the two `name` values in config order without token fields
