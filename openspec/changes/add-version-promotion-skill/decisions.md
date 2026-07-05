# 自主決策紀錄 — add-version-promotion-skill

> 使用者指示：「有什麼問題都先自己先做決定不要問我，請把你有疑問自己做決定的部分記錄在一個檔案中」
> 本檔記錄在實作過程中遇到疑問時，自行判斷後採用的決定。

## D1：`-race` 在本機 Windows 環境不可用
- 問題：Task 5.1 指定 `go test ./... -race`，但本機 Windows 環境 CGO 未啟用（無 C 工具鏈），`-race` 失敗。
- 決定：本機改執行 `go vet ./...` + `go test ./...`（不含 `-race`）；`-race` 由 CI workflow `pr-test.yml` 在 Ubuntu（cgo 可用）上執行，保留 race 保證。

## D2：bump skill 的 SKILL.md frontmatter 格式
- 問題：未指定 frontamage 欄位，需與既有 skills 一致。
- 決定：參考 `.github/skills/spectra-commit/SKILL.md` 的 frontmatter（name/description/license/compatibility/metadata），並在 metadata 標注 `generatedBy: "add-version-promotion-skill"`。

## D3：README 既有的「版本管理」段落處理
- 問題：Task 4.2 要求「新增版本號與發布流程段落」，但 README 已有「版本管理」段落且內容過時（預設值 `dev`、只提 main）。
- 決定：直接更新既有「版本管理」段落，整合新的 canonical base、develop/main 版號格式、晉升流程與 bump skill 說明，避免重複段落。