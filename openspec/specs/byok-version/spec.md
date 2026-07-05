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