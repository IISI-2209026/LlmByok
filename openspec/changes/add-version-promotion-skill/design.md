## Context

版號目前以 `internal/version/version.go` 的 `Version` 字面值為唯一來源（現值 `dev`），由 Release workflow 以 `sed` 讀取後依分支附加後綴：main → `v<base>`、develop → `v<base>-dev`。Makefile 同样以 `sed` 讀取供本機 build 注入 ldflags。develop tag 固定 `-dev` 後綴，重複推送會撞 tag。沒有標準的「發布後跳下一版」流程。

## Decisions

### Decision: canonical base 版號來源為 version.go 字面值

`internal/version/version.go` 的 `Version` 字面值為 canonical base 版號（semver，無 `v` prefix、無後綴）。起始值設為 `0.1.0`（取代現行 `dev`）。Makefile 與 Release workflow 皆以 `sed` 讀取此字面值，維持單一來源。bump skill 直接編輯此字面值，不引入額外 VERSION 檔或以 Git tag 為來源，避免雙寫不一致。

### Decision: develop 預發布版號為 `<base>-dev.<run_number>`

develop 分支的預發布二進位版號與 tag 改為 `<base>-dev.<run_number>`（如 `0.1.0-dev.42` / tag `v0.1.0-dev.42`），`run_number` 取自 GitHub Actions `github.run_number`，確保每次推送 develop 產生唯一 tag，不再撞 tag。main 分支維持穩定 `<base>` / `v<base>`。Release workflow 的「Read version」步驟新增將 `github.run_number` 帶入版號與 tag 的邏輯（僅 develop 分支）。

### Decision: bump skill 採 patch 預設、可選 minor/major，執行後 commit + push 到 develop

skill `.github/skills/byok-bump-version/SKILL.md` 以預設 patch 等級晉升（`0.1.0`→`0.1.1`），可經參數選 minor（`0.2.0`）或 major（`1.0.0`）。skill 流程：讀取並解析 `internal/version/version.go` 的 base → 依等級計算下一版 → 編輯 `Version = "<next>"` → `git add internal/version/version.go` → commit `chore: bump version to <next>` → `git push origin develop`。skill 預設目標分支為 develop；若偵測當前在 main 分支則中止並提示應於 develop 執行（防呆）。skill 不建立 Git tag（tag 由 Release workflow 於推送後自動產生）。

### Decision: 版號晉升流程為「merge develop→main 觸發穩定發布 → 於 develop 執行 bump skill 跳下一版」

標準晉升流程（寫入 AGENTS.md 與 README.md）：(1) develop 累積預發布至可發布狀態 → (2) merge develop 到 main 並推送 main → Release workflow 自動產生穩定發布 `v<base>` → (3) 於 develop 執行 `byok-bump-version` skill 將 base 晉升到下一個 patch → (4) push 到 develop，使下一輪預發布使用更高的 base（如 `0.1.1-dev.N`），下一輪 main 發布即為 `0.1.1`。此順序確保 main 穩定版號與 develop 預發布版號單調遞增且不衝突。

## Implementation Contract

**Behavior:**

- `internal/version/version.go` 的 `Version` 字面值為 `0.1.0`（起始 base），取代現行 `dev`。
- 推送 develop → Release workflow 產生預發布，tag `v<base>-dev.<run_number>`、二進位版號 `<base>-dev.<run_number>`、`prerelease: true`。
- 推送 main → Release workflow 產生穩定發布，tag `v<base>`、二進位版號 `<base>`、`prerelease: false`。
- 執行 `byok-bump-version` skill（預設 patch）→ `version.go` 的 `Version` 改為下一個 patch 版號、產生 commit `chore: bump version to <next>` 並 push 到 `origin develop`。
- skill 偵測當前分支為 main 時中止並印出「請於 develop 分支執行」。

**Interface / data shape:**

- `.github/skills/byok-bump-version/SKILL.md`：描述 skill 觸發條件、參數（等級 patch/minor/major，預設 patch）、執行步驟（讀取→計算→編輯→commit→push）、錯誤處理（非 semver base、在 main 分支、push 失敗）。
- `.github/workflows/release.yml`「Read version」步驟：develop 分支 `SUFFIX="-dev.${{ github.run_number }}"`、`TAG="v${VERSION}-dev.${{ github.run_number }}"`、`PRERELEASE="true"`；main 維持原樣。三個 job（build/release/notify）的 version 步驟同步更新（或抽出以避免重複，至少結果一致）。
- `internal/version/version.go`：`var Version = "0.1.0"`。

**Failure modes:**

- `version.go` 的 base 非 semver → skill 中止並印出「無法解析版號」。
- skill 在 main 分支執行 → 中止並提示切到 develop。
- `git push` 失敗（如遠端落後）→ skill 印出錯誤並建議先 `git pull --rebase`，不強推。
- Release workflow中 `run_number` 帶入 tag：若同一次 run 的 develop 推送已產生 tag（極罕見，run_number 全域唯一），不會撞 tag。

**Acceptance criteria:**

- `internal/version/version.go` 的 `Version` 為 `0.1.0`，`go test ./internal/version/...` 通過。
- `release.yml` 靜態檢視確認 develop 分支 tag 含 `${{ github.run_number }}`、main 不含。
- `.github/skills/byok-bump-version/SKILL.md` 存在且描述完整步驟；手動依步驟執行可將 `0.1.0` bump 為 `0.1.1`、commit 並 push 到 develop。
- `AGENTS.md` 含「版本號機制」區塊；`README.md` 含「版本號與發布流程」段落。

**Scope boundaries:**

- In scope：canonical base 來源正式化、develop tag 改 `<base>-dev.<run_number>`、bump skill、AGENTS.md/README 文件。
- Out of scope：版本來源改為 VERSION 檔或 Git tag、平台矩陣變更、Pushover 邏輯變更、自動化 develop→main 合併、main 分支的 bump 流程。

## Risks / Trade-offs

- [以 `github.run_number` 作為 develop 預發布序號] → run_number 在 repo 層級單調遞增且唯一，足以避免撞 tag；其數值不連續（跨 workflow）屬可接受，因預發布版號僅需唯一與遞增，不需連續。
- [bump skill 直接編輯 version.go 字面值] → 以 `sed`/字串取代 `Version = "..."` 為脆弱點；skill 須以正則精確匹配並驗證結果為合法 semver，失敗則中止。
- [base 版號同時用於 develop 與 main] → merge develop→main 時兩者 base 相同，main 發布 `v<base>` 後才於 develop bump，確保 main 的穩定 tag 不會與下一輪 develop 預發布衝突（下一輪 base 已晉升）。
- [skill 僅 push 到 develop，不處理 main] → 防呆避免誤在 main 產生新 commit；main 的版號推進一律經 merge develop 取得。
