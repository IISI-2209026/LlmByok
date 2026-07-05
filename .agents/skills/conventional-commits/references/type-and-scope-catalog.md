# Type and Scope Catalog

Deep reference for the `conventional-commits` skill. Read this when you need the
full rationale for a type, scope-naming guidance for this project, or the
SemVer-bump decision table.

## Table of Contents

1. Type Catalog
2. Scope-Naming Guidance for LlmByok
3. SemVer-Bump Decision Table
4. Angular vs Conventional Commits vs commitlint config-conventional
5. Footers in Detail

---

## 1. Type Catalog

Each entry: definition, examples, common mistakes, Angular/commitlint notes.

### feat

- **Definition:** A new feature for the user or library consumer. Introduces a
  new capability that didn't exist before.
- **SemVer:** MINOR (when `!` absent).
- **Examples:**
  - `feat(runner): inject BYOK env vars into child process`
  - `feat(version): print build commit and date in --version`
- **Common mistake:** using `feat` for an internal refactor that exposes no new
  user-facing capability. Use `refactor` for that.

### fix

- **Definition:** A bug fix — corrects existing behaviour that was wrong.
- **SemVer:** PATCH.
- **Examples:**
  - `fix(config): detect missing file via errors.Is on Windows`
  - `fix(runner): preserve parent env when overriding COPILOT_*`
- **Common mistake:** using `fix` for a new guardrail that wasn't previously
  broken. If the previous behaviour was correct, it's not a `fix`.

### build

- **Definition:** Changes to the build system or external dependencies that
  don't touch production source or tests. gulp, npm, make, docker, vcpkg, etc.
- **SemVer:** none.
- **Examples:**
  - `build: bump go to 1.22 in go.mod`
  - `build(docker): switch base image to alpine 3.19`
- **Note:** changing `go.mod`/`go.sum` for a code change is part of that code
  change's commit, not a separate `build:` commit.

### chore

- **Definition:** Maintenance tasks that don't affect `src` or tests — tooling
  config, repository metadata, CI runners config that isn't covered by `ci`.
- **SemVer:** none.
- **Examples:**
  - `chore: enable tdd and parallel_tasks in .spectra.yaml`
  - `chore: add .editorconfig`
- **Common mistake:** using `chore` for source refactors. Use `refactor`.

### ci

- **Definition:** Changes to CI configuration files and scripts — GitHub
  Actions workflows, GitLab CI, Jenkinsfile, Travis config.
- **SemVer:** none.
- **Examples:**
  - `ci(release): trigger workflow on develop branch with -dev suffix`
  - `ci: add golangci-lint action`
- **Note:** `ci:` is a specialization of `chore:` for CI specifically. Prefer
  `ci:` when the change is purely CI config.

### docs

- **Definition:** Documentation-only changes — README, doc comments, spec
  markdown, OpenAPI descriptions.
- **SemVer:** none.
- **Examples:**
  - `docs(readme): document BYOK environment variables`
  - `docs: fix typo in release workflow description`
- **Note:** the Angular convention allows omitting the body for `docs:`.

### perf

- **Definition:** A code change that improves performance. Should be backed by a
  benchmark or measurement.
- **SemVer:** none (unless breaking).
- **Examples:**
  - `perf(parser): avoid allocating slice for each token`
- **Common mistake:** using `perf` for a refactor that happens to be faster but
  wasn't measured. Use `refactor` unless you have numbers.

### refactor

- **Definition:** A code change that neither fixes a bug nor adds a feature —
  restructuring for clarity, extracting helpers, renaming.
- **SemVer:** none.
- **Examples:**
  - `refactor(config): extract profile selection into helper`
- **Common mistake:** using `refactor` for a bug fix that happened to require
  restructuring. If behaviour was wrong and is now correct, it's `fix`.

### revert

- **Definition:** Reverts a previous commit. See SKILL.md "Revert Format".
- **SemVer:** undoes the prior commit's bump.
- **Examples:**
  - `revert: feat(parser): add array parsing`
- **Body MUST contain** `This reverts commit <SHA>.` and a reason.

### style

- **Definition:** Formatting, whitespace, semi-colons, import ordering — no
  production code change.
- **SemVer:** none.
- **Examples:**
  - `style: run gofmt on config package`
- **Note:** `gofmt`/`gofumpt` runs as part of other commits should not get a
  separate `style:` commit; fold them in. Reserve `style:` for pure formatting
  sweeps.

### test

- **Definition:** Adding missing tests or correcting existing tests. No
  production code change.
- **SemVer:** none.
- **Examples:**
  - `test(config): add table-driven tests for profile selection`
- **Note:** tests added alongside the feature they cover belong in that
  feature's `feat:` commit. Reserve `test:` for standalone test additions or
  fixes.

