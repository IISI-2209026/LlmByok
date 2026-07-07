# byok-setup Specification

## Purpose

TBD - created by archiving change 'add-byok-cli'. Update Purpose after archive.

## Requirements

### Requirement: README.md with tool overview and Go environment setup guide

The repository SHALL include a `README.md` at the project root written for developers who have programming experience but have never written Go. The README SHALL open with a tool overview section that describes what `byok` does (a command-line tool that temporarily injects BYOK environment variables to launch Copilot CLI with the user's own OpenAI-compatible API key, without modifying the system environment), the problem it solves, and the key features (profile-based key management, one-command launch, transient environment injection that does not affect normal Copilot usage). The README SHALL then cover, in order: prerequisites, Go installation per platform (Windows, macOS, Linux), cloning the repository, building with `go build ./cmd/byok` and installing with `go install github.com/IISI-2209026/LlmByok/cmd/byok@latest`, running with `go run ./cmd/byok`, creating a config file, and example `byok` invocations. The README SHALL document OS keychain key management (`byok config set-key`, `byok config del-key`, `byok config import-keys`) including a note that Linux requires a secret-service daemon (e.g. gnome-keyring) for keychain storage and that plaintext `api_key` is used as fallback when the keychain is unavailable. A usage guide section SHALL explain every command (`launch copilot`, `launch codex`, `config add`, `config list`, `config set-key`, `config del-key`, `config import-keys`, `config remove`, `config set-default`) with its flags, a plain-language description of what each command does, and at least one concrete example invocation for each.

#### Scenario: Newcomer can build and run

- **WHEN** a developer with no prior Go experience follows the README from top to bottom on a clean machine
- **THEN** they are able to install Go, build the `byok` binary via `go build ./cmd/byok`, and run `byok config list` without consulting external documentation

#### Scenario: Bare go build outputs byok

- **WHEN** a developer runs `go build ./cmd/byok` in the repository root
- **THEN** the produced binary is named `byok` (or `byok.exe` on Windows), matching the release asset name

#### Scenario: Reader understands key management

- **WHEN** a reader opens the key management section of README.md
- **THEN** they find instructions for `byok config set-key`, `byok config del-key`, and `byok config import-keys`, plus a note that Linux requires a secret-service daemon and that plaintext `api_key` is the fallback


<!-- @trace
source: add-keychain-secret-storage
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
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

The repository SHALL include a `Makefile` providing at minimum `build`, `run`, and `clean` targets so developers have a consistent build entry point across platforms. The `build` target SHALL compile the main package located at `./cmd/byok` and produce a `byok` (or `byok.exe` on Windows) binary in the `dist/` directory. The `run` target SHALL execute the main package via `go run ./cmd/byok`.

#### Scenario: Build via make

- **WHEN** a developer runs `make build` on a machine with Go installed
- **THEN** a `byok` (or `byok.exe` on Windows) binary is produced in the `dist/` directory

#### Scenario: Run via make

- **WHEN** a developer runs `make run ARGS="config list"`
- **THEN** `go run ./cmd/byok config list` is executed


<!-- @trace
source: add-keychain-secret-storage
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
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

---
### Requirement: README documents prebuilt binary installation via GitHub Releases

The README SHALL document an installation path that does not require a Go toolchain: downloading a prebuilt `byok` binary from the project's GitHub Releases page. The documentation SHALL describe selecting the asset matching the user's platform using the `byok-<version>-<os>-<arch>.<ext>` naming convention (where `<ext>` is `zip` for Windows and `tar.gz` for Linux and macOS), extracting the binary, placing it on `PATH`, and verifying with `byok --version`. This SHALL be presented alongside the existing `go install github.com/IISI-2209026/LlmByok@latest` path as a peer install method, and the README SHALL clearly state that the GitHub Releases install path is the recommended way to enable `byok update` self-updates.

#### Scenario: Reader without Go can install via release asset

- **WHEN** a reader without a Go toolchain opens README.md
- **THEN** they find step-by-step instructions to download, extract, and install a prebuilt `byok` binary from GitHub Releases for their platform without running `go build` or `go install`

#### Scenario: Update command documented

- **WHEN** a reader opens the README usage section
- **THEN** they find a `byok update` entry describing that it checks GitHub Releases for the same channel (dev/stable) as the running binary, downloads the matching platform asset, and replaces the executable, plus the `--check` flag for a no-mutate query

<!-- @trace
source: add-self-update
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
-->