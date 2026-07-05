---
name: conventional-commits
description: >
  Draft and write Git commit messages that follow the Conventional Commits 1.0.0
  specification and the Angular/Google commit-message guidelines (the de facto
  industry standard used by commitlint config-conventional, semantic-release,
  and most large open-source projects). Use this skill whenever the user asks to
  "commit", "git commit", "write a commit message", "幫我 commit", "寫 commit
  message", "提交", "做一個 commit", or asks how a commit message should be
  formatted. Also use when the user asks about Conventional Commits, semantic
  versioning from commits, changelog generation, or wants to fix/amend a commit
  message that does not follow the convention. Covers: the header/body/footer
  structure, the full type list (feat/fix/build/chore/ci/docs/perf/refactor/
  revert/style/test), scope rules, subject rules (imperative mood, no
  capitalization, no trailing period, ≤72 chars), body rules (explain WHY,
  wrapped at 72), footer/trailer rules (BREAKING CHANGE, Closes/Refs/Reviewed-by,
  DEPRECATED), the `!` breaking-change notation, revert format, and the SemVer
  correlation (feat→MINOR, fix→PATCH, BREAKING CHANGE→MAJOR). Do NOT use for
  Spectra change commits — those use the `spectra-commit` skill which already
  produces a `spectra(<change>):` prefix. Do NOT use for PR titles/bodies — use
  the `github-create-pr` skill for those.
---

# Conventional Commits

## Overview

This skill produces Git commit messages that conform to **Conventional Commits
1.0.0** (the official spec, conventionalcommits.org) layered with the
**Angular/Google commit-message guidelines** — the most widely adopted
implementation, used by `@commitlint/config-conventional`, `semantic-release`,
and most large open-source projects. Following it lets tooling auto-generate
CHANGELOGs and derive semantic-version bumps from commit history.

A Conventional Commit looks like:

```
<type>[optional scope][!]: <description>

[optional body]

[optional footer(s)]
```

## Quick Reference

| Element | Rule |
|---------|------|
| Type | One of the fixed nouns below. `feat` and `fix` are mandated by the spec; the rest come from the Angular convention. |
| Scope | Optional noun in parentheses naming the affected module/package, e.g. `(parser)`. |
| `!` | Placed immediately before the `:` to flag a breaking change, e.g. `feat(api)!: ...`. |
| Description | Imperative, present tense, lowercase first letter, no trailing period, ≤72 chars. |
| Body | One blank line after the description. Explain the **WHY** (motivation, before/after). Imperative mood. Wrap at 72 chars. |
| Footer | One blank line after the body. `BREAKING CHANGE:`, `DEPRECATED:`, `Closes #N`, `Refs #N`, `Reviewed-by: X`. |
| SemVer | `feat` → MINOR, `fix` → PATCH, `BREAKING CHANGE` / `!` → MAJOR. |

---

## The Type List

The spec mandates `feat` and `fix`; the Angular convention and
`config-conventional` add the rest. Use exactly these types:

| Type | When to use it | SemVer impact |
|------|----------------|---------------|
| `feat` | A new feature for the user/library | MINOR |
| `fix` | A bug fix | PATCH |
| `build` | Build system or external dependencies (gulp, npm, make, docker) | none |
| `chore` | Maintenance tasks that don't affect src or tests (tooling, config) | none |
| `ci` | CI configuration files and scripts (GitHub Actions, GitLab CI) | none |
| `docs` | Documentation only changes | none |
| `perf` | A code change that improves performance | none |
| `refactor` | A code change that neither fixes a bug nor adds a feature | none |
| `revert` | Reverts a previous commit (see Revert format below) | undoes prior |
| `style` | Formatting, whitespace, semi-colons; no production code change | none |
| `test` | Adding missing tests or correcting existing tests | none |

**Choosing a type when more than one fits:** the spec says to go back and make
multiple commits. Part of the value of Conventional Commits is driving smaller,
focused commits. If a change is genuinely both, prefer the higher-impact type
(`feat` > `fix` > `perf` > `refactor` > the rest).

---

## Scope

A scope is an optional noun in parentheses naming the part of the codebase
affected: `feat(parser): add array parsing`.

- Use the package/module name as a reader of the generated changelog would
  expect it. For this project that means a Go package path segment
  (`config`, `runner`, `cmd`, `version`) or a workflow/skill name.
