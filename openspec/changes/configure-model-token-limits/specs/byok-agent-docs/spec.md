## ADDED Requirements

### Requirement: Documentation explains model token limits

The `README.md` SHALL document the profile-level `default_model_limits`, scalar and mapping `models` forms, `context_window_tokens`, `max_output_tokens`, and the launch flags `--context-window-tokens` and `--max-output-tokens`. It SHALL state that each field resolves independently in CLI, selected-model, profile-default, unset order and that an unset result creates no byok token override.

The README SHALL include a target support matrix naming the exact Copilot, Codex, Codex App, Claude, and Pi mappings. It SHALL state that Codex and Codex App ignore maximum output with a warning, that Copilot maps the generic context field to maximum prompt tokens without subtraction, that Claude model names are passed unchanged, and that Pi derives response headroom from an explicit maximum output value.

The README SHALL explain that token settings declare or limit the capacity presented to the downstream target, do not expand the backing model's real capacity, and do not configure a cross-target compaction threshold. It SHALL include downgrade guidance for converting mapping model entries back to scalar names before using an older byok binary.

#### Scenario: README contains a complete YAML example

- **WHEN** a reader opens the configuration section
- **THEN** the README shows one profile with `default_model_limits`, one fully configured mapping model, one partially configured model, and explains legacy scalar compatibility

#### Scenario: README contains precedence and flags

- **WHEN** a reader opens the launch options section
- **THEN** the README names both token flags and the exact four-level per-field precedence

#### Scenario: README contains target support matrix

- **WHEN** a reader opens the token-limit target support section
- **THEN** the README lists all five targets, exact downstream keys, unsupported output warnings for Codex targets, and the Copilot prompt-token semantic difference

#### Scenario: README separates context from compaction

- **WHEN** a reader reviews limitations
- **THEN** the README states that Context Window configuration does not add a common compaction-threshold control and cannot exceed the backing model's capacity

### Requirement: AGENTS.md records token-limit contracts

The `AGENTS.md` configuration and development-convention sections SHALL document the new model data shape, profile defaults, launch flags, per-field precedence, target mappings, warning-only unsupported behavior, Pi temporary settings behavior, and removal of Copilot and Claude hard-coded defaults. The Spectra-managed block SHALL remain unchanged.

#### Scenario: AGENTS records config and CLI contracts

- **WHEN** a maintainer reads the configuration and launch sections
- **THEN** AGENTS.md describes the mapping model fields, `default_model_limits`, both launch flags, and their per-field precedence

#### Scenario: AGENTS records injection boundaries

- **WHEN** a maintainer reads the development conventions
- **THEN** AGENTS.md states that token overrides affect child processes or Pi temporary files only, unsupported settings warn and continue, and no user target config file is modified
