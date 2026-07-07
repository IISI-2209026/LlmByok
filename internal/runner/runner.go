// Package runner 建置以 BYOK（Bring Your Own Key）provider 設定
// 啟動 Copilot CLI 所需的環境。
package runner

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// byokKeys 是 BuildEnv 會從現行程序環境中覆寫的四個 Copilot BYOK
// 環境變數鍵名。
var byokKeys = map[string]struct{}{
	"COPILOT_PROVIDER_BASE_URL": {},
	"COPILOT_PROVIDER_TYPE":     {},
	"COPILOT_PROVIDER_API_KEY":   {},
	"COPILOT_MODEL":              {},
}

// BuildEnv 回傳環境切片（os.Environ() 形式的 "KEY=VALUE"
// 字串），適合指定給 exec.Cmd.Env。它以現行程序環境
// （os.Environ()）為起點，並覆寫下列四個 Copilot BYOK 變數：
//
//	COPILOT_PROVIDER_BASE_URL = profile.APIBase
//	COPILOT_PROVIDER_TYPE     = profile.Provider（空字串時回退為 "openai"）
//	COPILOT_PROVIDER_API_KEY  = profile.APIKey
//	COPILOT_MODEL             = model（呼叫端已解析的單一模型字串）
//
// 其餘現有環境變數保持不變。model 為空時 COPILOT_MODEL 設為空字串；
// 模型解析（候選清單選擇）由呼叫端（cmd/launch 層）完成。
func BuildEnv(profile *config.Profile, model string) []string {
	env := make([]string, 0, len(os.Environ())+4)

	// 複製現有環境，略過既存的 BYOK 鍵，使下方的覆寫成為
	// 這些鍵的唯一資料來源。
	for _, entry := range os.Environ() {
		key := entry
		if i := strings.IndexByte(entry, '='); i >= 0 {
			key = entry[:i]
		}
		if _, isByok := byokKeys[key]; isByok {
			continue
		}
		env = append(env, entry)
	}

	// 附上覆寫後的 BYOK 項目。
	provider := profile.Provider
	if provider == "" {
		provider = "openai"
	}

	env = append(env,
		"COPILOT_PROVIDER_BASE_URL="+profile.APIBase,
		"COPILOT_PROVIDER_TYPE="+provider,
		"COPILOT_PROVIDER_API_KEY="+profile.APIKey,
		"COPILOT_MODEL="+model,
		"COPILOT_PROVIDER_MAX_PROMPT_TOKENS=1048576",
		"COPILOT_PROVIDER_MAX_OUTPUT_TOKENS=131072",
	)

	return env
}

// Launch 以 profile（及可選的 modelOverride）建置的 BYOK 環境變數，
// 將 exePath 指向的可執行檔啟動為子程序。stdin、stdout 與 stderr 透明
// 連接，讓使用者如常與子程序互動。父程序環境永不被修改 — 僅子程序
// 接收覆寫後的變數。
//
// exePath 必須為絕對路徑或可於 PATH 中解析的名稱；呼叫端通常在
// 呼叫 Launch 前先以 exec.LookPath 解析之。
//
// extraArgs 會原樣附加為子程序的命令列參數；傳入 nil 或空切片
// 時不附加任何參數（與舊版行為一致）。
func Launch(profile *config.Profile, model, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command(exePath, extraArgs...)
	cmd.Env = BuildEnv(profile, model)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
