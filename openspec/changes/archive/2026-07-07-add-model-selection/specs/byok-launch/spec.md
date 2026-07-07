## MODIFIED Requirements

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

## ADDED Requirements

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