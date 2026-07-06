# byok-release Specification

## Purpose

TBD - created by archiving change 'add-version-and-release'. Update Purpose after archive.

## Requirements

### Requirement: Multi-platform build via GitHub Actions matrix

The release workflow SHALL build the `byok` binary for the following target platforms using a GitHub Actions matrix strategy: `windows/amd64`, `linux/amd64`, `darwin/amd64`, `darwin/arm64`. Each build job SHALL inject the version string via ldflags, compile the main package located at `./cmd/byok`, and produce a compressed archive named `byok-<version>-<os>-<arch>.<ext>` where `<ext>` is `zip` for Windows and `tar.gz` for Linux and macOS. The built binary file name inside the archive SHALL be platform-specific: `byok.exe` for Windows and `byok` for Linux and macOS.

#### Scenario: Build Windows binary

- **WHEN** the release workflow runs the `windows/amd64` matrix job
- **THEN** a `byok-<version>-windows-amd64.zip` archive is produced containing the `byok.exe` binary built from `./cmd/byok` with ldflags-injected version

#### Scenario: Build Linux binary

- **WHEN** the release workflow runs the `linux/amd64` matrix job
- **THEN** a `byok-<version>-linux-amd64.tar.gz` archive is produced containing the `byok` binary built from `./cmd/byok` with ldflags-injected version

#### Scenario: Build macOS arm64 binary

- **WHEN** the release workflow runs the `darwin/arm64` matrix job
- **THEN** a `byok-<version>-darwin-arm64.tar.gz` archive is produced containing the `byok` binary built from `./cmd/byok` with ldflags-injected version

#### Scenario: Windows archive member has exe extension

- **WHEN** the `windows/amd64` matrix job completes and the resulting zip is inspected
- **THEN** the sole binary member inside the zip SHALL be named `byok.exe` (not `byok`)

#### Scenario: Non-Windows archive member has no extension

- **WHEN** the `linux/amd64` or `darwin/*` matrix job completes and the resulting tar.gz is inspected
- **THEN** the sole binary member inside the archive SHALL be named `byok` (not `byok.exe`)


<!-- @trace
source: fix-windows-archive-exe-extension
updated: 2026-07-06
code:
  - internal/runner/testdata/stub/main.go
  - internal/runner/runner.go
  - README.md
  - cmd/launch.go
  - internal/runner/pi.go
  - .github/workflows/release.yml
  - AGENTS.md
  - cmd/launch_pi.go
tests:
  - cmd/launch_pi_test.go
  - internal/runner/pi_test.go
  - cmd/launch_dispatch_test.go
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

The Release workflow SHALL derive the Git tag and prerelease flag from the branch: on `main` the tag SHALL be `v<base>` and the release SHALL be marked as a stable release (`prerelease: false`); on `develop` the tag SHALL be `v<base>-dev.<run_number>` (where `<run_number>` is `github.run_number`) and the release SHALL be marked as a prerelease (`prerelease: true`). The `<run_number>` component SHALL ensure the develop tag is unique per workflow run so repeated pushes to develop never collide with an existing tag. The GitHub Release SHALL be created with `target_commitish` set to the commit that triggered the workflow (`github.sha`), so a push to `develop` produces a tag pointing at the develop commit and a push to `main` produces a tag pointing at the main commit; the workflow SHALL NOT rely on the GitHub release action default commitish (the repository default branch).

#### Scenario: main stable release tag

- **WHEN** the Release workflow runs on `main` with base `0.1.0`
- **THEN** it creates (or updates) a GitHub release with tag `v0.1.0` marked as a stable release and `target_commitish` equal to the `main` commit that triggered the run

#### Scenario: develop prerelease tag is unique per run

- **WHEN** the Release workflow runs on `develop` with base `0.1.0` and `github.run_number` `42`
- **THEN** it creates a GitHub release with tag `v0.1.0-dev.42` marked as a prerelease and `target_commitish` equal to the `develop` commit that triggered the run
- **AND** a subsequent run with `github.run_number` `43` creates tag `v0.1.0-dev.43` without colliding

#### Scenario: develop push does not collide on repeated runs

- **WHEN** develop is pushed twice without a base version change
- **THEN** the two workflow runs produce two distinct tags differing by `run_number`, and neither release step fails due to an existing tag

#### Scenario: develop tag lands on develop commit not main

- **WHEN** the Release workflow runs on a push to `develop` whose HEAD commit is `abc123` while the `main` branch HEAD is `def456`
- **THEN** the created release tag `v0.1.0-dev.<run_number>` points at commit `abc123` (the develop commit), not `def456` (the main commit)


