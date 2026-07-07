// Package version 提供 byok 的版本號變數，可於建置時透過 ldflags 注入。
//
// Version 字面值為 canonical base 版號（semver，無 prefix），起始值 "0.1.0"；
// 建置時使用以下 ldflags 覆寫為含分支後綴的完整版號：
//
//	-ldflags "-X github.com/IISI-2209026/LlmByok/internal/version.Version=0.1.0"
package version

// Version 是 byok 的 canonical base 版號（semver，無 prefix），可於 link 時以 ldflags 覆寫。
// Release workflow 依分支附加後綴：main 為 <base>、develop 為 <base>-dev.<run_number>。
var Version = "0.2.2"