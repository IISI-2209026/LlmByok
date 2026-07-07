---
name: github-create-pr
description: >
  Push the current branch to the remote and open a GitHub Pull Request via the
  REST API, reusing the local git credential helper for auth (no PAT env var
  needed). Use this skill whenever the user asks to "push and create a PR",
  "open a pull request", "發 PR", "建立 Pull Request", "提 MR", or to merge the
  current branch into another branch via a PR on GitHub. Also use when the user
  asks to update/fix the title or body of an existing PR, or when a previously
  created PR has garbled/mojibake Chinese text in its body. Covers: drafting a
  PR title and body from the branch's commit range, showing them to the user
  for explicit confirmation before sending, pushing the branch, and creating or
  patching the PR. Do NOT use for pushing without a PR, for merging a PR (use
  the GitHub UI or a merge API call directly), or for non-GitHub code review
  platforms (GitLab/Gitea/Bitbucket). **Mandatory: must notify the user before
  creating a PR and before merging — never skip these confirmation checkpoints.**
---

# GitHub Create PR

## Overview

Push the current branch to `origin` and open a GitHub Pull Request through the
REST API, authenticating via the local `git credential helper` (no `GH_TOKEN`
env var required). The skill always **shows the drafted PR title and body to
the user for confirmation before sending** — nothing is posted until the user
approves.

> ⚠ **強制通知規則** — 建立 PR 前與合併 PR 前都**必須**使用 `AskUserQuestion`
> 工具通知使用者並等待明確確認。未經使用者確認，**不得**建立 PR 或合併 PR。
> 此規則適用於所有 PR 流程，包含版號晉升（`byok-bump-version`）。

Why this skill exists: the GitHub MCP server tools available in this
environment are read-only (list/get/search), and `gh` CLI is not installed.
Calling the REST API via `curl`/PowerShell works but PowerShell 5.1's
`ConvertTo-Json` mangles non-ASCII text and GitHub strips HTML-like tags
(`<version>`) from PR bodies, producing garbled descriptions. This skill uses a
small Python helper that handles UTF-8 JSON correctly.

## Quick Reference

| Task | Approach |
|------|----------|
| Push branch + open PR | `git push -u origin <branch>` → draft title/body → confirm → `scripts/create_pr.py` |
| Update an existing PR's title/body | `scripts/create_pr.py --update-existing <N>` |
| Get auth token | `git credential fill` for `host=github.com` (handled inside the script) |

---

## Prerequisites

1. **git** is available (`git --version`).
2. A credential is stored for `https://github.com` (verify with
   `printf 'protocol=https\nhost=github.com\n\n' | git credential fill` — it
   should print a `password=` line). On Windows this is usually the Git
   Credential Manager. If no credential is stored, tell the user to authenticate
   with `git credential approve` or a `git push` first, then stop.
3. The current branch has a remote `origin` pointing at a GitHub repo
   (`git remote -v`). If the repo is a fork, confirm the owner/repo to target.

If any prerequisite is missing, report it and stop — do not attempt to create
the PR through a fallback that bypasses auth.

---

## Workflow

### 1. Inspect the branch

Run these to understand what will be in the PR:

```bash
git branch --show-current
git remote -v
git log --oneline -5
```

Identify the **base branch** (the branch the PR will merge into). If the user
named it explicitly (e.g. "merge into develop"), use that. Otherwise ask via
the AskUserQuestion tool which branch to target — do not assume `main`.

### 2. Push the branch (if not already pushed)

Check whether the branch exists on the remote first:

```bash
git ls-remote --heads origin <branch>
```

If empty, push and set upstream:

```bash
git push -u origin <branch>
```

If already pushed but local is ahead, push the new commits:

```bash
git push origin <branch>
```

If the local branch is up to date with the remote, skip pushing.

### 3. Gather the commit range and changed files

The PR body must describe **everything** the branch changes relative to the
base, not just the latest commit. A common mistake is to describe only the most
recent commit when the branch contains many.

```bash
git log --oneline origin/<base>..<head>
git diff --stat origin/<base>..<head>
```

Group the commits by the change/feature they belong to (Spectra change name,
chore, fix, etc.) and summarise the files each touched. Use `git show --stat
<sha>` for individual commits when the grouping is unclear.

### 4. Draft the PR title and body

**Title**: a concise `type(scope): summary` line. If the branch bundles several
changes, pick a title that reflects the overall theme, e.g.
`feat: add develop-branch CI release workflow (-dev prerelease)`.

**Body**: a markdown document covering, in order:

1. **摘要** — 1-2 sentences: what this PR merges and the total scope
   (N commits, M files, +/- lines).
2. **變更內容（依 commit 順序）** — one subsection per logical change, listing
   the commit SHA(s) and the key files/behaviour each introduces. Include
   archived Spectra changes and spec syncs.
3. **行為** — observable behaviour changes (e.g. what a push to each branch now
   does).
4. **驗證** — how the changes were validated (tests run, linters, manual
   checks).
5. **備註** — caveats, follow-ups, dependencies on other changes.

Write the body in the user's locale (Traditional Chinese for this project,
matching the `.spectra.yaml` `locale: tw`).

**Critical body-writing rules:**

