## ADDED Requirements

### Requirement: Print a masked equivalent target command

The system SHALL provide a `--dry-run` flag on `byok launch <target>`. With this flag, the system SHALL resolve the config file, selected profile, provider, model, optional effort, optional sub-model, yolo mode, and passthrough arguments, then print an equivalent shell command to stdout and exit without starting the target executable. The system SHALL validate optional effort with the same target-specific rules as a normal launch. The system SHALL NOT resolve an API key from the keychain or config file, SHALL NOT include an actual API key in output, and SHALL NOT call `exec.LookPath` for the target executable. Every API-key position in the generated command SHALL contain a correctly quoted `***` placeholder.

#### Scenario: Dry run does not start target or resolve API key

- **WHEN** the user runs `byok launch codex --model gpt-5 --dry-run` for a profile with no available API key and without a `codex` executable on PATH
- **THEN** the system SHALL print a masked Codex command containing the selected model and BYOK configuration overrides, SHALL NOT report a missing key or executable error, and SHALL NOT start a child process

#### Scenario: Dry run still rejects invalid effort

- **WHEN** the user runs `byok launch claude --effort none --dry-run`
- **THEN** the system SHALL print an error naming `claude`, `none`, and the valid Claude levels, exit with code 1, and SHALL NOT print a command or start a child process

### Requirement: Render platform-specific equivalent commands

The system SHALL render dry-run output as PowerShell syntax on Windows and POSIX shell syntax on non-Windows platforms. For Copilot, Codex, Codex App, and Claude, the output SHALL include the target command, target-specific arguments, and environment assignments required by the normal launch mapping. For pi, the output SHALL be a complete shell fragment that creates a unique temporary directory, writes a `models.json` containing the profile API base and a masked API key placeholder, invokes pi with `PI_CODING_AGENT_DIR` set to that directory and the resolved arguments, and removes the temporary directory on completion. The output SHALL preserve yolo and passthrough argument order.

#### Scenario: Windows renders a PowerShell Claude command

- **WHEN** the user runs `byok launch claude --model claude-sonnet-4-5 --sub-model claude-haiku-4-5 --dry-run` on Windows
- **THEN** stdout SHALL contain PowerShell environment assignments for `ANTHROPIC_BASE_URL`, `ANTHROPIC_API_KEY='***'`, `ANTHROPIC_MODEL`, and `CLAUDE_CODE_SUBAGENT_MODEL`, followed by a `claude` command

#### Scenario: Unix renders a POSIX Codex App command

- **WHEN** the user runs `byok launch codex-app --model gpt-5 --effort high --dry-run` on a non-Windows platform
- **THEN** stdout SHALL contain a POSIX environment assignment for `BYOK_CODEX_API_KEY='***'` followed by `codex app`, the existing BYOK `--config` pairs, and `model_reasoning_effort="high"`

#### Scenario: Pi dry run includes temporary configuration lifecycle

- **WHEN** the user runs `byok launch pi --model gpt-5 --effort high --dry-run`
- **THEN** stdout SHALL contain commands that create a temporary directory, write masked `models.json`, set `PI_CODING_AGENT_DIR`, run `pi --model gpt-5 --thinking high`, and remove the temporary directory
