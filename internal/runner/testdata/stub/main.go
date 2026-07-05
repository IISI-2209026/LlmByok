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
}