- Omit the scope for cross-cutting changes (e.g. `test: add missing unit tests`,
  `docs: fix typo in README`).
- Keep the scope a single short token; don't chain scopes.

---

## Subject (the description line)

- **Imperative, present tense**: "add" not "added" or "adds". Think "If applied,
  this commit will ___" — the subject should complete that sentence.
- **No capitalization** of the first letter.
- **No trailing period** (`.`).
- **≤72 characters** total for the header line (type+scope+description). This is
  the git/coreutils and GitHub display convention; longer headers truncate in
  `git log --oneline` and most UIs.
- Be specific and concrete: "prevent racing of requests" beats "fix bug".

---

## Body

- Begin **one blank line** after the description.
- Explain the **motivation** for the change — the WHY, not the WHAT (the diff
  already shows what changed). Contrast before/after behavior when helpful.
- Use the **imperative, present tense**, same as the subject.
- **Wrap lines at 72 characters** (the conventional limit; some teams use 80 or
  100 — match the surrounding history if it has a consistent width).
- May be multiple paragraphs separated by blank lines.
- For non-trivial changes the body is expected; for `docs:`-only or trivial
  one-line changes it can be omitted.

---

## Footer

Begin **one blank line** after the body. Footers follow the git trailer
convention: a token, then `:<space>` or `<space>#`, then a value.

### Breaking changes

Two equivalent ways to flag a breaking change:

1. `!` immediately before the `:` in the header — `feat(api)!: drop Node 6
   support`.
2. A `BREAKING CHANGE:` footer (uppercase, with the space) — followed by a blank
   line and a detailed description + migration instructions.

If `!` is used, the `BREAKING CHANGE:` footer may be omitted and the description
used to explain the break. If both are used, the footer carries the detail.
`BREAKING-CHANGE` is accepted as a synonym for `BREAKING CHANGE` in the footer
token.

### Deprecation

```
DEPRECATED: <what is deprecated>

<deprecation description + recommended update path>

Closes #123
```

### Issue/PR references and review trailers

- `Closes #123` — closes the referenced issue when merged.
- `Refs #123` — references without closing.
- `Reviewed-by: Jane Doe <jane@example.com>` — review attribution.
- `Co-authored-by: Name <email>` — GitHub co-author attribution.

Footer tokens use `-` in place of whitespace (e.g. `Reviewed-by`), except
`BREAKING CHANGE` / `DEPRECATED` which keep the space.

---

## Revert Format

A commit that reverts a previous commit begins with `revert:` followed by the
header of the reverted commit:

```
revert: feat(parser): add array parsing

This reverts commit abc1234.

Reason: the array parser broke quoted-string handling; a fix will be
re-landed in a follow-up.
```

The body MUST contain `This reverts commit <SHA>.` and a clear reason.

---

## SemVer Correlation

This is why the convention pairs with `semantic-release`:

- `feat:` → MINOR bump (new backward-compatible feature).
- `fix:` → PATCH bump (backward-compatible bug fix).
- `feat!:` / `BREAKING CHANGE:` → MAJOR bump.
- All other types (`docs`, `chore`, `refactor`, etc.) → no version bump, unless
  they carry a `BREAKING CHANGE`.

---

## Examples

### Good — feature with scope and body

```
feat(runner): inject BYOK env vars into child process

Copy the parent environment and override the four COPILOT_*
variables before spawning copilot, so the user's BYOK credentials
flow through without leaking into the shell history.

Closes #42
```

### Good — bug fix with body explaining why

```
fix(config): detect missing file via errors.Is on Windows

os.IsNotExist returns false for an error wrapped with %w on
Windows, so a missing config file was reported as a parse error.
Switch to errors.Is(err, os.ErrNotExist) to detect the sentinel
across platforms.
```

### Good — breaking change with `!` and footer

```
feat(api)!: require base-url and api-key together

BREAKING CHANGE: previously base-url could be set without
api-key and the launch would fail at runtime with a confusing
error. Both must now be provided together; validate at config
load and fail fast with a clear message.
```

### Good — chore, no body needed

```
chore: enable tdd and parallel_tasks in .spectra.yaml
```

### Good — ci

```
ci(release): trigger workflow on develop branch with -dev suffix
```

### Bad — wrong tense, capitalized, period, no type