- **Never use `<placeholder>` in the body.** GitHub strips anything that looks
  like an HTML tag, so `tag v<version>` renders as `tag v`. Use
  `{placeholder}` instead — e.g. `tag v{version}`, archive
  `byok-{version}-dev-{os}-{arch}.{ext}`. Add a one-line note at the end of the
  body: "佔位符以 `{version}`/`{os}`/`{arch}`/`{ext}` 表示（對應 spec 中的
  `<version>` 等）".
- Write the body to a **UTF-8 file**, not a shell variable. This avoids
  PowerShell 5.1 `ConvertTo-Json` mangling non-ASCII. The Python helper reads
  the file directly, so encoding is preserved end-to-end.

### 5. Show the user the title and body for confirmation

This is the mandatory gate — **do not skip it and do not create the PR before
the user confirms**. Present the drafted title and the full body so the user can
review and edit. Use the AskUserQuestion tool with these options:

- **Confirm and send** — proceed to create the PR as drafted.
- **Edit title/body** — let the user provide corrections (free-text input),
  then re-display and re-confirm.
- **Cancel** — do not create the PR.

When showing the body, render it as markdown so the user sees roughly what
GitHub will display. If the user edits, write the updated title/body back to the
UTF-8 files and re-show before sending.

### 6. Create (or update) the PR

Write the confirmed title and body to UTF-8 files (e.g. in the session's
`files/` directory — these are temp artifacts, not committed):

```
<session-files>/pr_title.txt   (UTF-8, no trailing newline issues — script strips)
<session-files>/pr_body.md     (UTF-8)
```

Then run the helper. To **create** a new PR:

```bash
python .agents/skills/github-create-pr/scripts/create_pr.py \
  --owner <owner> --repo <repo> \
  --head <head-branch> --base <base-branch> \
  --title-file <session-files>/pr_title.txt \
  --body-file <session-files>/pr_body.md
```

To **update** an existing PR's title/body (e.g. fixing a garbled body):

```bash
python .agents/skills/github-create-pr/scripts/create_pr.py \
  --owner <owner> --repo <repo> \
  --head <head-branch> --base <base-branch> \
  --title-file <session-files>/pr_title.txt \
  --body-file <session-files>/pr_body.md \
  --update-existing <PR_NUMBER>
```

The script prints `PR #N created: <url>` on success. On an HTTP error it prints
the status and response body to stderr and exits non-zero — surface the error
to the user and stop.

### 7. Verify and report

After the script succeeds, read the PR back with the
`github-mcp-server-pull_request_read` tool (`method: get`) and confirm:

- `title` and `body` render correctly (no mojibake, `{placeholders}` intact).
- `base.ref` matches the intended base branch.
- `mergeable_state` is `clean` (or note it as `unknown`/`blocked` if checks are
  still running).
- `commits` and `changed_files` counts match what `git log/diff --stat` showed.

Report the PR URL, mergeable state, and counts to the user. Clean up the temp
`pr_title.txt` / `pr_body.md` files (they live in the session `files/` dir and
are not committed, but removing them avoids clutter).

### 8. Notify user before merging (mandatory)

This skill does **not** merge PRs, but when the user expresses intent to merge
(or when a downstream skill like `byok-bump-version` requires a merge), you
**must** notify the user before any merge action is taken. Use the
`AskUserQuestion` tool with:

- PR number and URL
- PR title
- `mergeable_state` (clean / blocked / unknown)
- CI check status (passing / failing / pending)
- Ask: "是否同意合併此 Pull Request？"

**Do NOT merge the PR until the user explicitly confirms.** If the user
declines or cancels, stop and do not merge. This checkpoint is mandatory and
applies to all PR merges, including version bumps.

---

## Error Handling

| Problem | What to do |
|---------|------------|
| `git credential fill` returns no password | Tell the user to authenticate for github.com (e.g. `git credential approve` or any `git push`) and retry. Do not ask for a raw PAT in chat. |
| Branch already has an open PR | The API returns `422 Validation Failed: A pull request for branch X already exists`. Report the existing PR number (parse the error) and offer to update its title/body with `--update-existing`. |
| HTTP 400 "Problems parsing JSON" | The body file was not valid UTF-8 or had a BOM. Re-write it as UTF-8 without BOM and retry. |
| Garbled Chinese in the created PR body | You used PowerShell `ConvertTo-Json` or inline shell escaping instead of the Python helper + UTF-8 file. Re-run with `--update-existing <N>` using the Python helper. |
| `mergeable_state: blocked` / failing checks | Report it; do not attempt to merge via this skill (it only creates/updates PRs). If the user asks to merge, still notify them of the blocked state before any action. |
| `gh` CLI is installed | Prefer `gh pr create` if available — but still show the drafted title/body to the user for confirmation before running it. This skill's confirmation gate applies regardless of the creation mechanism. |

---

## Resources

### scripts/

- `scripts/create_pr.py` — Creates a new PR or PATCHes an existing one via the
  GitHub REST API. Reads title/body from UTF-8 files and gets the auth token
  from `git credential fill`. See the script's docstring for full usage.

  ```bash
  python .agents/skills/github-create-pr/scripts/create_pr.py --help
  ```

### references/

(none)

### assets/

(none)