---

## 2. Scope-Naming Guidance for LlmByok

Pick the scope from the list of Go packages and top-level concerns in this
repo. Keep it a single short token.

| Scope | Use for changes in... |
|-------|-----------------------|
| `config` | `internal/config` — profile loading, env var mapping |
| `runner` | `internal/runner` — child-process spawning and env injection |
| `cmd` | `cmd/` — CLI entrypoint and flag wiring |
| `version` | `internal/version` or version-info output |
| `release` | `.github/workflows/release.yml` and release tooling |
| `ci` | other CI workflows (lint, test) — but the type is usually `ci` already, so prefer `ci:` without a scope or `ci(lint):` |
| `spec` | `openspec/` specs and change artifacts |
| `skills` | `.agents/skills/` and `.github/skills/` skill definitions |
| `docs` | top-level docs (README, CONTRIBUTING) — but the type is usually `docs` already |

When a change touches multiple packages, either split the commit or pick the
most affected package and omit the scope if uncertain.

---

## 3. SemVer-Bump Decision Table

| Commit shape | Version bump | Why |
|--------------|--------------|-----|
| `feat: ...` | MINOR | new backward-compatible capability |
| `fix: ...` | PATCH | backward-compatible bug fix |
| `feat!: ...` | MAJOR | breaking new feature |
| `fix!: ...` | MAJOR | breaking bug fix |
| `feat: ...` + `BREAKING CHANGE:` footer | MAJOR | footer overrides |
| `feat: ...` + `DEPRECATED:` footer | MINOR | deprecation is not a break |
| `chore/docs/style/test/refactor/build/ci/perf: ...` | none | no user-facing API change |
| `chore!: ...` (breaking tooling change) | none for the library, MAJOR only if the tooling change breaks downstream consumers | rare |
| `revert: feat: ...` | undoes the MINOR | reverts to prior state |

A `BREAKING CHANGE` footer always wins: any type with a `BREAKING CHANGE:`
footer is a MAJOR bump, including `chore!:` (rare but valid).

---

## 4. Angular vs Conventional Commits vs commitlint config-conventional

| Aspect | Conventional Commits 1.0.0 | Angular | commitlint config-conventional |
|--------|----------------------------|---------|-------------------------------|
| Mandated types | `feat`, `fix` only | `build, ci, docs, feat, fix, perf, refactor, test` | Angular + `chore, revert, style` |
| Scope | optional, free-form | optional, must be a noun | optional, free-form |
| Subject case | unspecified | lowercase, no capital | lowercase, no capital |
| Subject tense | unspecified | imperative, present | imperative, present |
| Subject period | unspecified | none | none |
| Subject length | unspecified | ≤72 chars (Git practice) | ≤72 chars (header-max-length) |
| Body | optional, free-form | mandatory except `docs:`, imperative, wrap 100 | mandatory except `docs:`, wrap 100 |
| Breaking change | `!` or `BREAKING CHANGE:` footer | `BREAKING CHANGE:` footer only | `!` or `BREAKING CHANGE:` |
| Revert | not specified | `revert:` prefix + body | `revert:` prefix + body |
| Footer `Closes #N` | allowed | allowed | allowed |

This skill adopts the **strictest intersection**: Angular's type list +
config-conventional's `chore/revert/style`, Angular's subject rules, and the
Conventional Commits `!` notation. This is what most real-world tooling
(`semantic-release`, `standard-version`, `release-please`) accepts.

---

## 5. Footers in Detail

### BREAKING CHANGE

```
BREAKING CHANGE: <one-line summary>

<paragraph 1 of migration detail>

<paragraph 2 if needed>
```

- Token is uppercase with a space: `BREAKING CHANGE` (not `BREAKING_CHANGE`).
- `BREAKING-CHANGE` is accepted as a synonym in the parser.
- May be paired with `!` in the header — then the footer carries the detail.
- The first line after the colon is a short summary; subsequent paragraphs
  (separated by blank lines) are the migration guide.

### DEPRECATED

```
DEPRECATED: <what is deprecated>

<deprecation description>

<recommended update path>

Closes #123
```

### Issue/PR references

- `Closes #123` — GitHub closes the issue on merge to the default branch.
- `Fixes #123`, `Resolves #123` — synonyms for `Closes`.
- `Refs #123` — reference without closing.
- Multiple: `Closes #123, #124` or one per line.

### Review and co-author trailers

- `Reviewed-by: Jane Doe <jane@example.com>`
- `Co-authored-by: Pat R <pat@example.com>`
- `Signed-off-by: Pat R <pat@example.com>`

These use `-` instead of a space in the token (git trailer convention), except
`BREAKING CHANGE` / `DEPRECATED` which keep the space.