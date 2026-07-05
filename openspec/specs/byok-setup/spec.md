# byok-setup Specification

## Purpose

TBD - created by archiving change 'add-byok-cli'. Update Purpose after archive.

## Requirements

### Requirement: README.md with tool overview and Go environment setup guide

The repository SHALL include a `README.md` at the project root written for developers who have programming experience but have never written Go. The README SHALL open with a tool overview section that describes what `byok` does (a command-line tool that temporarily injects BYOK environment variables to launch Copilot CLI with the user's own OpenAI-compatible API key, without modifying the system environment), the problem it solves, and the key features (profile-based key management, one-command launch, transient environment injection that does not affect normal Copilot usage). The README SHALL then cover, in order: prerequisites, Go installation per platform (Windows, macOS, Linux), cloning the repository, building with `go build` and `go install`, running with `go run main.go`, creating a config file, and example `byok` invocations. A usage guide section SHALL explain every command (`launch copilot`, `config add`, `config list`, `config remove`, `config set-default`) with its flags, a plain-language description of what each command does, and at least one concrete example invocation for each.

#### Scenario: Newcomer can build and run

- **WHEN** a developer with no prior Go experience follows the README from top to bottom on a clean machine
- **THEN** they are able to install Go, build the `byok` binary, and run `byok config list` without consulting external documentation

#### Scenario: Reader understands tool purpose and usage

- **WHEN** a reader opens README.md
- **THEN** the first section explains the tool's purpose (BYOK launch for Copilot CLI with transient environment injection) and a usage guide section documents every supported command with flags and at least one example invocation each


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
### Requirement: Go module manifest

The repository SHALL include a `go.mod` file at the project root declaring the module path (e.g. `github.com/IISI-2209026/LlmByok`) and the Go version used. Dependencies (Cobra, yaml.v3) SHALL be declared in `go.mod` and pinned in `go.sum`.

#### Scenario: Reproducible build

- **WHEN** a developer runs `go build` after cloning the repository
- **THEN** the build succeeds using only the dependencies declared in `go.mod` and `go.sum`


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
### Requirement: Build via Makefile

The repository SHALL include a `Makefile` providing at minimum `build`, `run`, and `clean` targets so developers have a consistent build entry point across platforms.

#### Scenario: Build via make

- **WHEN** a developer runs `make build` on a machine with Go installed
- **THEN** a `byok` (or `byok.exe` on Windows) binary is produced in the repository root or a `dist/` directory


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
### Requirement: Example config in README

The README SHALL include a copy-pasteable example config file in YAML showing two profiles (one remote OpenAI-compatible endpoint with an API key, one local no-auth endpoint) so the reader understands the config schema without reading source code.

#### Scenario: Example config present

- **WHEN** a reader opens README.md
- **THEN** they find a YAML code block containing a `profiles` list with at least two entries and a `default_profile` field

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