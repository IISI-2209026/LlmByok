# byok-claude-launch Specification

## Purpose

TBD - created by archiving change 'add-claude-launch'. Update Purpose after archive.

## Requirements

### Requirement: Launch Claude with BYOK profile

The `byok launch claude` command SHALL read the selected profile from the byok config file and start the `claude` executable as a child process with BYOK settings injected from the profile. The injection SHALL NOT write to `~/.claude/settings.json` or any other Claude Code configuration file. The API key, provider base URL, and model SHALL be carried to the `claude` child process via environment variables set only in the child process environment: `ANTHROPIC_BASE_URL` set to the profile `api_base`, `ANTHROPIC_API_KEY` set to the profile api key, and `ANTHROPIC_MODEL` set to the selected model appended with the suffix `[1m]`. When no `--profile` flag is provided, the default profile SHALL be used. The child process stdin, stdout, and stderr SHALL be transparently connected to the parent process.

#### Scenario: Launch with default profile

- **WHEN** user runs `byok launch claude` with a config file containing a default profile named `openai-official` whose `api_base` is `https://api.openai.com/v1`, `api_key` is `sk-xxxx`, and selected model is `gpt-4o`
- **THEN** the `claude` child process is started with `ANTHROPIC_BASE_URL=https://api.openai.com/v1`, `ANTHROPIC_API_KEY=sk-xxxx`, and `ANTHROPIC_MODEL=gpt-4o[1m]` in its environment and zero command-line arguments

#### Scenario: Override model with --model flag

- **WHEN** user runs `byok launch claude --model claude-sonnet-4-5`
- **THEN** the `claude` child process is started with `ANTHROPIC_MODEL=claude-sonnet-4-5[1m]`

#### Scenario: Select profile with --profile flag

- **WHEN** user runs `byok launch claude --profile local-ollama`
- **THEN** the `claude` child process is started using the `local-ollama` profile settings instead of the default profile


<!-- @trace
source: add-official-website
updated: 2026-07-09
code:
  - public/icons/pi.svg
  - public/script.js
  - GEMINI.md
  - public/icons/anthropic.svg
  - .spectra.yaml
  - internal/runner/claude.go
  - public/index.html
  - public/style.css
  - AGENTS.md
  - CLAUDE.md
  - public/icons/copilot.svg
  - public/icons/openai.svg
tests:
  - internal/runner/claude_test.go
  - cmd/launch_claude_test.go
-->

---
### Requirement: Parent process environment unchanged for claude

The `byok` parent process SHALL inject the `ANTHROPIC_BASE_URL`, `ANTHROPIC_API_KEY`, and `ANTHROPIC_MODEL` environment variables only into the `claude` child process environment. The parent process environment and the user shell environment SHALL NOT be modified before, during, or after the launch. The `~/.claude/settings.json` file SHALL NOT be created or modified by `byok`.

#### Scenario: Environment isolation

- **WHEN** user runs `byok launch claude` and the launch completes
- **THEN** the parent `byok` process environment variables remain identical to their values before the command ran and `~/.claude/settings.json` is not modified


<!-- @trace
source: add-claude-launch
updated: 2026-07-06
code:
  - AGENTS.md
  - README.md
  - cmd/launch_claude.go
  - cmd/root.go
  - cmd/launch.go
  - internal/runner/claude.go
tests:
  - cmd/launch_claude_test.go
  - cmd/launch_dispatch_test.go
  - cmd/launch_test.go
  - internal/runner/claude_test.go
-->

---
### Requirement: Claude executable presence check

Before launching, the `byok launch claude` command SHALL verify the `claude` executable is resolvable on PATH via `exec.LookPath`. When the executable is not found, the command SHALL print an error message instructing the user to install Claude Code and exit with code 1.

#### Scenario: Claude not installed

- **WHEN** user runs `byok launch claude` and `claude` is not on PATH
- **THEN** the command prints an error message mentioning Claude Code installation and exits with code 1


<!-- @trace
source: add-claude-launch
updated: 2026-07-06
code:
  - AGENTS.md
  - README.md
  - cmd/launch_claude.go
  - cmd/root.go
  - cmd/launch.go
  - internal/runner/claude.go
tests:
  - cmd/launch_claude_test.go
  - cmd/launch_dispatch_test.go
  - cmd/launch_test.go
  - internal/runner/claude_test.go
-->

---
### Requirement: Claude missing profile error

When the resolved profile (default or named via `--profile`) does not exist in the config file, the `byok launch claude` command SHALL print an error message listing available profile names and exit with code 1.

#### Scenario: Named profile missing

- **WHEN** user runs `byok launch claude --profile nonexistent` and the config file contains profiles `openai-official` and `local-ollama`
- **THEN** the command prints an error listing `openai-official` and `local-ollama` as available profiles and exits with code 1


<!-- @trace
source: add-claude-launch
updated: 2026-07-06
code:
  - AGENTS.md
  - README.md
  - cmd/launch_claude.go
  - cmd/root.go
  - cmd/launch.go
  - internal/runner/claude.go
tests:
  - cmd/launch_claude_test.go
  - cmd/launch_dispatch_test.go
  - cmd/launch_test.go
  - internal/runner/claude_test.go
-->

---
### Requirement: Claude missing config file error

When the config file does not exist, the `byok launch claude` command SHALL print an error message instructing the user to run `byok config add` first and exit with code 1.

#### Scenario: No config file

- **WHEN** user runs `byok launch claude` and `~/.byok/config.yaml` does not exist
- **THEN** the command prints an error suggesting `byok config add` and exits with code 1


