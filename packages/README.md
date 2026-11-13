# packages/

发布用最终产物目录。本地或 CI 跑打包脚本后写入此处，文件名与 GitHub Release 资源一致。

## 平台格式（「DMG 风格」单文件分发）

| 平台 | 格式 | 说明 |
|------|------|------|
| **Windows** | `*-portable.zip`（可选 NSIS `*installer*.exe`） | 便携 zip / 安装包 |
| **macOS (Apple Silicon)** | **`.dmg`** | 标准磁盘映像；拖到「应用程序」安装 |
| **Linux (x86_64 桌面)** | **`.AppImage`** | **不是 DMG**——DMG 仅为 macOS 格式；Linux 用 AppImage 作为对等的单文件可执行包 |

示例文件名：

- `GoogleTranslateHostsUpdater-v0.1.3-windows-amd64-YYYYMMDD-HHMM-portable.zip`
- `GoogleTranslateHostsUpdater-v0.1.3-darwin-arm64-YYYYMMDD-HHMM.dmg`
- `GoogleTranslateHostsUpdater-v0.1.3-linux-amd64-YYYYMMDD-HHMM.AppImage`

中间产物仍在 `build/bin/`。本目录下的二进制包默认不入库（见根 `.gitignore`）。
