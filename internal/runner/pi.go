package runner

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// BuildPiEnv 回傳環境切片（os.Environ() 形式的 "KEY=VALUE" 字串），
// 適合指定給 exec.Cmd.Env。它以現行程序環境為起點，覆寫
// PI_CODING_AGENT_DIR=<tempDir>，其餘環境變數保持不變。
// 父程序環境永不被修改。
//
// 與 BuildClaudeEnv 不同，pi 的 base URL 與 API key 透過臨時目錄中的
// models.json 注入（由 LaunchPi 負責），而非環境變數，因此 profile
// 參數在此函式中不參與環境組裝。
func BuildPiEnv(profile *config.Profile, tempDir string) []string {
	_ = profile

	env := make([]string, 0, len(os.Environ())+1)

	for _, entry := range os.Environ() {
		key := entry
		if i := strings.IndexByte(entry, '='); i >= 0 {
			key = entry[:i]
		}
		if key == "PI_CODING_AGENT_DIR" {
			continue
		}
		env = append(env, entry)
	}

	env = append(env, "PI_CODING_AGENT_DIR="+tempDir)

	return env
}

// LaunchPi 以臨時目錄 + models.json + PI_CODING_AGENT_DIR 環境變數
// 啟動 exePath 指向的 pi 可執行檔為子程序。它會：
//  1. 建立臨時目錄（os.MkdirTemp）
//  2. 寫入 models.json（{"providers":{"openai":{"baseUrl":"...","apiKey":"..."}}}）
//  3. 以 BuildPiEnv 組裝子程序環境
//  4. 將 --model <model（呼叫端已解析的單一模型字串）> 附加到 extraArgs 前端
//  5. 啟動子程序，連接 stdin/stdout/stderr
//  6. defer os.RemoveAll 清理臨時目錄
//
// 父程序環境與使用者 pi 設定檔永不被修改。模型解析（候選清單選擇）
// 由呼叫端（cmd/launch 層）完成。
func buildPiArgs(model, effort string, extraArgs []string) []string {
	args := []string{"--model", model}
	if effort != "" {
		args = append(args, "--thinking", effort)
	}
	return append(args, extraArgs...)
}

func LaunchPi(profile *config.Profile, model, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer, effort ...string) error {
	tempDir, err := os.MkdirTemp("", "byok-pi-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	models := map[string]any{
		"providers": map[string]any{
			"openai": map[string]any{
				"baseUrl": profile.APIBase,
				"apiKey":  profile.APIKey,
			},
		},
	}
	data, err := json.Marshal(models)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tempDir, "models.json"), data, 0600); err != nil {
		return err
	}

	e := ""
	if len(effort) > 0 {
		e = effort[0]
	}
	args := buildPiArgs(model, e, extraArgs)

	cmd := exec.Command(exePath, args...)
	cmd.Env = BuildPiEnv(profile, tempDir)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
