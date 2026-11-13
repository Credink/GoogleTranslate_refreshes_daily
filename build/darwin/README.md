# macOS build notes (Apple Silicon / arm64)

## Code paths

| 能力 | 实现 |
|------|------|
| 托盘 | `internal/tray/tray_darwin.go`：`SetTemplateIcon` + 短标题 `Hosts` |
| 图标 | `build/darwin/icon.png` + `trayTemplate.png`（`icon_darwin.go` embed） |
| 提权 | `osascript` `do shell script … with administrator privileges` |
| 开机自启 | LaunchAgent：`~/Library/LaunchAgents/GoogleTranslateHostsUpdater.plist` |
| DNS 刷新 | `dscacheutil -flushcache` + `killall -HUP mDNSResponder` |
| 代理插件 | `~/Library/Application Support/Google/Chrome` 等 |

## 本机 / CI 打包

```bash
# Apple Silicon Mac 或 GitHub Actions macos-14
bash scripts/package-darwin.sh
```

产物：`build/dist/GoogleTranslateHostsUpdater-v*-darwin-arm64-*.zip`

## 限制

- 完整 `wails build` / CGO / 菜单栏托盘 **需要 macOS**；Windows 无法交叉验证 GUI。
- 本仓库 Release 由 `.github/workflows/release.yml` 在 `macos-latest`（优先 arm64）上构建并上传；若 runner 不可用则 Windows/Linux 仍会发布。
- **未做公证 / DMG**；首次打开若被拦截：系统设置 → 隐私与安全性 → 仍要打开。
- 真机建议验收：浅色/深色菜单栏图标、点击出菜单、隐藏到托盘、LaunchAgent 登录自启。
