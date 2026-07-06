# byok-release Specification

## Purpose

TBD - created by archiving change 'add-version-and-release'. Update Purpose after archive.

## Requirements

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


<!-- @trace
source: add-keychain-secret-storage
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
-->

---
### Requirement: GitHub Release creation on main branch push

The release workflow SHALL trigger on push to the `main` branch. After all matrix build jobs complete, the workflow SHALL create a GitHub Release tagged with the current version string and attach all platform archives to the release. The workflow SHALL require `contents: write` permission.

#### Scenario: Push to main triggers release

- **WHEN** code is pushed to the `main` branch
- **THEN** the release workflow builds all platform binaries and creates a GitHub Release with tag `<version>` and all platform archives attached

#### Scenario: Push to feature branch does not trigger release

- **WHEN** code is pushed to a branch other than `main`
- **THEN** the release workflow does not run

<!-- @trace
source: add-version-and-release
updated: 2026-07-05
code:
  - cmd/launch.go
  - internal/runner/testdata/stub/main.go
  - main.go
  - .spectra.yaml
  - .github/workflows/release.yml
  - cmd/version.go
  - internal/runner/runner.go
  - Makefile
  - internal/version/version.go
  - README.md
tests:
  - cmd/launch_test.go
  - internal/runner/launch_integration_test.go
  - cmd/version_test.go
  - internal/version/version_test.go
-->

---
### Requirement: Branch-specific release tag and prerelease flag

The Release workflow SHALL derive the Git tag and prerelease flag from the branch: on `main` the tag SHALL be `v<base>` and the release SHALL be marked as a stable release (`prerelease: false`); on `develop` the tag SHALL be `v<base>-dev.<run_number>` (where `<run_number>` is `github.run_number`) and the release SHALL be marked as a prerelease (`prerelease: true`). The `<run_number>` component SHALL ensure the develop tag is unique per workflow run so repeated pushes to develop never collide with an existing tag.

#### Scenario: main stable release tag

- **WHEN** the Release workflow runs on `main` with base `0.1.0`
- **THEN** it creates (or updates) a GitHub release with tag `v0.1.0` marked as a stable release

#### Scenario: develop prerelease tag is unique per run

- **WHEN** the Release workflow runs on `develop` with base `0.1.0` and `github.run_number` `42`
- **THEN** it creates a GitHub release with tag `v0.1.0-dev.42` marked as a prerelease
- **AND** a subsequent run with `github.run_number` `43` creates tag `v0.1.0-dev.43` without colliding

#### Scenario: develop push does not collide on repeated runs

- **WHEN** develop is pushed twice without a base version change
- **THEN** the two workflow runs produce two distinct tags differing by `run_number`, and neither release step fails due to an existing tag

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
### Requirement: Categorized changelog generated from commit history for GitHub Releases

The release workflow SHALL, before creating a GitHub Release, generate a Markdown changelog from the commit subjects between the most recent existing release tag and the current HEAD. The changelog SHALL categorize entries by conventional commit prefix into at least three sections: "新增功能" for commits whose subject begins with `feat:`, "優化功能" for commits whose subject begins with `refactor:` or `perf:`, and "修復功能" for commits whose subject begins with `fix:`. The generated changelog SHALL be used as the release body in place of GitHub's auto-generated release notes. When no previous release tag exists (first release), the changelog SHALL cover all commits reachable from HEAD. The categorization SHALL apply to both prerelease (develop branch) and stable (main branch) workflows.

#### Scenario: Stable release changelog categorizes commits since last tag

- **WHEN** the release workflow runs on push to `main` and the most recent existing tag is `v0.1.0` and commits since `v0.1.0` include `feat: add update command`, `fix: codex name field`, and `refactor: merge launch help`
- **THEN** the generated release body contains a "新增功能" section listing `feat: add update command`, a "修復功能" section listing `fix: codex name field`, and a "優化功能" section listing `refactor: merge launch help`

#### Scenario: Prerelease release changelog covers dev commits

- **WHEN** the release workflow runs on push to `develop` and the most recent existing tag is a dev tag `v0.1.0-dev.40` and commits since that tag include `feat: channel flag` and `docs: readme update`
- **THEN** the generated release body contains a "新增功能" section listing `feat: channel flag` and the `docs:` commit does not appear under 新增/優化/修復 sections

#### Scenario: First release has no previous tag

- **WHEN** the release workflow runs and no prior release tag exists in the repository
- **THEN** the changelog is generated from all commits reachable from HEAD without error and the release is created with that changelog as its body

<!-- @trace
source: add-self-update
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
-->