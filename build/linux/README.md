# Linux build notes (amd64 desktop)

## Code paths

| 能力 | 实现 |
|------|------|
| 托盘 | `internal/tray/tray_other.go`（energye/systray + libayatana-appindicator / StatusNotifier） |
| 提权写 hosts | `pkexec` → `lxqt-sudo`/`gksudo`/`kdesudo` → `sudo` |
| 开机自启 | XDG `~/.config/autostart/GoogleTranslateHostsUpdater.desktop` |
| DNS 刷新 | `resolvectl` / `systemd-resolve` / `nscd` / `nmcli`（best-effort） |
| 代理插件 | 扫描 `~/.config/google-chrome`、`chromium`、`microsoft-edge` |

## 本机打包

```bash
# Debian/Ubuntu 类依赖示例
sudo apt-get update
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.0-dev libayatana-appindicator3-dev
# Wails v2.17 的 pkg-config 需要 webkit2gtk-4.0；部分新发行版才有 4.1

wails doctor
bash scripts/package-linux.sh
```

产物：`packages/GoogleTranslateHostsUpdater-v*-linux-amd64-*.AppImage`

> **DMG vs AppImage：** `.dmg` 是 macOS 磁盘映像，**不能**用于 Linux。本项目 Linux 发布格式为 **AppImage**（对等的单文件可执行包）。AppImage 未完整捆绑 WebKit；目标机仍需桌面 GTK/WebKitGTK。

## 限制

- 需要 **Linux 主机 + CGO**（GTK/WebKit）；无法在 Windows 上交叉编译完整 GUI。
- 发行版差异大：无托盘宿主时菜单可能不可见，请用主窗口操作。
- 完整桌面验收需真机；CI（`ubuntu-latest`）负责产出 Release 资源。
