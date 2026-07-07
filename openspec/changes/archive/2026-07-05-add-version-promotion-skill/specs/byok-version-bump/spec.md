## ADDED Requirements

### Requirement: Version bump skill computes next base version

The `byok-bump-version` skill SHALL read the current base version from the `Version` string literal in `internal/version/version.go`, parse it as a semantic version, and compute the next base version according to a bump level. The default bump level SHALL be `patch` (increment the third segment: `0.1.0` -> `0.1.1`). The skill SHALL accept `minor` (increment the second segment and reset patch to `0`: `0.1.0` -> `0.2.0`) and `major` (increment the first segment and reset minor and patch to `0`: `0.1.0` -> `1.0.0`) as alternatives. If the literal cannot be parsed as a semantic version, the skill SHALL abort and print an error without modifying the file.

#### Scenario: Default patch bump

- **WHEN** the skill runs with no bump level and the current base is `0.1.0`
- **THEN** the next base version is `0.1.1`

#### Scenario: Minor bump

- **WHEN** the skill runs with the `minor` level and the current base is `0.1.0`
- **THEN** the next base version is `0.2.0`

#### Scenario: Major bump

- **WHEN** the skill runs with the `major` level and the current base is `0.1.0`
- **THEN** the next base version is `1.0.0`

#### Scenario: Unparseable base aborts

- **WHEN** the `Version` literal is `dev` (not a semantic version) and the skill runs
- **THEN** the skill aborts and prints an error stating the base version cannot be parsed, and `internal/version/version.go` is not modified

### Requirement: Bump skill edits version.go, commits, and pushes to develop

The skill SHALL edit the `Version` literal in `internal/version/version.go` to the computed next base version, stage only that file, create a commit with the message `chore: bump version to <next>`, and push to `origin develop`. The skill SHALL NOT create a Git tag (tags are produced by the Release workflow after the push). The skill SHALL NOT push to or commit on any branch other than `develop`.

#### Scenario: Edit and commit

- **WHEN** the skill runs with current base `0.1.0` and default patch level on the develop branch
- **THEN** `internal/version/version.go` contains `var Version = "0.1.1"` and a commit `chore: bump version to 0.1.1` is created

#### Scenario: Push to origin develop

- **WHEN** the commit is created on develop
- **THEN** the skill runs `git push origin develop` so the next develop push triggers a Release workflow run with the new base

### Requirement: Bump skill guards against running on main

The skill SHALL detect the current Git branch before modifying files. If the current branch is `main`, the skill SHALL abort and print a message directing the user to run the skill on the `develop` branch, without modifying `internal/version/version.go` or creating any commit.

#### Scenario: Abort on main

- **WHEN** the skill runs while the current branch is `main`
- **THEN** the skill prints an error directing the user to switch to develop and does not modify any file or create any commit

### Requirement: Bump skill push failure handling

If `git push origin develop` fails (for example, because the remote is ahead), the skill SHALL print the push error and advise running `git pull --rebase origin develop` before retrying. The skill SHALL NOT force-push (`--force`) to develop.

#### Scenario: Remote ahead push failure

- **WHEN** `git push origin develop` fails because the remote branch is ahead
- **THEN** the skill prints the push error and advises `git pull --rebase origin develop`, and does not force-push

### Requirement: Version bump mechanism documented in AGENTS.md and README

The version promotion mechanism SHALL be documented in a dedicated section of `AGENTS.md` (the canonical base source, the develop `<base>-dev.<run_number>` and main `<base>` formats, and the promotion flow: merge develop to main to trigger a stable release, then run the bump skill on develop to advance the base). The same mechanism SHALL be documented in a `README.md` section describing how to use the `byok-bump-version` skill and the resulting release tags.

#### Scenario: AGENTS.md documents the mechanism

- **WHEN** a contributor reads `AGENTS.md`
- **THEN** it contains a version mechanism section describing the canonical source, the branch-specific version formats, and the promotion flow

#### Scenario: README documents the skill usage

- **WHEN** a user reads `README.md`
- **THEN** it contains a section describing the develop/main release tags and how to run the `byok-bump-version` skill to advance the base version
