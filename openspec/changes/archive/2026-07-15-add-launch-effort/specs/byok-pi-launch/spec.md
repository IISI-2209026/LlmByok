## ADDED Requirements

### Requirement: Launch Pi with an optional thinking level

The `byok launch pi` command SHALL add `--thinking <level>` to the pi child process only when a validated `--effort <level>` is provided. The child process arguments SHALL place `--model <model>` first, then `--thinking <level>`, then yolo and passthrough arguments. When `--effort` is omitted, the command SHALL not add `--thinking`. The temporary `PI_CODING_AGENT_DIR` and models.json mechanism SHALL remain unchanged.

#### Scenario: Pi thinking argument injection

- **WHEN** the user runs `byok launch pi --model gpt-5 --effort high`
- **THEN** the pi child process arguments SHALL begin with `--model`, `gpt-5`, `--thinking`, and `high` in that order

#### Scenario: Pi omits thinking argument by default

- **WHEN** the user runs `byok launch pi` without `--effort`
- **THEN** the pi child process SHALL receive no `--thinking` argument
