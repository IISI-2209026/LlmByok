## 1. Canonical base 版號

- [x] 1.1 修改 `internal/version/version.go`，將 `var Version = "dev"` 改為 `var Version = "0.1.0"` 作為起始 canonical base（實作 Canonical base version source）（對應設計 Decision: canonical base 版號來源為 version.go 字面值）。驗證：`go test ./internal/version/...` 通過，`go run . --version` 輸出 `byok version 0.1.0`。

## 2. Release workflow develop tag 唯一化

- [ ] 2.1 修改 `.github/workflows/release.yml` 三個 job（build/release/notify）的「Read version」步驟：develop 分支 `SUFFIX="-dev.${{ github.run_number }}"`、`TAG="v${VERSION}-dev.${{ github.run_number }}"`、`PRERELEASE="true"`；main 維持 `SUFFIX=""`、`TAG="v${VERSION}"`、`PRERELEASE="false"`（實作 Branch-specific release tag and prerelease flag 與 Branch-specific binary version string format）（對應設計 Decision: develop 預發布版號為 `<base>-dev.<run_number>`）。驗證：YAML 靜態檢視確認 develop tag 含 `${{ github.run_number }}`、main 不含；`full_version` 在 develop 為 `<base>-dev.<run_number>`。
- [ ] 2.2 確認 build 步驟 ldflags 注入的 `full_version` 在 develop 為 `<base>-dev.<run_number>`、main 為 `<base>`，產物命名隨之正確。驗證：YAML 靜態檢視確認 `-ldflags ...Version=${{ steps.version.outputs.full_version }}` 與產物檔名使用 `full_version`。

## 3. Bump skill

- [ ] 3.1 新增 `.github/skills/byok-bump-version/SKILL.md`：描述觸發條件（使用者要求晉升版號/bump version）、參數（bump 等級 patch（預設）/minor/major）、執行步驟（讀取 `internal/version/version.go` 的 `Version` 字面值→以正則精確匹配 `var Version = "..."`→解析 semver→依等級計算下一版→編輯字面值→驗證結果為合法 semver→`git add internal/version/version.go`→commit `chore: bump version to <next>`→`git push origin develop`）（實作 Version bump skill computes next base version、Bump skill edits version.go, commits, and pushes to develop、Bump skill guards against running on main、Bump skill push failure handling）（對應設計 Decision: bump skill 採 patch 預設、可選 minor/major，執行後 commit + push 到 develop）。驗證：手動依 SKILL.md 步驟執行可將 `0.1.0` bump 為 `0.1.1`、產生 commit 並 push 到 develop；在 main 分支執行時中止不改檔。
- [ ] 3.2 在 SKILL.md 明確標示 skill 不建立 Git tag（tag 由 Release workflow 於 push 後自動產生）、不 push 到 main、不強推。驗證：SKILL.md 文字審查確認含此限制。

## 4. AGENTS.md 與 README 文件

- [ ] 4.1 在 `AGENTS.md` 新增「版本號機制」區塊（位於開發規範之後）：說明 canonical base 來源為 `internal/version/version.go`、develop 二進位版號與 tag 為 `<base>-dev.<run_number>`、main 為 `<base>`/`v<base>`、晉升流程（merge develop→main 觸發穩定發布 → 於 develop 執行 `byok-bump-version` skill 跳下一版 → push 到 develop）（實作 Version bump mechanism documented in AGENTS.md and README）（對應設計 Decision: 版號晉升流程為「merge develop→main 觸發穩定發布 → 於 develop 執行 bump skill 跳下一版」）。驗證：`AGENTS.md` 含此區塊且流程步驟完整。
- [ ] 4.2 在 `README.md` 新增「版本號與發布流程」段落：說明 develop 預發布（`v<base>-dev.<run_number>`，prerelease）、main 穩定發布（`v<base>`）、如何使用 `byok-bump-version` skill 晉升版號（預設 patch，可選 minor/major，執行後 commit 並 push 到 develop）（實作 Version bump mechanism documented in AGENTS.md and README）。驗證：`README.md` 含此段落且步驟可複製執行。

## 5. 整體驗證

- [ ] 5.1 執行 `go vet ./...` 與 `go test ./... -race`，確認全部通過且無退化。驗證：指令結束碼為 0。
- [ ] 5.2 靜態檢視 `release.yml` 與 `internal/version/version.go`，確認 develop tag 含 `run_number`、base 為 `0.1.0`。驗證：人工檢視通過。
