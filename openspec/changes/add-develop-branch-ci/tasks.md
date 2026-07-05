## 1. Workflow 觸發與版本後綴判斷

- [x] 1.1 修改 `.github/workflows/release.yml` 的 `on.push.branches` 由 `[main]` 改為 `[main, develop]`，使 push 至 develop 也觸發 workflow。驗證：以 Python `yaml.safe_load` 解析 release.yml 成功且 `on.push.branches` 包含 `main` 與 `develop`。
- [x] 1.2 在 build job 與 release job 各自的「Read version」step 之後新增一步「Compute suffix」，依 `github.ref` 判斷輸出四個 step outputs：`version_suffix`（main 為空字串、develop 為 `-dev`）、`prerelease`（main 為 `false`、develop 為 `true`）、`tag`（main 為 `v<version>`、develop 為 `v<version>-dev`）、`full_version`（base 版本串接 suffix）。實作「以 github.ref 分支名稱判斷後綴與 prerelease」決策。驗證：yaml 解析後該 step 存在且四個 outputs 皆定義；不符合 main/develop 時 suffix 預設為空、prerelease 為 false（失敗模式保護）。

## 2. Build 與 Release 使用動態版本

- [x] 2.1 修改 build job 的「Build binary」step，ldflags 注入版本改用 `steps.version.outputs.full_version`；「Package (zip)」與「Package (tar.gz)」step 的 archive 檔名改用 `full_version`，使 develop 產出 `byok-<version>-dev-<os>-<arch>.<ext>`。對應「單一 workflow 檔案搭配 branch 條件判斷」與「版本後綴由 workflow 動態附加，不修改 version.go」決策（本任務僅更動 release.yml，不修改 `internal/version/version.go`）。驗證：對照 spec scenario "Push to develop triggers dev pre-release"，archive 命名含 `-dev`。
- [x] 2.2 修改 release job 的 `softprops/action-gh-release` action：`tag_name` 改用 `steps.version.outputs.tag`，新增 `prerelease: ${{ steps.version.outputs.prerelease }}` 欄位。完成「GitHub Release creation on main and develop branches」需求。驗證：對照 spec 三個 scenario（main→stable 非 prerelease、develop→`v<version>-dev` prerelease、feature→不觸發）皆符合；yaml 中 release action 區塊含 `tag_name` 與 `prerelease` 兩欄位。

## 3. 整體驗證

- [x] 3.1 以 Python `yaml.safe_load` 載入修改後的 release.yml 確認 YAML 語法正確無誤，且關鍵欄位（`on.push.branches`、`jobs.build.steps`、`jobs.release.steps`）結構完整。驗證：`yaml.safe_load` 不拋例外且回傳 dict。
- [x] 3.2 人工審查最終 release.yml 對照 design 的 Implementation Contract：main push 行為與改動前一致（tag `v<version>`、非 prerelease、`byok version` 輸出 `<version>`）；develop push 行為符合 dev pre-release（tag `v<version>-dev`、prerelease、`byok version` 輸出 `<version>-dev`、archive 含 `-dev`）；feature 分支不觸發。驗證：逐項勾對 Implementation Contract 驗收條件。
