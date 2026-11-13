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
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev libayatana-appindicator3-dev
# 若 4.1 不可用可试 libwebkit2gtk-4.0-dev

wails doctor
bash scripts/package-linux.sh
```

产物：`build/dist/GoogleTranslateHostsUpdater-v*-linux-amd64-*.tar.gz`

## 限制

- 需要 **Linux 主机 + CGO**（GTK/WebKit）；无法在 Windows 上交叉编译完整 GUI。
- 发行版差异大：无托盘宿主时菜单可能不可见，请用主窗口操作。
- 完整桌面验收需真机；CI（`ubuntu-latest`）负责产出 Release 资源。
