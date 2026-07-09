---
name: byok-bump-version
description: "Bump the canonical base version in internal/version/version.go, commit, draft the PR title and body, confirm with the user, and open a Pull Request to develop"
license: MIT
compatibility: Requires git, push access, and GitHub PR creation access.
metadata:
  author: byok
  version: "1.2"
  generatedBy: "add-version-promotion-skill"
---

Bump the `byok` canonical base version (the `Version` literal in `internal/version/version.go`), commit the change, push the branch, and open a Pull Request to `develop`.

> ⚠ **禁止直接 push 到 develop** — 版號晉升必須透過 Pull Request 合併至 develop，不可直接推送。發送 PR 前與合併前都必須通知使用者確認。

This is a **utility skill**. It performs a single, well-scoped action and is safe to re-run.

## When to use

- The user asks to "bump version", "晉升版號", "升版", or "prepare next release".
- After a `develop → main` merge has triggered a stable release, to advance the base for the next development cycle.

## Parameters

- **bump level** — one of `patch` (default), `minor`, `major`. If the user does not specify, use `patch`.
  - `patch`: `0.1.0` → `0.1.1`
  - `minor`: `0.1.0` → `0.2.0`
  - `major`: `0.1.0` → `1.0.0`

## Steps

1. **Guard: not on main.** Run `git rev-parse --abbrev-ref HEAD`. If the current branch is `main`, **stop** and print:
   > 錯誤：bump skill 不得在 main 分支執行。請切換到 develop 或 feature 分支後再執行。
   Do not modify any file.

2. **Read current base.** Read `internal/version/version.go` and extract the `Version` literal with the regex:
   ```
   var Version = "([^"]+)"
   ```
   If the regex does not match, **stop** and print:
   > 錯誤：無法從 internal/version/version.go 解析版號字面值。

3. **Parse semver.** Parse the captured value as semver `MAJOR.MINOR.PATCH` (numeric only, no prefix). If it is not valid semver, **stop** and print:
   > 錯誤：目前版號 %q 非 semver（MAJOR.MINOR.PATCH），無法晉升。

4. **Compute next version** by the chosen level:
   - `patch`: `MAJOR.MINOR.(PATCH+1)`
   - `minor`: `MAJOR.(MINOR+1).0`
   - `major`: `(MAJOR+1).0.0`

5. **Edit `internal/version/version.go`.** Replace the `Version = "..."` literal with the next version, preserving the surrounding `var Version = "..."` form. The result must be exactly `var Version = "<next>"`.

6. **Validate result.** Re-read the file and confirm the new literal is valid semver. If not, **stop** and restore the original content.

7. **Commit.**
   ```
   git add internal/version/version.go
   git commit -m "chore: bump version to <next>"
   ```

8. **Push the current branch.**
   ```
   git push origin HEAD
   ```
   If the push fails (e.g., remote is ahead), **do not force-push**. Print the error and suggest:
   > 請先執行 `git pull --rebase origin <branch>` 後重試。

