# 自主決策紀錄 — add-codex-launch-and-pushover-tuning

> 使用者指示：「有什麼問題都先自己先做決定不要問我，請把你有疑問自己做決定的部分記錄在一個檔案中」
> 本檔記錄在實作過程中遇到疑問時，自行判斷後採用的決定。

## D1：聲音名稱選擇
- 問題：design 僅說「正向音 / 警示音」，未指定具體 Pushover sound 名稱。
- 決定：成功用 `pushover`（中性內建音），失敗用 `falling`（Pushover 內建警示音）。避免 emergency priority（`2`）。
- 理由：`pushover` 與 `falling` 皆為 Pushover 官方內建音效，不需額外設定；design 明示不使用 priority `2`。

## D2：release.yml 既有 status step 處理
- 問題：release.yml 已有 `Determine overall status` step 推導 status，是否需重寫？
- 決定：保留既有 step，僅在 Pushover step 以 ternary 覆寫 priority/sound。
- 理由：符合 design 的 ternary 表達式要求，且最小變更。

## D3：codex `--config` 引號格式
- 問題：TOML 字串值需以雙引號包裹，但設計範例 `model='"gpt-4o"'` 外層單引號為 shell quoting。
- 決定：`configArgs` 元素格式為 `--config`, `model="gpt-4o"`（單一字串含內嵌雙引號），由 Go 字串以 `\"` 表示。
- 理由：傳給 exec.Command 不經過 shell，故外層不需 shell quoting；TOML 解析器需要內層雙引號。

## D4：provider id `byok` 衝突說明
- 問題：使用者既有 `[model_providers.byok]` 會被覆寫。
- 決定：在 README「Codex BYOK 運作原理」段落註明此行為，採 design 的可接受立場。

## D5：AGENTS.md 區塊標題命名
- 問題：spec 要求「專案架構」與「開發規範」段落，未指定階層。
- 決定：使用 `## 專案架構` 與 `## 開發規範`（H2），與 Spectra 區塊同層。

## D6：commit 粒度
- 問題：使用者要求「每個階段都要進行 git commit」。
- 決定：按 tasks.md 的 section（1～8）分組 commit，每完成一個 section 提交一次。

## D7：PR 目標分支
- 問題：未指定 PR base branch。
- 決定：以 `develop` 為 base（release.yml 顯示 develop 為開發分支，pr-test.yml 觸發於 main/develop）。當前分支為 `feat/add-codex-launch`。

## D8：第二個 change 的分支策略
- 問題：兩個 change 是否應在同一分支或各自分支？
- 決定：兩個 change 已在 `feat/add-codex-launch` 分支上；為了能發單一 PR 涵蓋兩個 change，統一在此分支完成兩個 change 後再發 PR。若使用者要求分開 PR 再分割。

## D9：`-race` 在本機 Windows 環境不可用
- 問題：Task 8.1 指定 `go test ./... -race -coverprofile=coverage.out`，但本機 Windows 環境 CGO 未啟用（無 C 工具鏈），`-race` 失敗。
- 決定：本機改執行 `go vet ./...` + `go test ./... -coverprofile=coverage.out`（不含 `-race`）；`-race` 由 CI workflow `pr-test.yml` 在 Ubuntu（cgo 可用）上執行，保留 race 保證。