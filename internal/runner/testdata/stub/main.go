// launch_integration_test.go 所使用的 stub copilot。
//
// 它將自身環境（每行一個 KEY=VALUE）寫入由 BYOK_STUB_OUT 環境
// 變數指定的檔案路徑，隨後以狀態 0 結束。藉此讓測試精確驗證
// runner.BuildEnv + runner.Launch 注入 BYOK 變數後，子程序收到了
// 哪些環境變數。
//
// 此檔案為獨立程式；由測試透過 `go build` 動態編譯，runner 套件
// 本身不會匯入它。
package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	out := os.Getenv("BYOK_STUB_OUT")
	if out == "" {
		os.Exit(0)
	}
	env := append([]string(nil), os.Environ()...)
	sort.Strings(env)
	_ = os.WriteFile(out, []byte(strings.Join(env, "\n")), 0600)

	// 若指定了參數輸出檔，則將命令列參數（每行一個）寫入，
	// 讓測試能驗證 extraArgs 是否正確轉發給子程序。
	argsOut := os.Getenv("BYOK_STUB_ARGS_OUT")
	if argsOut != "" {
		_ = os.WriteFile(argsOut, []byte(strings.Join(os.Args[1:], "\n")), 0600)
	}

	// 若指定了 models.json 輸出檔且 PI_CODING_AGENT_DIR 已設定，
	// 則讀取該目錄下的 models.json 並寫入輸出檔，讓 pi 測試能驗證
	// LaunchPi 寫入的 models.json 內容。向後相容：未設定時不執行。
	modelsOut := os.Getenv("BYOK_STUB_MODELS_OUT")
	piDir := os.Getenv("PI_CODING_AGENT_DIR")
	if modelsOut != "" && piDir != "" {
		data, err := os.ReadFile(filepath.Join(piDir, "models.json"))
		if err == nil {
			_ = os.WriteFile(modelsOut, data, 0600)
		}
	}
}