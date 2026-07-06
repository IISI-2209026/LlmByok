## ADDED Requirements

### Requirement: README documents prebuilt binary installation via GitHub Releases

The README SHALL document an installation path that does not require a Go toolchain: downloading a prebuilt `byok` binary from the project's GitHub Releases page. The documentation SHALL describe selecting the asset matching the user's platform using the `byok-<version>-<os>-<arch>.<ext>` naming convention (where `<ext>` is `zip` for Windows and `tar.gz` for Linux and macOS), extracting the binary, placing it on `PATH`, and verifying with `byok --version`. This SHALL be presented alongside the existing `go install github.com/IISI-2209026/LlmByok@latest` path as a peer install method, and the README SHALL clearly state that the GitHub Releases install path is the recommended way to enable `byok update` self-updates.

#### Scenario: Reader without Go can install via release asset

- **WHEN** a reader without a Go toolchain opens README.md
- **THEN** they find step-by-step instructions to download, extract, and install a prebuilt `byok` binary from GitHub Releases for their platform without running `go build` or `go install`

#### Scenario: Update command documented

- **WHEN** a reader opens the README usage section
- **THEN** they find a `byok update` entry describing that it checks GitHub Releases for the same channel (dev/stable) as the running binary, downloads the matching platform asset, and replaces the executable, plus the `--check` flag for a no-mutate query
