## MODIFIED Requirements

### Requirement: Multi-platform build via GitHub Actions matrix

The release workflow SHALL build the `byok` binary for the following target platforms using a GitHub Actions matrix strategy: `windows/amd64`, `linux/amd64`, `darwin/amd64`, `darwin/arm64`. Each build job SHALL inject the version string via ldflags, compile the main package located at `./cmd/byok`, and produce a compressed archive named `byok-<version>-<os>-<arch>.<ext>` where `<ext>` is `zip` for Windows and `tar.gz` for Linux and macOS.

#### Scenario: Build Windows binary

- **WHEN** the release workflow runs the `windows/amd64` matrix job
- **THEN** a `byok-<version>-windows-amd64.zip` archive is produced containing the `byok.exe` binary built from `./cmd/byok` with ldflags-injected version

#### Scenario: Build Linux binary

- **WHEN** the release workflow runs the `linux/amd64` matrix job
- **THEN** a `byok-<version>-linux-amd64.tar.gz` archive is produced containing the `byok` binary built from `./cmd/byok` with ldflags-injected version

#### Scenario: Build macOS arm64 binary

- **WHEN** the release workflow runs the `darwin/arm64` matrix job
- **THEN** a `byok-<version>-darwin-arm64.tar.gz` archive is produced containing the `byok` binary built from `./cmd/byok` with ldflags-injected version
