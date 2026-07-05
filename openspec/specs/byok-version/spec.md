# byok-version Specification

## Purpose

TBD - created by archiving change 'add-version-and-release'. Update Purpose after archive.

## Requirements

### Requirement: Version variable injection via ldflags

The build system SHALL support injecting the version string into the `internal/version.Version` variable via Go ldflags (`-X`). When no version is injected at build time, the variable SHALL default to the string `dev`.

#### Scenario: Default version without ldflags

- **WHEN** the binary is built without ldflags injection
- **THEN** `byok --version` outputs `byok version dev`

#### Scenario: Injected version via ldflags

- **WHEN** the binary is built with `-ldflags "-X github.com/IISI-2209026/LlmByok/internal/version.Version=0.1.0"`
- **THEN** `byok --version` outputs `byok version 0.1.0`


<!-- @trace
source: add-version-and-release
updated: 2026-07-05
code:
  - cmd/launch.go
  - internal/runner/testdata/stub/main.go
  - main.go
  - .spectra.yaml
  - .github/workflows/release.yml
  - internal/runner/runner.go
  - Makefile
  - internal/version/version.go
  - README.md
tests:
  - cmd/launch_test.go
  - internal/runner/launch_integration_test.go
  - internal/version/version_test.go
-->

---
### Requirement: Version flag

The `byok` CLI SHALL provide a `--version` flag on the root command that prints the current version string in the format `byok version <Version>` and exits with code 0. The flag is provided by cobra's built-in `Version` field on the root command; no dedicated `version` subcommand SHALL be registered.

#### Scenario: Display version

- **WHEN** user runs `byok --version`
- **THEN** the command prints `byok version <current Version value>` to stdout and exits with code 0

<!-- @trace
source: add-version-and-release
updated: 2026-07-05
code:
  - cmd/root.go
  - main.go
  - internal/version/version.go
  - README.md
tests:
  - cmd/launch_test.go
  - internal/runner/launch_integration_test.go
  - internal/version/version_test.go
-->

---
### Requirement: Canonical base version source

The canonical base version of byok SHALL be the `Version` string literal in `internal/version/version.go`, formatted as a semantic version with no `v` prefix and no prerelease suffix (for example `0.1.0`). The default value when no version is injected at build time SHALL remain `dev`, but the committed literal in the repository SHALL be a concrete base version (initial `0.1.0`) that is the single source of truth read by the Makefile and the Release workflow. Any change to the base version SHALL be made by editing this literal (via the bump skill), not by adding a separate version file or relying on Git tags as the source.

#### Scenario: Committed literal is a concrete base version

- **WHEN** the repository is checked out at the default state
- **THEN** `internal/version/version.go` contains `var Version = "0.1.0"` (initial base), not `dev`

#### Scenario: Build without ldflags still reports dev fallback

- **WHEN** the binary is built without any ldflags injection
- **THEN** the in-memory default remains `dev` only when the literal is `dev`; once the literal is `0.1.0`, a no-ldflags build reports `0.1.0`


<!-- @trace
source: add-version-promotion-skill
updated: 2026-07-05
code:
  - .github/workflows/release.yml
  - internal/runner/codex.go
  - README.md
  - cmd/launch_codex.go
  - AGENTS.md
  - .github/workflows/pr-test.yml
  - cmd/launch.go
  - internal/version/version.go
  - LICENSE
  - .github/skills/byok-bump-version/SKILL.md
tests:
  - cmd/launch_dispatch_test.go
  - internal/version/version_test.go
  - internal/runner/codex_launch_test.go
  - internal/runner/codex_test.go
  - cmd/launch_codex_test.go
-->

---
### Requirement: Branch-specific binary version string format

The binary version string injected via ldflags SHALL differ by branch: on `main` it SHALL be exactly the base version (`<base>`); on `develop` it SHALL be `<base>-dev.<run_number>` where `<run_number>` is the GitHub Actions `github.run_number`. The `byok --version` output SHALL reflect this injected string.

#### Scenario: main binary version

- **WHEN** the Release workflow builds on `main` with base `0.1.0`
- **THEN** the binary reports `byok version 0.1.0`

#### Scenario: develop binary version

- **WHEN** the Release workflow builds on `develop` with base `0.1.0` and `github.run_number` `42`
- **THEN** the binary reports `byok version 0.1.0-dev.42`

<!-- @trace
source: add-version-promotion-skill
updated: 2026-07-05
code:
  - .github/workflows/release.yml
  - internal/runner/codex.go
  - README.md
  - cmd/launch_codex.go
  - AGENTS.md
  - .github/workflows/pr-test.yml
  - cmd/launch.go
  - internal/version/version.go
  - LICENSE
  - .github/skills/byok-bump-version/SKILL.md
tests:
  - cmd/launch_dispatch_test.go
  - internal/version/version_test.go
  - internal/runner/codex_launch_test.go
  - internal/runner/codex_test.go
  - cmd/launch_codex_test.go
-->