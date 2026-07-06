## 1. 修正 Release Workflow 打包二進位檔名

- [x] 1.1 在 `.github/workflows/release.yml` 的 matrix include 項加入 `binary` 欄位（`windows` → `byok.exe`，`linux`/`darwin` → `byok`），並將 Build binary、Package (zip)、Package (tar.gz) 三個步驟的固定 `byok` 改為 `${{ matrix.binary }}`，使 Windows 產出 zip 內含 `byok.exe`、其他平台 tar.gz 內含 `byok`。驗證：以 `yq` 或人工檢視確認 windows matrix 的 `-o` 與 `zip` 命令引用 `byok.exe`，linux/darwin matrix 引用 `byok`；對應 spec「Multi-platform build via GitHub Actions matrix」之 scenario「Windows archive member has exe extension」與「Non-Windows archive member has no extension」。

## 2. 驗證 Updater 相容性

- [x] [P] 2.1 確認 updater 的 `extractZip` 與 `extractTarGz` 已同時接受 `byok.exe` 與 `byok` 成員（程式碼已支援），`SelectAsset` 依資產檔名（`byok-<version>-<os>-<arch>.<ext>`）比對不受成員名影響，不需修改 updater 程式碼。驗證：`go test ./internal/updater/ -race` 全數通過，其中 `TestDownloadAndExtract_Zip` 以 `byok.exe` 成員驗證、`TestDownloadAndExtract_TarGz` 以 `byok` 成員驗證。
