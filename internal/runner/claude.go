package runner

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// claudeByokKeys 是 BuildClaudeEnv 會從現行程序環境中覆寫的三個
// Claude Code BYOK 環境變數鍵名。
var claudeByokKeys = map[string]struct{}{
	"ANTHROPIC_BASE_URL":               {},
	"ANTHROPIC_API_KEY":                {},
	"ANTHROPIC_MODEL":                  {},
	"CLAUDE_CODE_ALWAYS_ENABLE_EFFORT": {},
	"CLAUDE_CODE_EFFORT_LEVEL":         {},
	"CLAUDE_CODE_SUBAGENT_MODEL":       {},
}

// BuildClaudeEnv 回傳環境切片（os.Environ() 形式的 "KEY=VALUE"
// 字串），適合指定給 exec.Cmd.Env。它以現行程序環境（os.Environ()）
// 為起點，並覆寫下列三個 Claude Code BYOK 變數：
//
//	ANTHROPIC_BASE_URL = profile.APIBase
//	ANTHROPIC_API_KEY  = profile.APIKey
//	ANTHROPIC_MODEL    = model + "[1m]"（呼叫端已解析的單一模型字串，附加 [1m] 後綴）
//
// 其餘現有環境變數保持不變。父程序環境永不被修改。模型解析（候選清單
// 選擇）由呼叫端（cmd/launch 層）完成。
func BuildClaudeEnv(profile *config.Profile, model string, options ...string) []string {
	effort, subModel := "", ""
	if len(options) > 0 {
		effort = options[0]
	}
	if len(options) > 1 {
		subModel = options[1]
	}
	env := make([]string, 0, len(os.Environ())+6)

	for _, entry := range os.Environ() {
		key := entry
		if i := strings.IndexByte(entry, '='); i >= 0 {
			key = entry[:i]
		}
		if _, isByok := claudeByokKeys[key]; isByok {
			continue
		}
		env = append(env, entry)
	}

	// Claude CLI 要求透過第三方端點連線時，模型名稱必須附加 [1m]
	// 以啟用 1M token context window。
	if !strings.HasSuffix(model, "[1m]") {
		model += "[1m]"
	}

	env = append(env,
		"ANTHROPIC_BASE_URL="+profile.APIBase,
		"ANTHROPIC_API_KEY="+profile.APIKey,
		"ANTHROPIC_MODEL="+model,
	)
	if effort != "" {
		env = append(env, "CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1", "CLAUDE_CODE_EFFORT_LEVEL="+effort)
	}
	if subModel != "" {
		env = append(env, "CLAUDE_CODE_SUBAGENT_MODEL="+subModel)
	}

	return env
}

// LaunchClaude 以 BuildClaudeEnv 組裝的環境啟動 exePath 指向的
// claude 可執行檔為子程序。extraArgs 會原樣附加為子程序的命令列
// 參數。stdin、stdout 與 stderr 透明連接。父程序環境永不被修改 —
// 僅子程序接收覆寫後的變數。
func LaunchClaude(profile *config.Profile, model, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer, options ...string) error {
	cmd := exec.Command(exePath, extraArgs...)
	cmd.Env = BuildClaudeEnv(profile, model, options...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
