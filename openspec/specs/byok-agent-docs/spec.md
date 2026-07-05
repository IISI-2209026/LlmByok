# byok-agent-docs Specification

## Purpose

TBD - created by archiving change 'add-codex-launch-and-pushover-tuning'. Update Purpose after archive.

## Requirements

### Requirement: AGENTS.md documents project architecture

The `AGENTS.md` file SHALL contain a project architecture section, placed after the Spectra instructions block, that documents the authoritative project structure derived from the source: the language and toolchain (Go 1.26+, cobra CLI), the module path (`github.com/IISI-2209026/LlmByok`), the responsibility of each top-level package (`cmd` for cobra commands and dispatch, `internal/config` for YAML profile loading and storage, `internal/runner` for BYOK environment building and child-process launching, `internal/version` for version embedding), the entry point (`main.go`), and the user configuration file location (`~/.byok/config.yaml`).

#### Scenario: Architecture section present

- **WHEN** a contributor reads `AGENTS.md`
- **THEN** the file contains a section describing the Go module path, the cobra-based CLI entry, the four top-level packages and their responsibilities, and the config file location

#### Scenario: Architecture reflects current source

- **WHEN** the package layout, module path, or config file location changes
- **THEN** the same change that alters those facts also updates the architecture section of `AGENTS.md`

<!-- @trace
source: add-codex-launch-and-pushover-tuning
updated: 2026-07-05
code:
  - AGENTS.md
-->


<!-- @trace
source: add-codex-launch-and-pushover-tuning
updated: 2026-07-05
code:
  - .github/workflows/release.yml
  - cmd/launch_codex.go
  - internal/runner/codex.go
  - .github/workflows/pr-test.yml
  - .github/skills/byok-bump-version/SKILL.md
  - README.md
  - AGENTS.md
  - internal/version/version.go
  - cmd/launch.go
tests:
  - cmd/launch_codex_test.go
  - cmd/launch_dispatch_test.go
  - internal/version/version_test.go
  - internal/runner/codex_launch_test.go
  - internal/runner/codex_test.go
-->

---
### Requirement: AGENTS.md documents development conventions

The `AGENTS.md` file SHALL contain a development conventions section, placed after the architecture section, that documents the binding conventions: BYOK injection only affects the launched child process (the parent `byok` and shell environment are never modified), user configuration files (`~/.byok/config.yaml`, `~/.codex/config.toml`) are never written or modified by byok, profile resolution errors produce a user-facing message and exit code 1, the default provider type is `openai`, and tests run via `go test ./... -race`.

#### Scenario: Conventions section present

- **WHEN** a contributor reads `AGENTS.md`
- **THEN** the file contains a section stating the child-process-only BYOK injection rule, the no-write-to-user-config rule, the error-and-exit-1 profile resolution behavior, the default openai provider, and the test command

<!-- @trace
source: add-codex-launch-and-pushover-tuning
updated: 2026-07-05
code:
  - AGENTS.md
-->


<!-- @trace
source: add-codex-launch-and-pushover-tuning
updated: 2026-07-05
code:
  - .github/workflows/release.yml
  - cmd/launch_codex.go
  - internal/runner/codex.go
  - .github/workflows/pr-test.yml
  - .github/skills/byok-bump-version/SKILL.md
  - README.md
  - AGENTS.md
  - internal/version/version.go
  - cmd/launch.go
tests:
  - cmd/launch_codex_test.go
  - cmd/launch_dispatch_test.go
  - internal/version/version_test.go
  - internal/runner/codex_launch_test.go
  - internal/runner/codex_test.go
-->

---
### Requirement: AGENTS.md maintenance rule

The `AGENTS.md` file SHALL contain an explicit maintenance rule stating that any change which alters the package structure, the BYOK injection mechanism, the configuration file format, the CLI surface, or any documented development convention MUST update the corresponding `AGENTS.md` section within the same change. The Spectra instructions block (`<!-- SPECTRA:START -->` ... `<!-- SPECTRA:END -->`) is managed by the Spectra CLI and MUST NOT be edited by hand.

#### Scenario: Maintenance rule is explicit

- **WHEN** a contributor reads `AGENTS.md`
- **THEN** the file states that architecture/convention changes must be reflected in `AGENTS.md` in the same change, and that the SPECTRA block is CLI-managed and not hand-edited

<!-- @trace
source: add-codex-launch-and-pushover-tuning
updated: 2026-07-05
code:
  - AGENTS.md
-->

<!-- @trace
source: add-codex-launch-and-pushover-tuning
updated: 2026-07-05
code:
  - .github/workflows/release.yml
  - cmd/launch_codex.go
  - internal/runner/codex.go
  - .github/workflows/pr-test.yml
  - .github/skills/byok-bump-version/SKILL.md
  - README.md
  - AGENTS.md
  - internal/version/version.go
  - cmd/launch.go
tests:
  - cmd/launch_codex_test.go
  - cmd/launch_dispatch_test.go
  - internal/version/version_test.go
  - internal/runner/codex_launch_test.go
  - internal/runner/codex_test.go
-->