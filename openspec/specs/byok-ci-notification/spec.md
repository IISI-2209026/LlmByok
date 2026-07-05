# byok-ci-notification Specification

## Purpose

TBD - created by archiving change 'add-codex-launch-and-pushover-tuning'. Update Purpose after archive.

## Requirements

### Requirement: Pushover notification priority and sound by result

GitHub Actions workflows that send Pushover notifications SHALL set the `priority` and `sound` parameters based on the overall job/result status. When the status is `success`, the notification SHALL use low-priority (`priority: '0'`) with a neutral or positive sound. When the status is `failure` (or any non-success status), the notification SHALL use high-priority (`priority: '1'`) with an alert sound. The emergency priority level (`2`) SHALL NOT be used.

#### Scenario: Successful release notification

- **WHEN** the Release workflow `notify` job runs and both `build` and `release` jobs succeeded
- **THEN** the Pushover notification is sent with `priority: '0'` and a neutral/positive sound

#### Scenario: Failed release notification

- **WHEN** the Release workflow `notify` job runs and at least one of `build` or `release` jobs did not succeed
- **THEN** the Pushover notification is sent with `priority: '1'` and an alert sound

#### Scenario: Successful PR test notification

- **WHEN** the PR Tests workflow `test` job succeeds
- **THEN** the Pushover notification is sent with `priority: '0'` and a neutral/positive sound

#### Scenario: Failed PR test notification

- **WHEN** the PR Tests workflow `test` job fails
- **THEN** the Pushover notification is sent with `priority: '1'` and an alert sound

<!-- @trace
source: add-codex-launch-and-pushover-tuning
updated: 2026-07-05
code:
  - .github/workflows/release.yml
  - .github/workflows/pr-test.yml
-->


<!-- @trace
source: add-codex-launch-and-pushover-tuning
updated: 2026-07-05
code:
  - .github/workflows/release.yml
  - cmd/launch_codex.go
  - internal/runner/codex.go
  - .github/workflows/pr-test.yml
  - .github/skills/byok-bump-version/SKILL.md
  - README.md
  - AGENTS.md
  - internal/version/version.go
  - cmd/launch.go
tests:
  - cmd/launch_codex_test.go
  - cmd/launch_dispatch_test.go
  - internal/version/version_test.go
  - internal/runner/codex_launch_test.go
  - internal/runner/codex_test.go
-->

---
### Requirement: Notification status derivation

The Release workflow `notify` job SHALL derive the overall status from `needs.build.result` and `needs.release.result`; the status SHALL be `success` only when both are `success`, otherwise `failure`. The PR Tests workflow SHALL derive the status from `job.status`. The derived status SHALL drive the `priority` and `sound` selection.

#### Scenario: Release status both success

- **WHEN** `needs.build.result` is `success` and `needs.release.result` is `success`
- **THEN** the derived overall status is `success`

#### Scenario: Release status build failure

- **WHEN** `needs.build.result` is `failure` and `needs.release.result` is `success`
- **THEN** the derived overall status is `failure`

<!-- @trace
source: add-codex-launch-and-pushover-tuning
updated: 2026-07-05
code:
  - .github/workflows/release.yml
  - .github/workflows/pr-test.yml
-->

<!-- @trace
source: add-codex-launch-and-pushover-tuning
updated: 2026-07-05
code:
  - .github/workflows/release.yml
  - cmd/launch_codex.go
  - internal/runner/codex.go
  - .github/workflows/pr-test.yml
  - .github/skills/byok-bump-version/SKILL.md
  - README.md
  - AGENTS.md
  - internal/version/version.go
  - cmd/launch.go
tests:
  - cmd/launch_codex_test.go
  - cmd/launch_dispatch_test.go
  - internal/version/version_test.go
  - internal/runner/codex_launch_test.go
  - internal/runner/codex_test.go
-->