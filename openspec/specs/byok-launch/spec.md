# byok-launch Specification

## Purpose

TBD - created by archiving change 'add-byok-cli'. Update Purpose after archive.

## Requirements

### Requirement: Target tool selection and dispatch

The `byok launch` command SHALL accept a target tool name as its first positional argument. The command SHALL dispatch to the `copilot` launch flow when the target is `copilot`, to the `codex` launch flow when the target is `codex`, to the `codex-app` launch flow when the target is `codex-app`, to the `claude` launch flow when the target is `claude`, and to the `pi` launch flow when the target is `pi`. When the target is omitted, the command SHALL print an error message stating that a target tool is required and exit with code 1. When the target is any value other than `copilot`, `codex`, `codex-app`, `claude`, or `pi`, the command SHALL print an error message listing the supported target tools and exit with code 1.

#### Scenario: Launch copilot dispatches to copilot flow

- **WHEN** user runs `byok launch copilot`
- **THEN** the command dispatches to the copilot launch flow and behaves identically to the existing copilot launch behavior

#### Scenario: Launch codex dispatches to codex flow

- **WHEN** user runs `byok launch codex`
- **THEN** the command dispatches to the codex launch flow

#### Scenario: Launch codex-app dispatches to codex-app flow

- **WHEN** user runs `byok launch codex-app`
- **THEN** the command dispatches to the codex-app launch flow which starts `codex app` as a child process

#### Scenario: Launch claude dispatches to claude flow

- **WHEN** user runs `byok launch claude`
- **THEN** the command dispatches to the claude launch flow

#### Scenario: Launch pi dispatches to pi flow

- **WHEN** user runs `byok launch pi`
- **THEN** the command dispatches to the pi launch flow

#### Scenario: Omitted target tool

- **WHEN** user runs `byok launch` with no positional argument
- **THEN** the command prints an error message stating a target tool is required and exits with code 1

#### Scenario: Unsupported target tool rejected

- **WHEN** user runs `byok launch gemini`
- **THEN** the command prints an error message listing `copilot`, `codex`, `codex-app`, `claude`, and `pi` as supported target tools and exits with code 1


<!-- @trace
source: add-pi-launch
updated: 2026-07-06
code:
  - cmd/launch.go
  - .github/workflows/release.yml
  - internal/runner/runner.go
  - AGENTS.md
  - cmd/launch_pi.go
  - internal/runner/pi.go
  - internal/runner/testdata/stub/main.go
  - README.md
tests:
  - cmd/launch_dispatch_test.go
  - internal/runner/pi_test.go
  - cmd/launch_pi_test.go
-->

---
### Requirement: Launch Copilot with BYOK profile

The `byok launch copilot` command SHALL read the specified profile from the config file and start the `copilot` executable as a child process with BYOK environment variables injected from the profile settings. When no `--profile` flag is provided, the default profile SHALL be used. The child process stdin, stdout, and stderr SHALL be transparently connected to the parent process so the user interacts with Copilot normally. The command SHALL forward extra arguments (from the `--yolo`/`-y` flag and the `--` passthrough) to the `copilot` executable in the order: yolo flag arguments first, then passthrough arguments.

The model injected into the child process environment SHALL be resolved by the following rules, evaluated in order against the resolved profile's candidate `models` list:

1. When `--model` is provided, the command SHALL use the `--model` value and SHALL NOT present an interactive selection.
2. When `--model` is omitted and the `models` list contains exactly one entry, the command SHALL use that entry.
3. When `--model` is omitted and the `models` list contains more than one entry and stdin is a terminal, the command SHALL present an interactive up/down arrow selection listing every entry in `models` and SHALL use the entry the user selects. The selection SHALL place stdin into raw mode for the duration of the menu (disabling local echo and line buffering so arrow keys are delivered as ANSI sequences) and SHALL restore the original terminal state on exit. The selection SHALL enable virtual-terminal processing on stdout so ANSI cursor and reverse-video control sequences render correctly on Windows consoles (Unix terminals support them natively). The menu SHALL render every candidate as one line, marking the currently selected entry with a cursor glyph and reverse video, and SHALL redraw the menu in place as the selection moves. Down arrow SHALL advance the selection to the next entry, wrapping from the last to the first; up arrow SHALL move the selection to the previous entry, wrapping from the first to the last. The command SHALL confirm the selection on Enter and SHALL use the confirmed entry as the injected model. The command SHALL allow the user to cancel the selection with Ctrl-C or a lone Esc (an Esc not followed by `[`), in which case it SHALL clear the menu, print a cancellation notice, and exit with code 1 without launching the child process.
4. When `--model` is omitted and the `models` list contains more than one entry and stdin is not a terminal, the command SHALL print an error directing the user to specify `--model` and exit with code 1 without launching the child process.
5. When the `models` list is empty, the command SHALL print an error directing the user to run `byok config set-models <profile name>` and exit with code 1 without launching the child process.

#### Scenario: Launch with default profile and single candidate model