9. **Draft PR title and body, then confirm with user.** Before creating the Pull Request, draft the **full** PR title and body — not just a one-line summary — and present them to the user for review. This is a mandatory gate; do not skip it and do not create the PR before the user confirms.

   1. **Gather the commit range and changed files** relative to `develop`:
      ```
      git log --oneline origin/develop..HEAD
      git diff --stat origin/develop..HEAD
      ```
      The PR body must describe **everything** the branch changes relative to `develop`, not just the latest version-bump commit. Use `git show --stat <sha>` for individual commits when grouping is unclear.

   2. **Draft the title.** A concise `type(scope): summary` line. If the branch bundles several changes, pick a title that reflects the overall theme (e.g. `feat: ... and bump version to {next}`). For a pure version bump with no other changes, use `chore: bump version to {next}`.

   3. **Draft the body** in Traditional Chinese, following the `github-create-pr` skill's body structure, in order:
      - **摘要** — 1-2 sentences: what this PR merges and the total scope (N commits, M files, +/- lines).
      - **變更內容（依 commit 順序）** — one subsection per logical change, listing the commit SHA(s) and the key files/behaviour each introduces. The version-bump commit (`chore: bump version to {next}`) should be one of these subsections.
      - **行為** — observable behaviour changes (e.g. what the new version means for develop/main release flows).
      - **驗證** — how the changes were validated.
      - **備註** — caveats, follow-ups, placeholder note.

      **Critical body-writing rules:**
      - **Never use `<placeholder>` in the body.** GitHub strips HTML-like tags, so `tag v<version>` renders as `tag v`. Use `{placeholder}` instead — e.g. `tag v{version}`, `byok-{version}-dev-{os}-{arch}.{ext}`. Add a one-line note at the end of the body: "佔位符以 `{version}`/`{os}`/`{arch}`/`{ext}` 表示（對應 spec 中的 `<version>` 等）".
      - When the body is finalised, write it to a **UTF-8 file** (e.g. `<session-files>/pr_body.md`) rather than a shell variable, to avoid PowerShell encoding issues with non-ASCII text.

   4. **Present the drafted title and full body to the user** via the `AskUserQuestion` tool, offering these options:
      - **Confirm and send** — proceed to create the PR as drafted.
      - **Edit title/body** — let the user provide corrections, then re-display and re-confirm.
      - **Cancel** — do not create the PR.

      Render the body as markdown so the user sees roughly what GitHub will display. **Do NOT create the PR until the user explicitly chooses "Confirm and send".**

10. **Create Pull Request to develop.** Use the `github-create-pr` skill to create a PR from the current branch to `develop`, passing the **user-confirmed** title and body. Write the confirmed title to `<session-files>/pr_title.txt` (UTF-8, no BOM) and the confirmed body to `<session-files>/pr_body.md` (UTF-8), then run the Python helper:
    ```
    python .agents/skills/github-create-pr/scripts/create_pr.py \
      --owner <owner> --repo <repo> \
      --head <head-branch> --base develop \
      --title-file <session-files>/pr_title.txt \
      --body-file <session-files>/pr_body.md
    ```
    If `github-create-pr` is not available, use the GitHub MCP API or `gh pr create --base develop --title "..." --body "..."` — but still use the user-confirmed title and body. After creation, verify the PR with `github-mcp-server-pull_request_read` (`method: get`): confirm `title`, `body` (no mojibake, `{placeholders}` intact), `base.ref` is `develop`, and report the PR URL, mergeable state, and commit/file counts to the user.

11. **Notify user before merging.** After the PR is created, **must** use `AskUserQuestion` (or plain text if unavailable) to notify the user with:
    - PR URL/number
    - PR status (checks passing or not)
    - Ask: "是否同意合併此 Pull Request？"
    - **Do NOT merge the PR until the user explicitly confirms.**

12. **Merge PR (only after user confirmation).** Merge the PR via GitHub UI, `gh pr merge`, or the GitHub MCP API. Do not merge without explicit user approval.

## Constraints

- **Does not create Git tags.** Tags are produced by the Release workflow on push to develop/main.
- **Does not directly push to `develop` or `main`.** Version bumps must go through a Pull Request to `develop`.
- **Does not force-push.** On push failure, surface the error and advise a rebase.
- **Does not modify any file other than `internal/version/version.go`.**
- **Must notify user before creating PR and before merging.** These are mandatory checkpoints — never skip them. The pre-PR checkpoint requires drafting the **full** PR title and body (gather commit range, changed files; follow `github-create-pr` body-writing rules) and presenting them to the user for explicit confirmation — not just a one-line summary.

## Verification (manual)

After running, confirm:

- `internal/version/version.go` contains `var Version = "<next>"`.
- `git log -1 --oneline` shows `chore: bump version to <next>`.
- The branch was pushed to `origin`.
- A Pull Request to `develop` was created, with the full title and body presented to and confirmed by the user before creation (not just a summary).
- The PR was merged only after user confirmation.
- Running on `main` aborts without modifying files.