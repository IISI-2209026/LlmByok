## ADDED Requirements

### Requirement: Launch Claude with an optional reasoning effort

The `byok launch claude` command SHALL set `CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1` and `CLAUDE_CODE_EFFORT_LEVEL=<level>` only in the Claude child process environment when a validated `--effort <level>` is provided. When `--effort` is omitted, the command SHALL not add either variable. The parent process environment and Claude configuration files SHALL remain unchanged.

#### Scenario: Claude effort environment injection

- **WHEN** the user runs `byok launch claude --effort xhigh`
- **THEN** the Claude child process environment SHALL contain `CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1` and `CLAUDE_CODE_EFFORT_LEVEL=xhigh`

#### Scenario: Claude no effort environment injection by default

- **WHEN** the user runs `byok launch claude` without `--effort`
- **THEN** the Claude child process environment SHALL not contain a byok-provided `CLAUDE_CODE_ALWAYS_ENABLE_EFFORT` or `CLAUDE_CODE_EFFORT_LEVEL` entry

### Requirement: Launch Claude with an optional subagent model

The `byok launch claude` command SHALL set `CLAUDE_CODE_SUBAGENT_MODEL=<model>` only in the Claude child process environment when `--sub-model <model>` is provided. The command SHALL pass the supplied model string without local validation. When `--sub-model` is omitted, the command SHALL not add a byok-provided `CLAUDE_CODE_SUBAGENT_MODEL` entry. The parent process environment and Claude configuration files SHALL remain unchanged.

#### Scenario: Claude subagent model environment injection

- **WHEN** the user runs `byok launch claude --sub-model claude-haiku-4-5`
- **THEN** the Claude child process environment SHALL contain `CLAUDE_CODE_SUBAGENT_MODEL=claude-haiku-4-5`

#### Scenario: Claude subagent model omission

- **WHEN** the user runs `byok launch claude` without `--sub-model`
- **THEN** the Claude child process environment SHALL not contain a byok-provided `CLAUDE_CODE_SUBAGENT_MODEL` entry
