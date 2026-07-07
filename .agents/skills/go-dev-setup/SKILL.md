---
name: go-dev-setup
description: >
  Check, install, and verify the Go development environment for the LlmByok
  project. Use this skill whenever the user asks to "install Go", "set up Go
  dev environment", "setup 開發環境", "安裝開發環境", "check Go toolchain",
  "verify Go env", or wants to prepare the project for development. Also use
  when the user encounters Go build/test errors that may stem from a missing
  or mismatched toolchain, missing C compiler (for -race), or stale
  dependencies. Covers: reading go.mod for the required Go version, verifying
  the installed Go matches, downloading module dependencies, building,
  vetting, running tests (with and without -race), and installing a C
  compiler (MinGW-w64) on Windows for CGO_ENABLED=1 race-detector support.
  Do NOT use for routine `go build`/`go test` during development — that is
  normal workflow, not environment setup. Do NOT use for non-Go projects.
---

# Go Development Environment Setup

## Overview

This skill ensures the Go toolchain, module dependencies, and (on Windows)
the C compiler required by `go test -race` are all present and compatible
with the LlmByok project before development begins.

## Quick Reference

| Task | Approach |
|------|----------|
| Check Go version | `go version` — compare against `go.mod` `go` directive |
| Download deps | `go mod download` |
| Build | `go build ./...` |
| Vet | `go vet ./...` |
| Test (no race) | `go test ./...` |
| Test (with race) | `CGO_ENABLED=1 go test ./... -race` — requires C compiler |
| Install C compiler (Windows) | `winget install BrechtSanders.WinLibs.POSIX.UCRT` |

---

## Step 1: Read the Required Go Version

Read `go.mod` in the project root and note the `go` directive (e.g.
`go 1.26.4`). This is the minimum Go version the project targets. The
installed Go must be **equal or newer**.

---

## Step 2: Check the Installed Go Toolchain

Run:

```powershell
go version
```

- If `go` is not found → Go is not installed. Install it first (see
  "Installing Go from scratch" below).
- If the installed version is **older** than the `go.mod` directive → upgrade
  Go to a matching or newer version.
- If the installed version is **equal or newer** → proceed to Step 3.

### Installing Go from scratch

If Go is not installed, use winget (Windows):

```powershell
winget install --id GoLang.Go --accept-source-agreements --accept-package-agreements
```

After installation, **restart the shell** so the new PATH takes effect, then
re-run `go version` to confirm.

---

## Step 3: Download Module Dependencies

```powershell
cd <project-root>
go mod download
```

This fetches all modules listed in `go.mod` / `go.sum` into the module cache.
It should produce no output on success. If it fails, check network access and
proxy settings (`GOPROXY`).

---

## Step 4: Build and Vet

```powershell
go build ./...
go vet ./...
```

Both should exit 0 with no output. Any errors here indicate a real code or
toolchain problem — fix before proceeding.

---

## Step 5: Run Tests

### Without race detector (always available)

```powershell
go test ./...
```

All packages should report `ok`. If any fail, investigate the test output.

### With race detector (requires CGO + C compiler)

The project's development spec requires `go test ./... -race`. The `-race`
flag requires CGO, which in turn requires a C compiler (`gcc`).

```powershell
gcc --version
```

- **If gcc is found** → run directly:

  ```powershell
  $env:CGO_ENABLED=1; go test ./... -race
  ```

- **If gcc is not found (Windows)** → install MinGW-w64 (see Step 6).

---

## Step 6: Install C Compiler on Windows (for -race)

### Preferred: winget

```powershell
winget install --id BrechtSanders.WinLibs.POSIX.UCRT --accept-source-agreements --accept-package-agreements
```

This installs WinLibs (MinGW-w64 GCC, POSIX threads, UCRT runtime). The
download is large (~200 MB) and may take several minutes — inform the user it
is a long-running operation and wait patiently.

### Alternative: scoop

If the user already uses [scoop](https://scoop.sh):

```powershell
scoop install gcc
```

### Post-install verification

After installation, the PATH may not be updated in the current shell. Reload
the machine + user PATH explicitly:

```powershell
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
gcc --version
```

Then run the race tests:

```powershell
$env:CGO_ENABLED=1; go test ./... -race
```

All packages should report `ok`. If `-race` still fails with
`-race requires cgo; enable cgo by setting CGO_ENABLED=1`, the PATH reload
did not pick up gcc — find the install location and add it to PATH manually.

---

## Verification Summary

A fully verified environment produces this output:

```
go version go1.26.4 windows/amd64          # matches go.mod
go mod download                             # (no output = success)
go build ./...                              # (no output = success)
go vet ./...                                # (no output = success)
go test ./...                               # all ok
$env:CGO_ENABLED=1; go test ./... -race     # all ok
```

If every line passes, the environment is ready for development.

---

## Error Handling

| Problem | What to do |
|---------|------------|
| `go: command not found` | Go is not installed or not on PATH. Install via winget or download from go.dev. |
| Installed Go older than go.mod directive | Upgrade Go: `winget upgrade GoLang.Go` or install the matching version. |
| `go mod download` fails | Check `GOPROXY` (default `proxy.golang.org`). Behind a firewall? Set `GOPROXY=https://goproxy.io,direct`. |
| `go build` fails after dependency change | Run `go mod tidy` to reconcile go.sum, then retry. |
| `-race requires cgo` | Install a C compiler (Step 6) and set `CGO_ENABLED=1`. |
| `gcc` still not found after winget install | Reload PATH from registry (see Step 6). If still missing, locate the install dir (commonly `C:\Program Files\winlibs-*\mingw64\bin`) and add to PATH manually. |
| Test timeout on `internal/runner` | The runner package launches real subprocesses and can take ~50 s. Use `initial_wait: 120` or higher for sync powershell calls. |

---

## Platform Notes

- **Windows**: CGO needs MinGW-w64. The project uses `go-keyring` which on
  Windows uses `wincred` (no CGO needed for keyring itself), but `-race`
  always needs CGO regardless of dependencies.
- **Linux**: Install `gcc` via the distro package manager (`apt install gcc`,
  `dnf install gcc`, etc.). `CGO_ENABLED=1` is often the default.
- **macOS**: Xcode Command Line Tools provide clang (`xcode-select --install`).
  `CGO_ENABLED=1` is the default.

---

## Resources

### scripts/

(none — all commands are standard `go` / `winget` invocations.)

### references/

(none)

### assets/

(none)