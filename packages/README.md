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

## Linux AppImage 怎么跑

**请以普通用户直接运行，不要 `sudo` 整个 AppImage。** 主界面与托盘必须在普通会话里；写入 `/etc/hosts` 时会通过 `pkexec`（polkit）只提权 `--apply-hosts` 助手并弹出授权对话框（与 Windows 上「普通 UI + 按次 UAC」同一模型）。

```bash
chmod +x GoogleTranslateHostsUpdater-v*-linux-amd64-*.AppImage
./GoogleTranslateHostsUpdater-v*-linux-amd64-*.AppImage
```

无 FUSE 时：

```bash
APPIMAGE_EXTRACT_AND_RUN=1 ./GoogleTranslateHostsUpdater-v*-linux-amd64-*.AppImage
# 或
./GoogleTranslateHostsUpdater-v*-linux-amd64-*.AppImage --appimage-extract-and-run
```

### 写 hosts 提权（Mint / Ubuntu 桌面）

1. 普通用户启动 AppImage（保留运行时注入的 `APPIMAGE` 环境变量）。
2. 点「立即刷新」或等到需要改 hosts 时，应出现 **pkexec / 策略套件** 密码或确认框。
3. 确认后仅短暂以 root 执行 `--apply-hosts`，写完即退出；主程序仍是普通用户。
4. **不要** `sudo ./…AppImage`：整进程 root 会导致托盘 D-Bus 崩溃；若误用 sudo，程序会跳过托盘并提示改用普通用户。

`Gtk-Message: Failed to load module "xapp-gtk3-module"` 在 Linux Mint 上通常无害，可忽略。

### AppImage 内已捆绑（构建机 Ubuntu 22.04 / webkit2gtk-4.0）

打包脚本用 **linuxdeploy + GTK plugin**，并把 WebKit 子进程路径打进镜像：

- `libwebkit2gtk-4.0` + `libjavascriptcoregtk-4.0`（JSC）
- WebKit 助手：`WebKitWebProcess` / `WebKitNetworkProcess`（及可选 `WebKitGPUProcess`）+ injected-bundle
- GTK3 / GDK3、GLib / GObject / GIO、libsoup
- cairo / pango / pangocairo / harfbuzz / atk / gdk-pixbuf 及相关 GTK 模块
- 托盘：`libayatana-appindicator3`（及 indicator / dbusmenu，若构建机有）

因此**不必**为了启动再 `apt install libwebkit2gtk-4.0-37`（这正是旧 AppImage 报 `libwebkit2gtk-4.0.so.37` 的原因）。

### 仍依赖主机（刻意不打进包）

| 类别 | 说明 |
|------|------|
| glibc / `ld-linux` | 目标发行版需能加载构建 ABI（CI 基线 Ubuntu 22.04） |
| 显示服务 | X11 或 Wayland 会话 |
| GPU 驱动 | Mesa/专有驱动留在系统 |
| 字体 | 使用系统 fontconfig/字体 |
| 托盘宿主 | StatusNotifier / AppIndicator 宿主；没有时仍可用主窗口 |
| FUSE | 可选；没有就用上面的 extract-and-run |
| 写 hosts 提权 | 系统需有 `pkexec`（policykit-1）；按次授权，勿 sudo 整包 |

### 若仍缺库（罕见 / 极旧发行版）

构建针对 **webkit2gtk-4.0**（不是 4.1）。若你**不用**捆绑版、改跑裸二进制，或极旧系统：

```bash
# Debian/Ubuntu 22.04 类
sudo apt-get install -y \
  libwebkit2gtk-4.0-37 \
  libgtk-3-0 \
  libayatana-appindicator3-1

# Ubuntu 24.04+ 默认仓库偏 webkit2gtk-4.1；请优先用本仓库打好的自包含 AppImage
# （4.1 与 4.0 SONAME 不兼容，不能互相替代）
```

更多构建说明见 [`build/linux/README.md`](../build/linux/README.md)。
