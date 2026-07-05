## ADDED Requirements

### Requirement: Canonical base version source

The canonical base version of byok SHALL be the `Version` string literal in `internal/version/version.go`, formatted as a semantic version with no `v` prefix and no prerelease suffix (for example `0.1.0`). The default value when no version is injected at build time SHALL remain `dev`, but the committed literal in the repository SHALL be a concrete base version (initial `0.1.0`) that is the single source of truth read by the Makefile and the Release workflow. Any change to the base version SHALL be made by editing this literal (via the bump skill), not by adding a separate version file or relying on Git tags as the source.

#### Scenario: Committed literal is a concrete base version

- **WHEN** the repository is checked out at the default state
- **THEN** `internal/version/version.go` contains `var Version = "0.1.0"` (initial base), not `dev`

#### Scenario: Build without ldflags still reports dev fallback

- **WHEN** the binary is built without any ldflags injection
- **THEN** the in-memory default remains `dev` only when the literal is `dev`; once the literal is `0.1.0`, a no-ldflags build reports `0.1.0`

### Requirement: Branch-specific binary version string format

The binary version string injected via ldflags SHALL differ by branch: on `main` it SHALL be exactly the base version (`<base>`); on `develop` it SHALL be `<base>-dev.<run_number>` where `<run_number>` is the GitHub Actions `github.run_number`. The `byok --version` output SHALL reflect this injected string.

#### Scenario: main binary version

- **WHEN** the Release workflow builds on `main` with base `0.1.0`
- **THEN** the binary reports `byok version 0.1.0`

#### Scenario: develop binary version

- **WHEN** the Release workflow builds on `develop` with base `0.1.0` and `github.run_number` `42`
- **THEN** the binary reports `byok version 0.1.0-dev.42`
