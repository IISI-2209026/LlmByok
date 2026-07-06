## 1. Runner 層：Claude 環境變數建置與子程序啟動

- [x] 1.1 在 `internal/runner/claude.go` 新增 `BuildClaudeEnv(profile *config.Profile, modelOverride string) []string`，回傳從 `os.Environ()` 複製後過濾 `ANTHROPIC_BASE_URL`/`ANTHROPIC_API_KEY`/`ANTHROPIC_MODEL` 再重新附加 profile 值的子程序環境切片，滿足 "Launch Claude with BYOK profile" 與 "Parent process environment unchanged for claude" 的要求。對應 design 段落「Decision: Inject Claude BYOK via environment variables only」與「Decision: Add a runner.LaunchClaude function parallel to runner.LaunchCodex」。驗證：`internal/runner/claude_test.go` 中 `TestBuildClaudeEnv` 斷言三個 env key 正確且 parent 未受污染。
- [x] 1.2 在 `internal/runner/claude.go` 新增 `LaunchClaude(profile *config.Profile, modelOverride, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer) error`，以 `exec.CommandContext` 啟動 `exePath` 並傳入 `BuildClaudeEnv` 結果與 `extraArgs`，stdin/stdout/stderr 透明連接，非零退出碼以 `errExit` 包裹。驗證：`internal/runner/claude_test.go` 中 `TestLaunchClaudeArgs` 斷言傳入的 args 與 env 正確。
- [x] 1.3 在 `internal/runner/claude_test.go` 撰寫單元測試涵蓋 `BuildClaudeEnv` 的 env 過濾與注入、`LaunchClaude` 的 args/env 傳遞與非零退出碼傳播。驗證：`go test ./internal/runner/ -run TestBuildClaudeEnv -race` 與 `go test ./internal/runner/ -run TestLaunchClaude -race` 通過。

## 2. 指令層：runLaunchClaude 與 executable 檢查

- [x] [P] 2.1 在 `cmd/launch_claude.go` 新增 `runLaunchClaude(cfgPath, profileName, model string, extraArgs []string, stdout, stderr io.Writer) error`，結構與 `runLaunchCodex` 一致：載入設定檔、解析 profile、驗證 provider 為 `openai`（否則印錯誤並 exit 1，滿足 "Claude provider validation"）、以 `exec.LookPath("claude")` 檢查可執行檔存在（否則印錯誤並 exit 1，滿足 "Claude executable presence check"）、呼叫 `runner.LaunchClaude`。profile 不存在或設定檔不存在時印錯誤並 exit 1（滿足 "Claude missing profile error" 與 "Claude missing config file error"）。驗證：`cmd/launch_claude_test.go` 中對應測試。
- [x] [P] 2.2 在 `cmd/launch_claude_test.go` 撰寫測試涵蓋：profile 不存在、設定檔不存在、非 openai provider、`claude` 不在 PATH、正常啟動路徑。驗證：`go test ./cmd/ -run TestRunLaunchClaude -race` 通過。

## 3. Dispatch 接線：byok launch 新增 claude 目標

- [x] 3.1 修改 `cmd/launch.go` 的 `RunE` dispatch switch，新增 `claude` case 呼叫 `runLaunchClaude`，並泛化 `buildExtraArgs` 使其接受 yolo literal 參數（copilot/codex 傳 `--yolo`，claude 傳 `--dangerously-skip-permissions`），滿足 "Target tool selection and dispatch" 的 claude 分派與 "Claude YOLO mode flag" 的 `--dangerously-skip-permissions` 映射。對應 design 段落「Decision: Map the byok --yolo flag to claude --dangerously-skip-permissions」與「Decision: Generalize the extra-args builder for the yolo literal」。更新指令 Short/Example/usage template 的 Targets 列表為 `copilot`/`codex`/`claude`。驗證：`cmd/launch_dispatch_test.go` 中 `TestLaunchClaudeDispatch` 斷言 `claude` case 正確分派，`TestUnsupportedTarget` 斷言錯誤訊息列出三個目標。
- [x] 3.2 在 `cmd/launch_dispatch_test.go` 撰寫測試涵蓋：`byok launch claude` 分派至 claude flow、`byok launch`（省略目標）印錯誤並 exit 1、`byok launch gemini` 印錯誤列出 `copilot`/`codex`/`claude` 並 exit 1、`byok launch claude -y` 產生含 `--dangerously-skip-permissions` 的 extraArgs（滿足 "Claude argument passthrough via double dash"）。驗證：`go test ./cmd/ -run TestLaunch -race` 通過。

## 4. README 文件更新

- [x] [P] 4.1 修改 `README.md`，將簡介從僅針對 Copilot 的一般化描述改為涵蓋 `copilot`、`codex`、`claude` 三個 target；更新「主要功能」、「解決什麼問題」與「前置需求」段落提及三個 CLI 工具。滿足 "Claude launch documentation in README" 的簡介一般化要求。驗證：`README.md` 前言段落不再僅提 Copilot。
- [x] [P] 4.2 修改 `README.md` 的 `byok launch <target>` 段落：Targets 表新增 `claude` 列、範例新增 `byok launch claude` 相關用法、`--yolo` 旗標說明註明 claude 映射為 `--dangerously-skip-permissions`。滿足 "Claude launch documentation in README" 的 Targets 表與範例要求。驗證：Targets 表列出三個 target 且含 claude 範例。
- [x] [P] 4.3 修改 `README.md` 的「運作原理」段落，新增 "Claude BYOK" 小節說明 `ANTHROPIC_BASE_URL`/`ANTHROPIC_API_KEY`/`ANTHROPIC_MODEL` 注入子程序且不寫入 `~/.claude/settings.json`；「官方文件」段落新增 Claude Code model-config 文件連結；「疑難排解」段落新增 `claude` 不在 PATH 的項目。滿足 "Claude launch documentation in README" 的運作原理、官方文件與疑難排解要求。驗證：三個段落均含 claude 相關內容。

## 5. 全域驗證

- [x] 5.1 執行 `go test ./... -race` 確認所有測試通過、無資料競爭。驗證：命令以 exit code 0 完成。
- [x] 5.2 執行 `go vet ./...` 確認無警告。驗證：命令無輸出且 exit code 0。