- **WHEN** user runs `byok launch copilot` with a config file containing a default profile named `openai-official` whose `api_base` is `https://api.openai.com/v1` and whose `models` list is `["gpt-4o"]`
- **THEN** the `copilot` child process is started with `COPILOT_PROVIDER_BASE_URL=https://api.openai.com/v1`, `COPILOT_PROVIDER_TYPE=openai`, `COPILOT_PROVIDER_API_KEY=<profile api_key>`, and `COPILOT_MODEL=gpt-4o` in its environment and zero command-line arguments

#### Scenario: Override model with --model flag

- **WHEN** user runs `byok launch copilot --model gemma4` using a profile whose `models` list is `["gpt-4o", "gpt-4o-mini"]`
- **THEN** the `copilot` child process is started with `COPILOT_MODEL=gemma4` overriding the profile candidates and zero command-line arguments, and no interactive selection is presented

#### Scenario: Interactive selection among multiple candidate models

- **WHEN** user runs `byok launch copilot` in a terminal using a profile whose `models` list is `["gpt-4o", "gpt-4o-mini"]` and the user selects `gpt-4o-mini` via the up/down arrow menu
- **THEN** the `copilot` child process is started with `COPILOT_MODEL=gpt-4o-mini` and zero command-line arguments

#### Scenario: Arrow keys change selection rather than falling back to the first model

- **WHEN** user runs `byok launch copilot` in a terminal using a profile whose `models` list is `["gpt-4o", "gpt-4o-mini"]`, presses the down arrow once, then presses Enter
- **THEN** the `copilot` child process is started with `COPILOT_MODEL=gpt-4o-mini` (the entry highlighted by the down arrow), not the first entry, confirming that arrow-key navigation drives the selected model

#### Scenario: Up arrow wraps from first to last candidate

- **WHEN** user runs `byok launch copilot` in a terminal using a profile whose `models` list is `["a", "b", "c"]`, presses the up arrow once (while the first entry is highlighted), then presses Enter
- **THEN** the `copilot` child process is started with `COPILOT_MODEL=c` (the last entry), confirming up-arrow wraps from the first to the last candidate

#### Scenario: Cancel interactive selection with Ctrl-C

- **WHEN** user runs `byok launch copilot` in a terminal using a profile whose `models` list is `["gpt-4o", "gpt-4o-mini"]` and presses Ctrl-C without confirming a selection
- **THEN** the command clears the menu, prints a cancellation notice, and exits with code 1 without starting the `copilot` child process

#### Scenario: Cancel interactive selection with Esc

- **WHEN** user runs `byok launch copilot` in a terminal using a profile whose `models` list is `["gpt-4o", "gpt-4o-mini"]` and presses a lone Esc (an Esc not followed by `[`) without confirming a selection
- **THEN** the command clears the menu, prints a cancellation notice, and exits with code 1 without starting the `copilot` child process

#### Scenario: Multiple candidate models rejected on non-tty stdin

- **WHEN** user runs `byok launch copilot` with stdin that is not a terminal using a profile whose `models` list is `["gpt-4o", "gpt-4o-mini"]` and no `--model` flag
- **THEN** the command prints an error directing the user to specify `--model` and exits with code 1 without starting the child process

#### Scenario: Empty candidate models rejected

- **WHEN** user runs `byok launch copilot` using a profile whose `models` list is empty and no `--model` flag
- **THEN** the command prints an error directing the user to run `byok config set-models <profile name>` and exits with code 1 without starting the child process

#### Scenario: Select profile with --profile flag

- **WHEN** user runs `byok launch copilot --profile local-ollama`
- **THEN** the `copilot` child process is started using the `local-ollama` profile settings instead of the default profile and zero command-line arguments

#### Scenario: Launch with no extra arguments

- **WHEN** user runs `byok launch copilot` without `-y`/`--yolo` or `--` passthrough
- **THEN** the `copilot` child process receives zero command-line arguments


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
### Requirement: Parent process environment unchanged

The `byok` parent process SHALL inject BYOK environment variables only into the `copilot` child process environment. The parent process environment and the user shell environment SHALL NOT be modified before, during, or after the launch.

#### Scenario: Environment isolation

- **WHEN** user runs `byok launch copilot` and the launch completes
- **THEN** the parent `byok` process environment variables remain identical to their values before the command ran


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
### Requirement: Config file path override

The `--config` flag SHALL allow the user to specify an alternate config file path. When omitted, the default path `~/.byok/config.yaml` SHALL be used.

#### Scenario: Custom config path

- **WHEN** user runs `byok launch copilot --config /tmp/my-config.yaml`
- **THEN** the config is read from `/tmp/my-config.yaml` instead of the default path


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
### Requirement: Copilot executable presence check

Before launching, the `byok launch copilot` command SHALL verify the `copilot` executable is resolvable on PATH via `exec.LookPath`. When the executable is not found, the command SHALL print an error message instructing the user to install Copilot CLI and exit with code 1.

#### Scenario: Copilot not installed

- **WHEN** user runs `byok launch copilot` and `copilot` is not on PATH
- **THEN** the command prints an error message mentioning Copilot CLI installation and exits with code 1


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
### Requirement: Missing profile error