<!-- @trace
source: fix-release-target-and-ai-changelog
updated: 2026-07-06
code:
  - cmd/launch.go
  - cmd/root.go
  - .github/workflows/release.yml
  - .agents/skills/spectra-analyze/SKILL.md
  - internal/config/config.go
  - internal/config/interactive.go
  - cmd/config.go
  - AGENTS.md
  - cmd/launch_claude.go
  - internal/runner/claude.go
  - .agents/skills/spectra-verify/SKILL.md
  - README.md
tests:
  - cmd/config_key_test.go
  - cmd/config_test.go
  - internal/config/interactive_test.go
  - cmd/launch_claude_test.go
  - cmd/launch_dispatch_test.go
  - internal/runner/claude_test.go
  - cmd/launch_test.go
-->

---
### Requirement: AI-generated categorized changelog for GitHub Releases

The release workflow SHALL, before creating a GitHub Release, generate a Markdown changelog by invoking a GitHub Models model (using the GitHub Models free tier) with a prompt that includes the changed byok source code and specification documents in the release range (the diff between the most recent existing release tag and `HEAD`, restricted to `cmd/`, `internal/`, and `openspec/specs/` paths). The model output SHALL be categorized into at least three sections: "新增功能" for new features, "優化功能" for refactors and performance changes, and "修復功能" for fixes. The generated changelog SHALL be used as the release body in place of GitHub's auto-generated release notes. When no previous release tag exists (first release), the changelog SHALL cover all changed files reachable from `HEAD`. The categorization SHALL apply to both prerelease (develop branch) and stable (main branch) workflows.

#### Scenario: Stable release AI changelog categorizes changed code and specs

- **WHEN** the release workflow runs on push to `main` and the most recent existing tag is `v0.1.0` and the changed files since `v0.1.0` include a new `cmd/launch_claude.go`, a modified `internal/runner/runner.go`, and a modified `openspec/specs/byok-launch/spec.md`
- **THEN** the model is invoked with the diff of those files and the generated release body contains a "新增功能" section describing the new claude launch capability, a "優化功能" or "修復功能" section as appropriate for the runner change, and the spec change is summarized in prose

#### Scenario: Prerelease AI changelog covers changed files since last dev tag

- **WHEN** the release workflow runs on push to `develop` and the most recent existing tag is a dev tag `v0.1.0-dev.40` and the changed files since that tag include `feat: channel flag` source changes
- **THEN** the model is invoked with the diff of changed `cmd/`, `internal/`, and `openspec/specs/` files and the generated release body contains a "新增功能" section describing those changes

#### Scenario: First release has no previous tag

- **WHEN** the release workflow runs and no prior release tag exists in the repository
- **THEN** the AI changelog is generated from all changed files reachable from `HEAD` without error and the release is created with that changelog as its body

#### Scenario: Model call failure falls back to commit-history changelog

- **WHEN** the GitHub Models call fails, times out, returns an empty body, or the model access configuration is unset
- **THEN** the release workflow SHALL fall back to generating the changelog from the commit subjects between the most recent existing release tag and `HEAD` using the conventional-commit prefix categorization (新增功能 / 優化功能 / 修復功能), and the release SHALL still be created with that fallback changelog as its body

#### Scenario: Only byok-relevant files are sent to the model

- **WHEN** the release range includes changes to `.github/workflows/release.yml`, `README.md`, and `internal/runner/runner.go`
- **THEN** the prompt sent to the model includes the diffs of `internal/runner/runner.go` and any `cmd/` or `openspec/specs/` changes, and workflow/README-only changes are summarized at a high level rather than sent verbatim

<!-- @trace
source: fix-release-target-and-ai-changelog
updated: 2026-07-06
code:
  - cmd/launch.go
  - cmd/root.go
  - .github/workflows/release.yml
  - .agents/skills/spectra-analyze/SKILL.md
  - internal/config/config.go
  - internal/config/interactive.go
  - cmd/config.go
  - AGENTS.md
  - cmd/launch_claude.go
  - internal/runner/claude.go
  - .agents/skills/spectra-verify/SKILL.md
  - README.md
tests:
  - cmd/config_key_test.go
  - cmd/config_test.go
  - internal/config/interactive_test.go
  - cmd/launch_claude_test.go
  - cmd/launch_dispatch_test.go
  - internal/runner/claude_test.go
  - cmd/launch_test.go
-->