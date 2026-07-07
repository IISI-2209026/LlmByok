# byok-secret-storage Specification

## Purpose

TBD - created by archiving change 'add-keychain-secret-storage'. Update Purpose after archive.

## Requirements

### Requirement: OS keychain secret storage abstraction

The `internal/secret` package SHALL provide cross-platform OS keychain access via `zalando/go-keyring` with a fixed service name `byok` and per-profile key naming `profile:<profile.Name>`. It SHALL expose `Store(profileName, apiKey string) error`, `Load(profileName string) (string, error)`, `Delete(profileName string) error`, and `Exists(profileName string) (bool, error)`. `Store` SHALL overwrite any existing value for the same profile name. `Load` SHALL return an error distinguishable from a not-found condition when the OS keychain backend is unavailable (e.g. headless Linux without secret-service).

#### Scenario: Store then load a key

- **WHEN** `Store("openai-official", "sk-xxxx")` succeeds and `Load("openai-official")` is called
- **THEN** the returned string equals `sk-xxxx`

#### Scenario: Overwrite existing key

- **WHEN** `Store("openai-official", "sk-old")` then `Store("openai-official", "sk-new")` then `Load("openai-official")` are called
- **THEN** the returned string equals `sk-new`

#### Scenario: Delete removes the key

- **WHEN** `Delete("openai-official")` is called after a successful `Store`
- **THEN** a subsequent `Exists("openai-official")` returns `false` and `Load` returns a not-found error

#### Scenario: Load non-existent key

- **WHEN** `Load("never-stored")` is called
- **THEN** it returns a not-found error (not a backend-unavailable error)


<!-- @trace
source: add-keychain-secret-storage
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
-->

---
### Requirement: API key resolution order

The `internal/config` package SHALL resolve a profile API key via a `KeyResolver` interface `Resolve(Profile) (apiKey string, source Source, err error)` where `Source` is one of `SourceKeychain`, `SourcePlaintext`, `SourceMissing`. The default resolver SHALL: (1) call `internal/secret.Load` keyed by `profile:<profile.Name>` and on success return the key with `SourceKeychain`; (2) if keychain lookup fails with not-found or backend-unavailable, and `profile.APIKey` is non-empty, return the plaintext value with `SourcePlaintext`; (3) if neither source yields a key, return `SourceMissing` and an error stating the profile name and that both keychain and config were empty.

#### Scenario: Keychain hit

- **WHEN** the keychain contains `profile:openai-official` and the profile `api_key` field is empty
- **THEN** `Resolve` returns the keychain value with `SourceKeychain`

#### Scenario: Plaintext fallback when keychain empty

- **WHEN** the keychain has no entry for the profile and `profile.APIKey` is `sk-xxxx`
- **THEN** `Resolve` returns `sk-xxxx` with `SourcePlaintext`

#### Scenario: Both sources empty

- **WHEN** the keychain has no entry and `profile.APIKey` is empty
- **THEN** `Resolve` returns `SourceMissing` and an error containing the profile name

#### Scenario: Keychain backend unavailable but plaintext present

- **WHEN** `internal/secret.Load` returns a backend-unavailable error and `profile.APIKey` is `sk-xxxx`
- **THEN** `Resolve` returns `sk-xxxx` with `SourcePlaintext`

<!-- @trace
source: add-keychain-secret-storage
updated: 2026-07-06
code:
  - .agents/skills/go-dev-setup/SKILL.md
-->