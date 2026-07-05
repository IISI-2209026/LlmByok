## ADDED Requirements

### Requirement: Target tool selection and dispatch

The `byok launch` command SHALL accept a target tool name as its first positional argument. The command SHALL dispatch to the `copilot` launch flow when the target is `copilot` and to the `codex` launch flow when the target is `codex`. When the target is omitted, the command SHALL print an error message stating that a target tool is required and exit with code 1. When the target is any value other than `copilot` or `codex`, the command SHALL print an error message listing the supported target tools and exit with code 1.

#### Scenario: Launch copilot dispatches to copilot flow

- **WHEN** user runs `byok launch copilot`
- **THEN** the command dispatches to the copilot launch flow and behaves identically to the existing copilot launch behavior

#### Scenario: Launch codex dispatches to codex flow

- **WHEN** user runs `byok launch codex`
- **THEN** the command dispatches to the codex launch flow

#### Scenario: Omitted target tool

- **WHEN** user runs `byok launch` with no positional argument
- **THEN** the command prints an error message stating a target tool is required and exits with code 1

#### Scenario: Unsupported target tool rejected

- **WHEN** user runs `byok launch gemini`
- **THEN** the command prints an error message listing `copilot` and `codex` as supported target tools and exits with code 1
