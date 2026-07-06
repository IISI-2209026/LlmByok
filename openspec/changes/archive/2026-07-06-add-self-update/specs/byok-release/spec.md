## ADDED Requirements

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
