---
name: byok-bump-version
description: "Bump the canonical base version in internal/version/version.go, commit, and open a Pull Request to develop"
license: MIT
compatibility: Requires git, push access, and GitHub PR creation access.
metadata:
  author: byok
  version: "1.1"
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

9. **Notify user before creating PR.** Before creating the Pull Request, **must** use `AskUserQuestion` (or plain text if unavailable) to notify the user with the following information and wait for confirmation:
   - Current branch name
   - Target branch: `develop`
   - PR title: `chore: bump version to <next>`
   - PR body summary: version bump from `<current>` to `<next>` (patch/minor/major)
   - Ask: "是否同意建立此 Pull Request？"
   - **Do NOT create the PR until the user explicitly confirms.**

10. **Create Pull Request to develop.** Use the `github-create-pr` skill to create a PR from the current branch to `develop`. If `github-create-pr` is not available, use the GitHub MCP API or `gh pr create --base develop --title "chore: bump version to <next>" --body "..."`.

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
- **Must notify user before creating PR and before merging.** These are mandatory checkpoints — never skip them.

## Verification (manual)

After running, confirm:

- `internal/version/version.go` contains `var Version = "<next>"`.
- `git log -1 --oneline` shows `chore: bump version to <next>`.
- The branch was pushed to `origin`.
- A Pull Request to `develop` was created (user was notified before creation).
- The PR was merged only after user confirmation.
- Running on `main` aborts without modifying files.