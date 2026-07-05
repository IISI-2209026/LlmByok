// Package version 提供 byok 的版本號變數，可於建置時透過 ldflags 注入。
//
// 預設值為 "dev"；建置時使用以下 ldflags 覆寫：
//
//	-ldflags "-X github.com/IISI-2209026/LlmByok/internal/version.Version=0.1.0"
package version

// Version 是 byok 二進位檔版本號，可於 link 時以 ldflags 覆寫。
var Version = "dev"