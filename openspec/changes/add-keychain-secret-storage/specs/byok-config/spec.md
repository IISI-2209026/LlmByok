## ADDED Requirements

### Requirement: Set API key in OS keychain

The `byok config set-key <profile>` command SHALL prompt the user to enter an API key via a non-echoing terminal read (`golang.org/x/term.ReadPassword`), store the entered key in the OS keychain under `profile:<profile.Name>`, and when the profile's `api_key` field in the config file is non-empty SHALL clear that field and rewrite the config file. When the entered key is empty, the command SHALL print "金鑰不可為空" and exit with code 1 without writing to the keychain. When the named profile does not exist in the config file, the command SHALL print "profile <name> 不存在" and exit with code 1.

#### Scenario: Set key for existing profile

- **WHEN** user runs `byok config set-key openai-official` and enters `sk-new` and the profile exists with a plaintext `api_key: sk-old`
- **THEN** `sk-new` is stored in the keychain under `profile:openai-official` and the config file is rewritten with `api_key` empty for that profile

#### Scenario: Empty key rejected

- **WHEN** user runs `byok config set-key openai-official` and presses Enter without typing a key
- **THEN** the command prints "金鑰不可為空" and exits with code 1 and the keychain is not modified

#### Scenario: Non-existent profile rejected

- **WHEN** user runs `byok config set-key nonexistent` and no profile named `nonexistent` exists
- **THEN** the command prints "profile nonexistent 不存在" and exits with code 1

---
### Requirement: Delete API key from OS keychain

The `byok config del-key <profile>` command SHALL delete the key stored under `profile:<profile.Name>` from the OS keychain. When the keychain has no entry for the profile, the command SHALL print "profile <name> 未在 keychain 中" and exit with code 1. When the named profile does not exist in the config file, the command SHALL print "profile <name> 不存在" and exit with code 1.

#### Scenario: Delete existing key

- **WHEN** user runs `byok config del-key openai-official` and the keychain contains `profile:openai-official`
- **THEN** the entry is removed and the command prints "已自 keychain 刪除金鑰（profile: openai-official）"

#### Scenario: Delete missing key

- **WHEN** user runs `byok config del-key openai-official` and the keychain has no entry for the profile
- **THEN** the command prints "profile openai-official 未在 keychain 中" and exits with code 1

---
### Requirement: Batch import plaintext keys into keychain

The `byok config import-keys` command SHALL iterate every profile in the config file whose `api_key` field is non-empty, store each key in the OS keychain under `profile:<profile.Name>`, then clear that profile's `api_key` field and rewrite the config file once after all profiles are processed. When a single profile's keychain store fails, the command SHALL record the failure and continue with the remaining profiles. When all stores succeed, the command SHALL print "匯入 N 個金鑰至 keychain" where N is the count of imported profiles. When one or more profiles failed, the command SHALL print the list of failed profile names and exit with code 1. When no profile has a non-empty `api_key`, the command SHALL print "設定檔中無明碼金鑰可匯入" and exit with code 0.

#### Scenario: Import all plaintext keys

- **WHEN** user runs `byok config import-keys` and the config file has two profiles with non-empty `api_key`
- **THEN** both keys are stored in the keychain, both `api_key` fields are cleared in the rewritten config file, and the command prints "匯入 2 個金鑰至 keychain"

#### Scenario: Partial failure continues and reports

- **WHEN** user runs `byok config import-keys` and the first profile stores successfully but the second fails
- **THEN** the first profile's `api_key` is cleared and the second is left as-is, and the command prints the failed profile name and exits with code 1

#### Scenario: No plaintext keys to import

- **WHEN** user runs `byok config import-keys` and no profile has a non-empty `api_key`
- **THEN** the command prints "設定檔中無明碼金鑰可匯入" and exits with code 0 without rewriting the config file

---
### Requirement: List profiles with key source indicator

The `byok config list` command SHALL, for each profile, display the key source alongside the existing masked key column. The source indicator SHALL be `keychain` when the key was resolved from the OS keychain, `plaintext` when resolved from the config file `api_key` field, and `missing` when neither source yielded a key. The masked key display SHALL use the resolved key value regardless of source.

#### Scenario: Mixed key sources in list

- **WHEN** user runs `byok config list` and profile `openai-official` has its key in the keychain while profile `local-ollama` has a plaintext `api_key` and profile `empty-profile` has no key in either source
- **THEN** the printed output shows `openai-official` with source `keychain`, `local-ollama` with source `plaintext`, and `empty-profile` with source `missing`

## MODIFIED Requirements

### Requirement: Add a profile

The `byok config add` command SHALL append a new profile to the config file with the fields `name`, `provider`, `api_base`, `api_key`, and `default_model`. The `api_key` field SHALL be optional; when omitted the profile is created without a plaintext key and the user is expected to set the key via `byok config set-key`. When the config file does not exist, it SHALL be created. When a profile with the same `name` already exists, the command SHALL print an error and exit with code 1 without modifying the file.

#### Scenario: Add new profile to empty config

- **WHEN** user runs `byok config add --name openai-official --provider openai --api-base https://api.openai.com/v1 --api-key sk-xxxx --default-model gpt-4o` and the config file does not exist
- **THEN** a config file is created at `~/.byok/config.yaml` containing the profile `openai-official` with the given fields

#### Scenario: Add profile without api key

- **WHEN** user runs `byok config add --name openai-official --provider openai --api-base https://api.openai.com/v1 --default-model gpt-4o` without `--api-key`
- **THEN** the profile is created with an empty `api_key` field and the command succeeds

#### Scenario: Duplicate profile name rejected

- **WHEN** user runs `byok config add --name openai-official ...` and a profile named `openai-official` already exists in the config file
- **THEN** the command prints an error stating the profile name already exists and exits with code 1 without modifying the config file
