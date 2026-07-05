---
name: byok-bump-version
description: "Bump the canonical base version in internal/version/version.go, commit, and push to develop"
license: MIT
compatibility: Requires git and push access to origin develop.
metadata:
  author: byok
  version: "1.0"
  generatedBy: "add-version-promotion-skill"
---

Bump the `byok` canonical base version (the `Version` literal in `internal/version/version.go`), commit the change, and push to `origin develop`.

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
   > 錯誤：bump skill 不得在 main 分支執行。請切換到 develop 後再執行。
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

8. **Push to develop.**
   ```
   git push origin develop
   ```
   If the push fails (e.g., remote is ahead), **do not force-push**. Print the error and suggest:
   > 請先執行 `git pull --rebase origin develop` 後重試。

## Constraints

- **Does not create Git tags.** Tags are produced by the Release workflow on push.
- **Does not push to `main`.** The skill only pushes to `origin develop`.
- **Does not force-push.** On push failure, surface the error and advise a rebase.
- **Does not modify any file other than `internal/version/version.go`.**

## Verification (manual)

After running, confirm:

- `internal/version/version.go` contains `var Version = "<next>"`.
- `git log -1 --oneline` shows `chore: bump version to <next>`.
- `git push` succeeded (or a rebase guidance was printed).
- Running on `main` aborts without modifying files.