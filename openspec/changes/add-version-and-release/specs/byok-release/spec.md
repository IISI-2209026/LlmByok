## ADDED Requirements

### Requirement: Multi-platform build via GitHub Actions matrix

The release workflow SHALL build the `byok` binary for the following target platforms using a GitHub Actions matrix strategy: `windows/amd64`, `linux/amd64`, `darwin/amd64`, `darwin/arm64`. Each build job SHALL inject the version string via ldflags and produce a compressed archive named `byok-<version>-<os>-<arch>.<ext>` where `<ext>` is `zip` for Windows and `tar.gz` for Linux and macOS.

#### Scenario: Build Windows binary

- **WHEN** the release workflow runs the `windows/amd64` matrix job
- **THEN** a `byok-<version>-windows-amd64.zip` archive is produced containing the `byok.exe` binary with ldflags-injected version

#### Scenario: Build Linux binary

- **WHEN** the release workflow runs the `linux/amd64` matrix job
- **THEN** a `byok-<version>-linux-amd64.tar.gz` archive is produced containing the `byok` binary with ldflags-injected version

#### Scenario: Build macOS arm64 binary

- **WHEN** the release workflow runs the `darwin/arm64` matrix job
- **THEN** a `byok-<version>-darwin-arm64.tar.gz` archive is produced containing the `byok` binary with ldflags-injected version

### Requirement: GitHub Release creation on main branch push

The release workflow SHALL trigger on push to the `main` branch. After all matrix build jobs complete, the workflow SHALL create a GitHub Release tagged with the current version string and attach all platform archives to the release. The workflow SHALL require `contents: write` permission.

#### Scenario: Push to main triggers release

- **WHEN** code is pushed to the `main` branch
- **THEN** the release workflow builds all platform binaries and creates a GitHub Release with tag `<version>` and all platform archives attached

#### Scenario: Push to feature branch does not trigger release

- **WHEN** code is pushed to a branch other than `main`
- **THEN** the release workflow does not run
