package runner

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// codexAPIKeyEnv 為承載 Codex BYOK API key 的內部環境變數名稱。
// 僅設定於 codex 子程序環境，父程序與 shell 不受影響。
const codexAPIKeyEnv = "BYOK_CODEX_API_KEY"

// codexProviderID 為 byok 用於 codex --config 覆寫的自訂 provider id。
// 採用不易衝突的固定 id，避開 codex 保留字（openai/ollama/lmstudio）。
const codexProviderID = "byok"

// BuildCodexArgs 以 profile 與可選的 modelOverride 建置 codex 子程序
// 所需的環境切片與 --config 旗標切片。
//
// env 以現行程序環境（os.Environ()）為起點，過濾掉既存的
// BYOK_CODEX_API_KEY 後附加 BYOK_CODEX_API_KEY=<profile.APIKey>；
// 其餘現有環境變數保持不變。
//
// configArgs 為成對的 ["--config", "<key>=<value>", ...] 切片，覆寫：
//
//	model                                  = modelOverride 或 profile.DefaultModel
//	model_provider                         = "byok"
//	model_providers.byok.name              = "BYOK"
//	model_providers.byok.base_url          = profile.APIBase
//	model_providers.byok.env_key           = "BYOK_CODEX_API_KEY"
//
// TOML 字串值以雙引號包裹（不經過 shell，故不需外層 shell quoting）。
func BuildCodexArgs(profile *config.Profile, modelOverride string) (env []string, configArgs []string) {
	env = make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		key := entry
		if i := strings.IndexByte(entry, '='); i >= 0 {
			key = entry[:i]
		}
		if key == codexAPIKeyEnv {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, codexAPIKeyEnv+"="+profile.APIKey)

	model := modelOverride
	if model == "" {
		model = profile.DefaultModel
	}

	configArgs = []string{
		"--config", `model="` + model + `"`,
		"--config", `model_provider="` + codexProviderID + `"`,
		"--config", `model_providers.` + codexProviderID + `.name="BYOK"`,
		"--config", `model_providers.` + codexProviderID + `.base_url="` + profile.APIBase + `"`,
		"--config", `model_providers.` + codexProviderID + `.env_key="` + codexAPIKeyEnv + `"`,
	}
	return env, configArgs
}

// LaunchCodex 以 BuildCodexArgs 組裝的環境與 --config 旗標啟動 codex
// 可執行檔為子程序。extraArgs 會插入於 --config 旗標之後、原樣附加為
// 子程序的命令列參數（呼叫端負責安排 --yolo 與透傳順序）。stdin、stdout
// 與 stderr 透明連接。父程序環境永不被修改 — 僅子程序接收覆寫後的變數。
//
// 命令列順序：codex [<--config ...>] [<extraArgs...>]。
func LaunchCodex(profile *config.Profile, modelOverride, exePath string, extraArgs []string, stdin io.Reader, stdout, stderr io.Writer) error {
	env, configArgs := BuildCodexArgs(profile, modelOverride)
	args := append([]string(nil), configArgs...)
	args = append(args, extraArgs...)

	cmd := exec.Command(exePath, args...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}