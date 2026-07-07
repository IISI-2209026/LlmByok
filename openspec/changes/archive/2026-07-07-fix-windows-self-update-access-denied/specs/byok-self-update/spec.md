## MODIFIED Requirements

### Requirement: `byok update` self-update command

The CLI SHALL provide a `byok update` subcommand that checks the latest GitHub Release matching the current binary's update channel, downloads the platform-appropriate release asset, and atomically replaces the currently running executable with the downloaded binary. The update channel SHALL be derived from the current version string: a version containing `-dev.` belongs to the `dev` channel and SHALL only consider releases marked `prerelease=true`; any other version belongs to the `stable` channel and SHALL only consider releases marked `prerelease=false`. The command SHALL NOT cross channels unless the user explicitly overrides the channel via the `--channel` flag. The command SHALL accept a `--channel` flag accepting the values `prerelease` or `release` (mapped to the `dev` or `stable` channel respectively); when provided, the command SHALL query the specified channel regardless of the current version, enabling a stable binary to update to a prerelease or vice versa. When `--channel` is omitted, the auto-detected channel SHALL be used. The command SHALL reject any other `--channel` value with an error and exit 1 without making an API request. When the current version is already the latest in its (possibly overridden) channel, the command SHALL print a "already up to date" message and exit 0 without modifying any file. The command SHALL accept a `--check` flag that only queries and prints the latest available version without downloading or replacing any file; `--check` and `--channel` SHALL be combinable. On Windows, the executable replacement SHALL use a rename-then-move strategy: the currently running executable SHALL be renamed to a backup name before moving the new binary into the original path, so that the replacement succeeds even when the target file is locked by the running process. The backup file SHALL be deleted if possible, or scheduled for deletion at next reboot via `MOVEFILE_DELAY_UNTIL_REBOOT` if it remains locked.

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

#### Scenario: Windows self-update replaces running executable via rename-then-move

- **WHEN** `byok update` is invoked on Windows and the target executable (`byok.exe`) is locked by the currently running process and a newer in-channel release is available
- **THEN** the command SHALL rename the running executable to a backup name (e.g., `byok.exe.old`), move the downloaded temporary binary to the original executable path, and exit 0 with the success message; the original executable path SHALL contain the new version binary

#### Scenario: Windows backup file scheduled for deletion at reboot

- **WHEN** the backup file (`byok.exe.old`) cannot be deleted immediately after the rename-then-move because it is still locked by the running process
- **THEN** the updater SHALL schedule deletion of the backup file at next reboot via `MOVEFILE_DELAY_UNTIL_REBOOT` and the update SHALL still be considered successful

#### Scenario: Windows rename-then-move failure restores backup

- **WHEN** the rename-then-move strategy fails after renaming the running executable to a backup name but before moving the new binary into place
- **THEN** the updater SHALL attempt to restore the backup to the original path and SHALL exit 1 with an error message

#### Scenario: Existing backup file from prior update is overwritten

- **WHEN** a backup file (`byok.exe.old`) from a prior update already exists in the target directory
- **THEN** the updater SHALL overwrite or remove the existing backup before renaming the current executable, and the update SHALL proceed normally
