# packages/

发布用最终产物目录（zip / tar.gz / installer）。

本地或 CI 跑打包脚本后，产物会出现在这里，文件名与 GitHub Release 资源一致，例如：

- `GoogleTranslateHostsUpdater-v0.1.2-windows-amd64-YYYYMMDD-HHMM-portable.zip`
- `GoogleTranslateHostsUpdater-v0.1.2-linux-amd64-YYYYMMDD-HHMM.tar.gz`
- `GoogleTranslateHostsUpdater-v0.1.2-darwin-arm64-YYYYMMDD-HHMM.zip`

中间产物仍在 `build/bin/`（由 Wails 生成）。本目录下的二进制包默认不入库（见根 `.gitignore`）。
