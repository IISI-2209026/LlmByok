## ADDED Requirements

### Requirement: `byok update` self-update command

The CLI SHALL provide a `byok update` subcommand that checks the latest GitHub Release matching the current binary's update channel, downloads the platform-appropriate release asset, and atomically replaces the currently running executable with the downloaded binary. The update channel SHALL be derived from the current version string: a version containing `-dev.` belongs to the `dev` channel and SHALL only consider releases marked `prerelease=true`; any other version belongs to the `stable` channel and SHALL only consider releases marked `prerelease=false`. The command SHALL NOT cross channels unless the user explicitly overrides the channel via the `--channel` flag. The command SHALL accept a `--channel` flag accepting the values `prerelease` or `release` (mapped to the `dev` or `stable` channel respectively); when provided, the command SHALL query the specified channel regardless of the current version, enabling a stable binary to update to a prerelease or vice versa. When `--channel` is omitted, the auto-detected channel SHALL be used. The command SHALL reject any other `--channel` value with an error and exit 1 without making an API request. When the current version is already the latest in its (possibly overridden) channel, the command SHALL print a "already up to date" message and exit 0 without modifying any file. The command SHALL accept a `--check` flag that only queries and prints the latest available version without downloading or replacing any file; `--check` and `--channel` SHALL be combinable.

#### Scenario: Stable binary updates to newer stable release

- **WHEN** the running binary version is `0.1.0` (stable channel) and the latest non-prerelease GitHub Release has tag `v0.1.1`
- **THEN** `byok update` downloads the asset matching the current `GOOS`/`GOARCH`, replaces the executable at `os.Executable()` path, prints `0.1.0 -> 0.1.1`, and exits 0

#### Scenario: Dev binary only considers prereleases

- **WHEN** the running binary version is `0.1.1-dev.42` (dev channel) and the latest prerelease GitHub Release has tag `v0.1.1-dev.50` while the latest stable release is `v0.1.1`
- **THEN** `byok update` updates to `0.1.1-dev.50` and SHALL NOT update to `v0.1.1`

#### Scenario: Already up to date

- **WHEN** the running binary version equals the latest release in its channel
- **THEN** `byok update` prints an "already up to date" message containing the current version and exits 0 without writing any file

#### Scenario: Check only does not modify files

- **WHEN** `byok update --check` is invoked and a newer version exists
- **THEN** the command prints the latest version and the current version and exits 0 without downloading or replacing the executable

#### Scenario: Channel override to prerelease from a stable binary

- **WHEN** the running binary version is `0.1.0` (stable channel) and `byok update --channel prerelease` is invoked and the latest prerelease GitHub Release has tag `v0.1.1-dev.50`
- **THEN** the command queries the `dev` channel and updates to `0.1.1-dev.50` instead of the stable `v0.1.1`

#### Scenario: Channel override to release from a dev binary

- **WHEN** the running binary version is `0.1.1-dev.42` (dev channel) and `byok update --channel release` is invoked and the latest non-prerelease GitHub Release has tag `v0.1.1`
- **THEN** the command queries the `stable` channel and updates to `0.1.1`

#### Scenario: Invalid channel value is rejected

- **WHEN** `byok update --channel beta` is invoked
- **THEN** the command prints an error naming the invalid value, exits 1, and makes no GitHub API request

#### Scenario: Platform asset not found

- **WHEN** the selected release has no asset whose name matches `byok-<version>-<goos>-<goarch>.<ext>`
- **THEN** `byok update` prints an error naming the missing `GOOS`/`GOARCH` asset and exits 1 without modifying the executable

#### Scenario: Network failure exits non-zero

- **WHEN** the GitHub Releases API request fails or times out
- **THEN** `byok update` prints an error and exits 1 without modifying any file

### Requirement: Startup version check after non-launch commands

After a `byok` subcommand other than `launch` or `update` completes, the CLI SHALL query the latest GitHub Release in the current binary's update channel and, when a newer version exists, print a single-line hint to stderr containing the latest version, the current version, and the text `byok update`. The check SHALL be skipped for the `launch` and `update` subcommands. The check SHALL tolerate network, timeout, rate-limit, and parse failures silently and SHALL NOT alter the command's exit code or stdout. The check MAY be disabled by setting the `BYOK_NO_UPDATE_CHECK=1` environment variable.

#### Scenario: Hint printed when newer version exists

- **WHEN** `byok config list` completes and the latest in-channel release is newer than the running version
- **THEN** a single line is printed to stderr mentioning the latest version, the current version, and `byok update`, and the exit code is unchanged

#### Scenario: No hint when up to date

- **WHEN** `byok config list` completes and the running version is already the latest in its channel
- **THEN** nothing is printed to stderr regarding updates

#### Scenario: Launch skips startup check

- **WHEN** `byok launch copilot` (or `byok launch codex`) is invoked
- **THEN** no version check is performed before or after the command

#### Scenario: Network failure is silent

- **WHEN** the startup version check request fails or times out
- **THEN** no update-related message is printed and the command's exit code is unchanged

#### Scenario: Disabled via environment variable

- **WHEN** `BYOK_NO_UPDATE_CHECK=1` is set and `byok config list` completes
- **THEN** no GitHub API request is made and no update hint is printed

### Requirement: Channel-aware version comparison

Version comparison SHALL only compare versions within the same channel. For the `dev` channel, the ordering SHALL be determined by the numeric `<run_number>` suffix in `-dev.<run_number>`. For the `stable` channel, the ordering SHALL follow semantic versioning of the `<base>`. A current version of literal `dev` (local development build without `-dev.`) SHALL be treated as not newer than any release and SHALL NOT trigger an update.

#### Scenario: Dev ordering by run number

- **WHEN** comparing `0.1.1-dev.42` (current) against `0.1.1-dev.50` (latest)
- **THEN** the latest is considered newer because 50 > 42

#### Scenario: Local dev build does not auto-update

- **WHEN** the current version is `dev` and any release exists
- **THEN** `IsNewer` returns false and `byok update` reports already up to date / no update
