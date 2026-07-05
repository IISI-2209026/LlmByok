## MODIFIED Requirements

### Requirement: GitHub Release creation on main and develop branches

The release workflow SHALL trigger on push to the `main` branch and the `develop` branch. After all matrix build jobs complete, the workflow SHALL create a GitHub Release and attach all platform archives to the release. The workflow SHALL require `contents: write` permission. The release tag, version suffix, and prerelease flag SHALL be determined by the pushed branch:

- `main` branch: tag `v<version>`, release is NOT a prerelease
- `develop` branch: tag `v<version>-dev`, release IS a prerelease, and the version string injected into binaries via ldflags SHALL be `<version>-dev`

Branches other than `main` and `develop` SHALL NOT trigger the release workflow.

#### Scenario: Push to main triggers stable release

- **WHEN** code is pushed to the `main` branch
- **THEN** the release workflow builds all platform binaries with ldflags version `<version>` and creates a GitHub Release with tag `v<version>`, all platform archives attached, and prerelease set to false

#### Scenario: Push to develop triggers dev pre-release

- **WHEN** code is pushed to the `develop` branch
- **THEN** the release workflow builds all platform binaries with ldflags version `<version>-dev` and creates a GitHub Release with tag `v<version>-dev`, all platform archives named `byok-<version>-dev-<os>-<arch>.<ext>` attached, and prerelease set to true

##### Example: develop push with base version 0.1.0

- **GIVEN** the version declared in `internal/version/version.go` is `0.1.0`
- **WHEN** code is pushed to the `develop` branch
- **THEN** the binaries are built with ldflags version `0.1.0-dev`, the GitHub Release tag is `v0.1.0-dev`, archives are named `byok-0.1.0-dev-windows-amd64.zip` (and equivalents per platform), and the release is marked as a prerelease

#### Scenario: Push to a feature branch does not trigger release

- **WHEN** code is pushed to a branch other than `main` or `develop` (for example a feature branch)
- **THEN** the release workflow does not run