<!-- @trace
source: add-claude-launch
updated: 2026-07-06
code:
  - AGENTS.md
  - README.md
  - cmd/launch_claude.go
  - cmd/root.go
  - cmd/launch.go
  - internal/runner/claude.go
tests:
  - cmd/launch_claude_test.go
  - cmd/launch_dispatch_test.go
  - cmd/launch_test.go
  - internal/runner/claude_test.go
-->

---
### Requirement: Claude provider validation

The `byok launch claude` command SHALL only accept profiles whose `provider` field is `openai` (empty `provider` defaults to `openai`). When a profile has any other provider value, the command SHALL print an error message stating that only the openai provider is supported and exit with code 1.

#### Scenario: Non-openai provider rejected

- **WHEN** user runs `byok launch claude --profile azure-prod` and the `azure-prod` profile has `provider: azure`
- **THEN** the command prints an error stating only the openai provider is supported and exits with code 1


<!-- @trace
source: add-claude-launch
updated: 2026-07-06
code:
  - AGENTS.md
  - README.md
  - cmd/launch_claude.go
  - cmd/root.go
  - cmd/launch.go
  - internal/runner/claude.go
tests:
  - cmd/launch_claude_test.go
  - cmd/launch_dispatch_test.go
  - cmd/launch_test.go
  - internal/runner/claude_test.go
-->

---
### Requirement: Claude YOLO mode flag

The `byok launch claude` command SHALL accept a `-y` / `--yolo` boolean flag. When the flag is set, the command SHALL append the string `--dangerously-skip-permissions` (the Claude Code permission-bypass flag equivalent to copilot/codex `--yolo`) to the `claude` executable arguments before any passthrough arguments.

#### Scenario: YOLO flag appends permission bypass

- **WHEN** user runs `byok launch claude --yolo`
- **THEN** the `claude` child process receives the argument `--dangerously-skip-permissions`

#### Scenario: Short form -y alias

- **WHEN** user runs `byok launch claude -y`
- **THEN** the `claude` child process receives the argument `--dangerously-skip-permissions`

#### Scenario: YOLO flag combined with passthrough

- **WHEN** user runs `byok launch claude -y -- --continue`
- **THEN** the `claude` child process receives the arguments `--dangerously-skip-permissions --continue` in that order


<!-- @trace
source: add-claude-launch
updated: 2026-07-06
code:
  - AGENTS.md
  - README.md
  - cmd/launch_claude.go
  - cmd/root.go
  - cmd/launch.go
  - internal/runner/claude.go
tests:
  - cmd/launch_claude_test.go
  - cmd/launch_dispatch_test.go
  - cmd/launch_test.go
  - internal/runner/claude_test.go
-->

---
### Requirement: Claude argument passthrough via double dash

The `byok launch claude` command SHALL accept a `--` separator followed by arbitrary arguments. All arguments after the `--` SHALL be forwarded verbatim to the `claude` executable as command-line arguments, without parsing or validation by `byok`.

#### Scenario: Single passthrough argument

- **WHEN** user runs `byok launch claude -- --continue`
- **THEN** the `claude` child process receives the argument `--continue`

#### Scenario: No passthrough arguments after dash

- **WHEN** user runs `byok launch claude --`
- **THEN** the `claude` child process receives zero passthrough arguments (yolo flag still applies if set)


<!-- @trace
source: add-claude-launch
updated: 2026-07-06
code:
  - AGENTS.md
  - README.md
  - cmd/launch_claude.go
  - cmd/root.go
  - cmd/launch.go
  - internal/runner/claude.go
tests:
  - cmd/launch_claude_test.go
  - cmd/launch_dispatch_test.go
  - cmd/launch_test.go
  - internal/runner/claude_test.go
-->

---
### Requirement: Claude launch documentation in README

The `README.md` SHALL document `claude` as a supported `byok launch` target alongside `copilot` and `codex`. The README SHALL include a Claude BYOK section in "運作原理" describing that `byok launch claude` injects `ANTHROPIC_BASE_URL`, `ANTHROPIC_API_KEY`, and `ANTHROPIC_MODEL` into the `claude` child process environment without writing `~/.claude/settings.json`. The README targets table, launch examples, prerequisites, official documentation links, and troubleshooting section SHALL be updated to include `claude`. Introductory text that describes `byok` as specific to Copilot SHALL be generalized to reflect that `byok` supports `copilot`, `codex`, and `claude`.

#### Scenario: Claude appears in targets table

- **WHEN** reader views the `byok launch <target>` Targets table in `README.md`
- **THEN** the table lists `copilot`, `codex`, and `claude` with descriptions

#### Scenario: Claude BYOK section in README

- **WHEN** reader views the "運作原理" section of `README.md`
- **THEN** a "Claude BYOK" subsection describes the three `ANTHROPIC_*` environment variables and states that `~/.claude/settings.json` is not modified

#### Scenario: Claude in official documentation links

- **WHEN** reader views the "官方文件" section of `README.md`
- **THEN** a link to the Claude Code model configuration documentation is listed alongside the Copilot and Codex BYOK documentation links

<!-- @trace
source: add-claude-launch
updated: 2026-07-06
code:
  - AGENTS.md
  - README.md
  - cmd/launch_claude.go
  - cmd/root.go
  - cmd/launch.go
  - internal/runner/claude.go
tests:
  - cmd/launch_claude_test.go
  - cmd/launch_dispatch_test.go
  - cmd/launch_test.go
  - internal/runner/claude_test.go
-->