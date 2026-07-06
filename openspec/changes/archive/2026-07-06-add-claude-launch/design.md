## Context

`byok launch` currently dispatches to `copilot` (env-var injection via `runner.Launch`) and `codex` (env-var + `--config` injection via `runner.LaunchCodex`). The user wants a third target, `claude`, that starts Claude Code with the same BYOK ergonomics: profile resolution, provider validation, executable check, `--model` override, `--yolo`, and `--` passthrough.

Claude Code reads its BYOK-relevant settings from environment variables: `ANTHROPIC_BASE_URL` (where requests are sent), `ANTHROPIC_API_KEY` (auth), and `ANTHROPIC_MODEL` (model alias or name). See the [model configuration docs](https://code.claude.com/docs/zh-TW/model-config). This maps cleanly onto byok's "inject into child process environment only, never write config files" principle. Claude Code's permission-bypass flag is `--dangerously-skip-permissions`, which is the functional equivalent of copilot/codex `--yolo`.

## Goals / Non-Goals

**Goals:**

- `byok launch claude` SHALL start the `claude` executable as a child process with `ANTHROPIC_BASE_URL`, `ANTHROPIC_API_KEY`, and `ANTHROPIC_MODEL` injected only into the child environment.
- The claude flow SHALL reuse the existing profile resolution, provider validation (`openai` only), executable presence check, key resolution (keychain → plaintext fallback), `--profile`/`--model`/`--config` flags, and `--` passthrough behavior, identical in shape to the copilot/codex flows.
- The `-y`/`--yolo` flag SHALL append `--dangerously-skip-permissions` to the `claude` child arguments.
- The parent process environment and `~/.claude/settings.json` SHALL remain unchanged.

**Non-Goals:**

- No support for non-`openai` providers (consistent with copilot/codex first version).
- No Claude-specific model pinning env vars (`ANTHROPIC_DEFAULT_OPUS_MODEL`, etc.); only base url, api key, and model are injected.
- No modification to `~/.claude/settings.json` or any Claude Code config file.
- No change to copilot or codex launch behavior.

## Decisions

### Decision: Inject Claude BYOK via environment variables only

Inject `ANTHROPIC_BASE_URL`, `ANTHROPIC_API_KEY`, and `ANTHROPIC_MODEL` into the child process environment (built from `os.Environ()` with those three keys filtered out and re-appended from the profile), mirroring `runner.BuildEnv` for copilot. No command-line `--config` overrides and no file writes — consistent with Claude Code's documented env-var configuration and byok's no-config-write rule.

**Alternative considered:** Pass `--model` as a CLI flag to `claude`. Rejected because env-var injection keeps the launch path uniform with copilot and avoids ordering concerns between the `--yolo`/passthrough args and a byok-injected `--model` flag.

### Decision: Map the byok --yolo flag to claude --dangerously-skip-permissions

The byok-level `-y`/`--yolo` flag is target-specific in the string it appends. For copilot and codex it appends `--yolo`; for claude it appends `--dangerously-skip-permissions`, the documented Claude Code permission-bypass flag. The dispatch builds the extra args per target so copilot/codex behavior is unchanged.

**Alternative considered:** Append the literal `--yolo` for claude. Rejected: `claude` has no `--yolo` flag and would error. Mapping to the equivalent permission-bypass flag preserves the user-facing "skip approvals" semantics.

### Decision: Add a runner.LaunchClaude function parallel to runner.LaunchCodex

Add `internal/runner/claude.go` with `BuildClaudeEnv(profile, modelOverride)` (returns the child env slice) and `LaunchClaude(profile, modelOverride, exePath, extraArgs, stdin, stdout, stderr)`. `cmd/launch_claude.go` adds `runLaunchClaude`, structurally identical to `runLaunchCodex` but resolving the `claude` binary and calling `runner.LaunchClaude`.

**Alternative considered:** Reuse `runner.Launch` (the copilot path) with a different env builder. Rejected: a dedicated function keeps the copilot env-key set and the claude env-key set independent and avoids coupling the two flows' env-var names.

### Decision: Generalize the extra-args builder for the yolo literal

`buildExtraArgs` currently hardcodes `--yolo`. Generalize it to accept the yolo literal string so the dispatch can pass `--yolo` for copilot/codex and `--dangerously-skip-permissions` for claude, while preserving identical copilot/codex output.

## Implementation Contract

- **Behavior (user-visible):** `byok launch claude` starts Claude Code with the selected profile's base URL, API key, and model injected via environment, with stdin/stdout/stderr transparently connected. `-y`/`--yolo` skips Claude Code permission prompts. `--` forwards args verbatim. Omitting the target, selecting a missing profile, missing config file, non-openai provider, or a missing `claude` binary each print a specific error and exit 1.
- **Interface / data shape:**
  - `runner.BuildClaudeEnv(profile *config.Profile, modelOverride string) []string` — child env slice with `ANTHROPIC_BASE_URL`/`ANTHROPIC_API_KEY`/`ANTHROPIC_MODEL` overridden.
  - `runner.LaunchClaude(profile *config.Profile, modelOverride, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer) error`.
  - `cmd.runLaunchClaude(cfgPath, profileName, model string, extraArgs []string, stdout, stderr io.Writer) error`.
  - The `byok launch` dispatch switch adds a `claude` case; the usage template and error messages list `copilot`, `codex`, `claude`.
- **Failure modes:** `claude` not on PATH → error + exit 1. Non-zero claude exit → propagated silently as `errExit` (matching copilot/codex). Other launch errors → printed to stderr + exit 1.
- **Acceptance criteria:**
  - `byok launch claude` with a default profile starts `claude` with the three env vars set in the child and the parent env unchanged (verifiable via `cmd/launch_claude_test.go` and `internal/runner/claude_test.go` asserting the env slice and command args).
  - `byok launch claude -y` produces child args starting with `--dangerously-skip-permissions`.
  - `byok launch gemini` prints an error listing `copilot`, `codex`, `claude` and exits 1 (verifiable via `cmd/launch_dispatch_test.go`).
  - `go test ./... -race` passes.
- **Scope boundaries:**
  - In scope: `cmd/launch_claude.go`, `internal/runner/claude.go`, `cmd/launch.go` dispatch + usage, and their tests.
  - Out of scope: copilot/codex launch behavior, config/keychain, version/release, other workflows.

## Risks / Trade-offs

- **Env-var name collision with user shell** → byok filters existing `ANTHROPIC_*` keys from the copied environment before re-appending, so only the profile values reach the child; the parent shell is never modified.
- **Claude Code flag drift** → if Claude Code renames `--dangerously-skip-permissions`, the yolo mapping breaks. Mitigation: the flag is documented and stable; surface the claude install hint on `LookPath` failure.
- **Model alias vs full name** → `ANTHROPIC_MODEL` accepts aliases or full names; byok forwards whatever the profile/override provides without validation, consistent with copilot/codex.
