## ADDED Requirements

### Requirement: Pi dry-run renderer regression coverage

The pi branch of the `byok launch <target> --dry-run` renderer SHALL be protected by a unit test. The test SHALL provide API base `https://example.test/v1`, API key `real-secret`, model `gpt-5`, effort `high`, and pi yolo mode. It SHALL require output containing a quoted `***` placeholder, `models.json`, `PI_CODING_AGENT_DIR`, pi model and thinking arguments, pi yolo mapping `--approve`, and the platform-native temporary-directory cleanup fragment. It SHALL fail if `real-secret` appears in output.

#### Scenario: Pi dry-run command is rendered without exposing the API key

- **WHEN** the pi dry-run renderer is invoked with API key `real-secret`, model `gpt-5`, effort `high`, and yolo mode
- **THEN** the unit test confirms the generated command contains `***`, `models.json`, `PI_CODING_AGENT_DIR`, `pi --model gpt-5 --thinking high`, `--approve`, and platform-native cleanup, while excluding `real-secret`
