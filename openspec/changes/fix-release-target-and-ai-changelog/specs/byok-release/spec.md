## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Categorized changelog generated from commit history for GitHub Releases

**Reason**: The commit-subject grep changelog is replaced by an AI-generated changelog produced via GitHub Models, which reads the changed byok source code and spec documents in the release range and writes a categorized Markdown changelog with richer semantic context.

**Migration**: The release workflow step that runs `git log --pretty=format:"%s"` and greps conventional-commit prefixes is replaced by a step that calls GitHub Models with the diff of changed `cmd/`, `internal/`, and `openspec/specs/` files. A fallback to the commit-history grep changelog is retained so a release never blocks when the model call fails or is unavailable.

## ADDED Requirements

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
