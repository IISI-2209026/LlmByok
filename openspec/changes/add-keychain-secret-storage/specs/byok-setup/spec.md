## MODIFIED Requirements

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

---
### Requirement: Build via Makefile

The repository SHALL include a `Makefile` providing at minimum `build`, `run`, and `clean` targets so developers have a consistent build entry point across platforms. The `build` target SHALL compile the main package located at `./cmd/byok` and produce a `byok` (or `byok.exe` on Windows) binary in the `dist/` directory. The `run` target SHALL execute the main package via `go run ./cmd/byok`.

#### Scenario: Build via make

- **WHEN** a developer runs `make build` on a machine with Go installed
- **THEN** a `byok` (or `byok.exe` on Windows) binary is produced in the `dist/` directory

#### Scenario: Run via make

- **WHEN** a developer runs `make run ARGS="config list"`
- **THEN** `go run ./cmd/byok config list` is executed