When the resolved profile (default or named via `--profile`) does not exist in the config file, the `byok launch copilot` command SHALL print an error message listing available profile names and exit with code 1.

#### Scenario: Named profile missing

- **WHEN** user runs `byok launch copilot --profile nonexistent` and the config file contains profiles `openai-official` and `local-ollama`
- **THEN** the command prints an error listing `openai-official` and `local-ollama` as available profiles and exits with code 1


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
### Requirement: Missing config file error

When the config file does not exist, the `byok launch copilot` command SHALL print an error message instructing the user to run `byok config add` first and exit with code 1.

#### Scenario: No config file

- **WHEN** user runs `byok launch copilot` and `~/.byok/config.yaml` does not exist
- **THEN** the command prints an error suggesting `byok config add` and exits with code 1


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
### Requirement: Provider validation

The `byok launch copilot` command SHALL only accept profiles whose `provider` field is `openai`. When a profile has any other provider value, the command SHALL print an error message stating that the first version only supports the openai provider and exit with code 1.

#### Scenario: Non-openai provider rejected

- **WHEN** user runs `byok launch copilot --profile azure-prod` and the `azure-prod` profile has `provider: azure`
- **THEN** the command prints an error stating only the openai provider is supported and exits with code 1

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
### Requirement: YOLO mode flag

The `byok launch copilot` command SHALL accept a `-y` / `--yolo` boolean flag. When the flag is set, the command SHALL append the string `--yolo` to the copilot executable arguments before any passthrough arguments.

#### Scenario: YOLO flag appends --yolo

- **WHEN** user runs `byok launch copilot --yolo`
- **THEN** the `copilot` child process receives the argument `--yolo`

#### Scenario: Short form -y alias

- **WHEN** user runs `byok launch copilot -y`
- **THEN** the `copilot` child process receives the argument `--yolo`

#### Scenario: YOLO flag combined with passthrough

- **WHEN** user runs `byok launch copilot -y -- --continue`
- **THEN** the `copilot` child process receives the arguments `--yolo --continue` in that order


<!-- @trace
source: add-launch-passthrough-yolo
updated: 2026-07-05
code:
  - internal/runner/runner.go
  - README.md
  - cmd/launch.go
  - .spectra.yaml
  - internal/runner/testdata/stub/main.go
tests:
  - cmd/launch_test.go
  - internal/runner/launch_integration_test.go
-->

---
### Requirement: Argument passthrough via double dash

The `byok launch copilot` command SHALL accept a `--` separator followed by arbitrary arguments. All arguments after the `--` SHALL be forwarded verbatim to the `copilot` executable as command-line arguments, without parsing or validation by `byok`.

#### Scenario: Single passthrough argument

- **WHEN** user runs `byok launch copilot -- --continue`
- **THEN** the `copilot` child process receives the argument `--continue`

#### Scenario: Multiple passthrough arguments

- **WHEN** user runs `byok launch copilot -- --continue --model x`
- **THEN** the `copilot` child process receives the arguments `--continue --model x` in that order

#### Scenario: No passthrough arguments after dash

- **WHEN** user runs `byok launch copilot --`
- **THEN** the `copilot` child process receives zero command-line arguments from the passthrough (yolo flag still applies if set)

<!-- @trace
source: add-launch-passthrough-yolo
updated: 2026-07-05
code:
  - internal/runner/runner.go
  - README.md
  - cmd/launch.go
  - .spectra.yaml
  - internal/runner/testdata/stub/main.go
tests:
  - cmd/launch_test.go
  - internal/runner/launch_integration_test.go
-->

---
### Requirement: Model resolution shared across launch targets

The model resolution rules defined for `byok launch copilot` SHALL apply identically to every launch target (`copilot`, `codex`, `codex-app`, `claude`, `pi`). For each target, when `--model` is omitted the command SHALL resolve the model from the resolved profile's candidate `models` list using the same single-model, interactive-selection, non-tty-rejection, and empty-list-rejection rules, and SHALL inject the resolved model into that target's model environment variable. When `--model` is provided, the command SHALL inject the `--model` value for every target without presenting an interactive selection.

#### Scenario: Single candidate model used for codex

- **WHEN** user runs `byok launch codex` using a profile whose `models` list is `["gpt-4o"]`
- **THEN** the `codex` child process is started with the target's model environment variable set to `gpt-4o` and no interactive selection is presented

#### Scenario: Interactive selection for claude

- **WHEN** user runs `byok launch claude` in a terminal using a profile whose `models` list is `["claude-sonnet-4-5", "claude-haiku-4-5"]` and the user selects `claude-haiku-4-5`
- **THEN** the `claude` child process is started with the target's model environment variable set to `claude-haiku-4-5`

#### Scenario: --model overrides candidates for pi

- **WHEN** user runs `byok launch pi --model gpt-4o` using a profile whose `models` list is `["qwen3", "llama3"]`
- **THEN** the `pi` child process is started with the target's model environment variable set to `gpt-4o` and no interactive selection is presented

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