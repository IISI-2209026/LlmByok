## ADDED Requirements

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
