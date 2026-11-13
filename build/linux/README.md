# Linux build notes (amd64 desktop)

## Code paths

| 能力 | 实现 |
|------|------|
| 托盘 | `internal/tray/tray_other.go`（energye/systray + libayatana-appindicator / StatusNotifier） |
| 提权写 hosts | 仅 `--apply-hosts`：`pkexec`（优先，带 DISPLAY/APPIMAGE）→ `lxqt-sudo`/`gksudo`/`kdesudo` → `sudo`；**禁止**整 GUI root |
| 开机自启 | XDG `~/.config/autostart/GoogleTranslateHostsUpdater.desktop` |
| DNS 刷新 | `resolvectl` / `systemd-resolve` / `nscd` / `nmcli`（best-effort） |
| 代理插件 | 扫描 `~/.config/google-chrome`、`chromium`、`microsoft-edge` |

## 本机打包

```bash
# Debian/Ubuntu 22.04 类（Wails v2.17 需要 webkit2gtk-4.0 的 pkg-config）
sudo apt-get update
sudo apt-get install -y \
  libgtk-3-dev \
  libwebkit2gtk-4.0-dev \
  libayatana-appindicator3-dev \
  pkg-config \
  build-essential \
  curl \
  file \
  desktop-file-utils \
  dpkg-dev \
  python3

wails doctor
bash scripts/package-linux.sh
```

产物：`packages/GoogleTranslateHostsUpdater-v*-linux-amd64-*.AppImage`

> **DMG vs AppImage：** `.dmg` 是 macOS 磁盘映像，**不能**用于 Linux。本项目 Linux 发布格式为 **AppImage**。
>
> **自包含策略：** `scripts/package-linux.sh` 用 **linuxdeploy + linuxdeploy-plugin-gtk** 把 WebKitGTK/GTK/GLib/libsoup/cairo-pango 与托盘相关 `.so` 打进 AppImage，并按 Wails 方式重定位 `WebKit*Process` 路径。GPU 驱动、桌面会话、字体、FUSE、glibc 仍用系统。详见 [`packages/README.md`](../../packages/README.md)。

## 运行

```bash
chmod +x packages/GoogleTranslateHostsUpdater-v*-linux-amd64-*.AppImage
# 普通用户 — 不要 sudo
./packages/GoogleTranslateHostsUpdater-v*-linux-amd64-*.AppImage
# 无 FUSE：
APPIMAGE_EXTRACT_AND_RUN=1 ./packages/GoogleTranslateHostsUpdater-v*-linux-amd64-*.AppImage
```

写 hosts 时应弹出 **pkexec** 对话框。提权会重新执行磁盘上的 `.AppImage`（`$APPIMAGE`），并带 `APPIMAGE_EXTRACT_AND_RUN=1`，避免 FUSE 挂载对 root 不可读。详见 [`packages/README.md`](../../packages/README.md)。

## 限制

- 需要 **Linux 主机 + CGO**（GTK/WebKit）；无法在 Windows 上交叉编译完整 GUI。
- CI 基线 **Ubuntu 22.04 + webkit2gtk-4.0**（与 Ubuntu 24.04 默认的 4.1 SONAME 不同；发布包以内嵌 4.0 为准）。
- 发行版差异大：无托盘宿主时菜单可能不可见，请用主窗口操作。
- **不要** `sudo` 跑完整 GUI/托盘（energye/systray 在 root 下会 D-Bus 崩溃）；误开时会跳过托盘并警告。
- Linux Mint 的 `xapp-gtk3-module` 加载失败提示通常可忽略。
- 完整桌面验收需真机；CI（`ubuntu-22.04`）负责产出 Release / artifact。