```
Fixed a bug in the parser.
```

Fixed: `fix(parser): handle multiple spaces in quoted strings`

### Bad — type missing, too vague

```
updated stuff
```

Fixed: `refactor(config): extract profile selection into helper`

### Bad — capitalization + trailing period on subject

```
Feat: Add Polish language.
```

Fixed: `feat(lang): add Polish language`

---

## Workflow (when asked to commit or write a commit message)

1. **Inspect the change.** Run `git status --porcelain` and
   `git diff --staged` (or `git diff` for unstaged). For multiple files, also
   `git diff --staged --stat` to see the scope. Read the diff to understand
   intent, not just file names.
2. **Pick the type** from the table above. If the change spans multiple types,
   suggest splitting into multiple commits (the spec explicitly recommends this).
3. **Pick the scope** (optional) — the affected package/module/workflow. Omit
   for cross-cutting changes.
4. **Draft the subject** in imperative present tense, lowercase, no period,
   ≤72 chars. Fill in "If applied, this commit will ___".
5. **Draft the body** explaining the WHY (motivation, before/after). Wrap at 72.
   Omit only for trivial `docs:`/`chore:` one-liners.
6. **Add footers** if relevant: `BREAKING CHANGE:` (or `!` in the header),
   `Closes #N`, `Refs #N`, `DEPRECATED:`.
7. **Locale:** This project's `.spectra.yaml` sets `locale: tw`, so write the
   **body in Traditional Chinese** to match the rest of the codebase docs.
   The **type, scope, and subject stay in English** (they are machine-parsed
   tokens and the convention is English). Example:
   ```
   fix(config): 以 errors.Is 偵測缺檔跨平台

   Windows 下 os.IsNotExist 對 %w 包裹錯誤回傳 false，導致缺檔被
   回報為格式錯誤。改用 errors.Is(err, os.ErrNotExist) 偵測 sentinel。
   ```
8. **Show the drafted message to the user for confirmation** before committing
   (use the AskUserQuestion tool with the full message rendered, options:
   confirm / edit / cancel). Do not commit until the user confirms.
9. **Commit** with `git commit -m "<subject>" -m "<body>"` (use multiple `-m`
   flags to create header/body/footer paragraphs, or a here-doc / file for
   multi-paragraph bodies). Stage files explicitly with `git add <file>` — never
   `git add .` or `git add -A` unless the user explicitly asks.
10. **Verify** with `git log --oneline -1` and show the hash + subject.

---

## Integration with other skills

- **`spectra-commit`** — for commits that belong to a Spectra change, prefer
  `spectra-commit`; it produces a `spectra(<change>):` prefix and stages only
  the change's files. The Conventional Commits rules (imperative, lowercase, no
  period, body explains why, ≤72 chars) still apply to the summary after the
  `spectra(<change>):` prefix.
- **`github-create-pr`** — PR titles often mirror the commit subject style
  (`type(scope): summary`). Use this skill to draft the PR title's
  `type(scope):` portion, then `github-create-pr` to send it with the
  confirmation gate.

---

## Error Handling

| Problem | What to do |
|---------|------------|
| Change spans multiple types | Suggest splitting into multiple commits; the spec recommends it. If the user insists on one, pick the highest-impact type. |
| Subject exceeds 72 chars | Move detail into the body; shorten the subject. |
| Not sure if it's `feat` or `fix` | `feat` adds new behaviour; `fix` corrects existing broken behaviour. A new capability = `feat`; restoring expected behaviour = `fix`. |
| Not sure if it's `refactor` or `perf` | `perf` is a refactor specifically to improve performance and should be backed by a benchmark; otherwise `refactor`. |
| Breaking change but `!` already in header | The `BREAKING CHANGE:` footer is optional when `!` is used; add it only if you need space for migration instructions. |
| Commit message already written by the user and non-conformant | Point out the specific rule violated (tense, caps, period, missing type) and propose a fixed version; do not silently rewrite. |

---

## Resources

### references/

- `references/type-and-scope-catalog.md` — Read when you need the full rationale
  for each type, scope-naming guidance for this project's packages, and the
  complete Angular/commitlint type list with examples. Also contains the
  SemVer-bump decision table in detail.

### scripts/

(none — this skill is pure guidelines; the actual commit is performed with
`git commit`.)

### assets/

(none)