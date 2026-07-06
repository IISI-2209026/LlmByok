## Why

Release workflow 的 Windows 打包步驟以 `go build -o byok` 產出無副檔名的 `byok`，再壓入 zip，導致 Windows 平台資產內含 `byok` 而非 `byok.exe`。這與 byok-release spec 已有的要求（zip 內含 `byok.exe`）不一致，也讓 Windows 使用者解壓後無法直接執行。

## Root Cause

`.github/workflows/release.yml` 的 Build binary 步驟固定輸出檔名 `byok`（`-o byok`），Package (zip) 步驟也固定打包 `byok`，未依 GOOS 在 Windows 加上 `.exe` 副檔名。

## Proposed Solution

在 release workflow 中依 GOOS 條件化輸出檔名：Windows 矩陣項建置為 `byok.exe` 並打包 `byok.exe` 進 zip；Linux/macOS 維持 `byok` 進 tar.gz。 updater 的 `extractZip` 與 `extractTarGz` 已同時接受 `byok.exe` 與 `byok` 成員，`SelectAsset` 依資產檔名（`byok-<version>-<os>-<arch>.<ext>`）比對、與成員名無關，故 `byok update` 不受影響、不需修改。

## Non-Goals

- 不改變資產檔名命名規則（`byok-<version>-<os>-<arch>.<ext>`）。
- 不修改 updater 解壓邏輯（已相容 `byok.exe` 與 `byok`）。
- 不改變 Linux/macOS 打包行為。
- 不在本地 Makefile 或其他建置流程加 `.exe` 處理（僅修 release workflow）。

## Success Criteria

- Windows `windows/amd64` 矩陣產出的 zip 解壓後成員為 `byok.exe`。
- Linux/macOS 矩陣產出的 tar.gz 解壓後成員仍為 `byok`（無 `.exe`）。
- `byok update` 在 Windows 下載並解壓新 release 後仍能正確替換執行檔（extractZip 已相容 `byok.exe`）。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `byok-release`: 強化既有「Multi-platform build」需求，新增明確 scenario 指定 archive 內 binary 成員檔名須依平台帶（Windows）或不帶（Linux/macOS）`.exe` 副檔名。byok-self-update spec 不涉及成員檔名，無需變更。

## Impact

- Affected code:
  - Modified: `.github/workflows/release.yml`
- Affected specs:
  - Modified: `byok-release`（新增 scenario 強化 archive 成員檔名要求）
