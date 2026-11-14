# 谷歌翻译 Hosts 更新器

跨平台桌面工具：按计划检测 `translate.googleapis.com` 可用 IP，**仅在当前 hosts 映射失效时**拉取远程 IP 库、测速选优并更新系统 hosts。  
技术栈：**Go + Wails v2**（设置窗口 + 系统托盘）。许可：**Apache-2.0**。当前版本：**0.1.4**。

**完整使用流程（推荐先读）：[`docs/user-flow.md`](docs/user-flow.md)**

## 已锁定行为

| 项 | 默认 |
|----|------|
| 检查频率 | **软件启动时刷新**（`on_app_start`）；可选每天本地午夜 / 按小时间隔 |
| Hosts 域名 | 默认 5 条：API×2 + `jnn-pa` + `.com` + `.com.hk`；可选地区 ccTLD 包（默认关） |
| 在线 IP 库 | **不定时**；仅当前 IP 探测失败且允许时拉取 |
| IPv6 | **关**；开启后先测本机 IPv6，不可用则本轮 IPv4；双栈优先 IPv4 |
| 开机自启 | **关**；设置页可开（Win Run / macOS LaunchAgent / Linux XDG） |
| 权限模式 | **默认普通用户**；写 hosts 时按次提权。可选「以管理员身份运行」（Win/macOS）；Linux 始终普通用户 + `pkexec` |
| 许可 | Apache-2.0（`LICENSE` + `NOTICE`） |

## 致谢 / 灵感来源

IP 探测与测速思路、公开 IP 列表用法受 [GoodCoder666/GoogleTranslate_IPFinder](https://github.com/GoodCoder666/GoogleTranslate_IPFinder)（作者 [GoodCoder666](https://github.com/GoodCoder666)，GPL-3.0）启发；相关逻辑已用 Go **独立重写**，**不复制、不引入其 GPL 源码**。详见 [`NOTICE`](NOTICE)。

| 借鉴自 IPFinder（思路 / 约定，非源码） | 本仓库原创 |
|----------------------------------------|------------|
| HTTPS 探测模板（`/translate_a/element.js`）、SNI/`Host` | Wails 桌面 GUI 与设置页 |
| 并发测速选优 | 仅在当前 hosts IP **失效时**才拉库换 IP 并写 hosts |
| 公开 IP 库用法（如 [gtdb](https://github.com/GoodCoder666/gtdb) 列表 URL） | 系统托盘、计划检查、跨平台提权 |
| | 发布打包（portable zip / DMG / AppImage）、代理插件 bypass 等 |

## 安装（最终产物在 `packages/`）

| 平台 | 产物 | 怎么跑 |
|------|------|--------|
| **Windows** | `*-portable.zip`（可选 `*installer*.exe`） | 解压后运行 `GoogleTranslateHostsUpdater.exe`，或跑 NSIS 安装包 |
| **macOS (Apple Silicon)** | **`.dmg`** | 打开映像，拖到「应用程序」；未公证时可能需「仍要打开」 |
| **Linux (x86_64)** | **`.AppImage`** | `chmod +x` 后**普通用户**直接运行（**不要 sudo**；**不是 DMG**） |

发布包命名示例与 Linux AppImage（含捆绑 WebKit）说明见 [`packages/README.md`](packages/README.md)。推送 tag（如 `v0.1.4`）后，[`.github/workflows/release.yml`](.github/workflows/release.yml) 会在三平台构建并上传同名约定资源。

关闭窗口 = **隐藏到托盘**；托盘「退出程序」才真正结束。托盘可：立即刷新 / 打开设置 / 退出。

## 快速流程

1. 安装并运行 → 可选开机自启；Win/macOS 可选管理员模式  
2. 「立即刷新」或等计划检查（默认：软件启动时；可选午夜/间隔）→ 失效才换 IP，写入默认 5 域 + flushdns  
3. Chrome：**关闭安全 DNS(DoH)**；ZeroOmega 等对 5 域 **DIRECT**（工具可幂等自动配置；**先完全退出浏览器**）  
4. 海外原文页走 Socks5（如 `127.0.0.1:7890`）；**翻译域名走 DIRECT / hosts**  
5. 验证：系统 DNS = hosts IP；必要时 `chrome://net-internals/#dns` 清缓存  

目标重点是 **Chrome 右键/自动网页翻译**，不只是打开翻译网站。

## 两种权限模式

| 模式 | 行为 |
|------|------|
| **普通用户（默认）** | 日常非管理员；真正改 hosts（+ flushdns）时再提权 |
| **管理员模式（可选）** | 设置勾选「以管理员身份运行」→ 提权重启；之后换 IP 可静默写 hosts |

- **Windows**：`ShellExecute runas`；降权尝试关联令牌  
- **macOS**：osascript best-effort；降权常需手动退出再开  
- **Linux**：**始终普通用户运行**；写 hosts 时 `pkexec` 只提权 `--apply-hosts`（AppImage 走 `$APPIMAGE`）。**不要** `sudo` 整个 GUI/托盘（会找错 `~`、托盘 D-Bus 崩溃）

配置目录：Windows `%AppData%\GoogleTranslateHostsUpdater`；macOS `~/Library/Application Support/GoogleTranslateHostsUpdater`；Linux `~/.config/GoogleTranslateHostsUpdater`。

## 重要：Hosts ≠ 代理路径（ZeroOmega / 系统代理）

改 hosts **只影响本机 DNS 解析**。若代理插件把流量送去 `127.0.0.1:7890`，请求**不会走** hosts 优选 IP。

### 需要 DIRECT / bypass 的 5 个托管域名

| 域名 | 用途 |
|------|------|
| `translate.googleapis.com` | API / 测速 |
| `translate-pa.googleapis.com` | Chrome 网页翻译主调用 |
| `jnn-pa.googleapis.com` | Chrome 内置翻译相关 |
| `translate.google.com` | 网页入口 |
| `translate.google.com.hk` | 网页入口 |

情景模式：规则 → **`direct`**（放在宽规则 `*google*` 之前）；或 FixedProfile **bypassList**。

### 本工具自动配置（best-effort、幂等）

写入 hosts 后 / 首次检查 / 「配置代理插件绕过」：扫描 ZeroOmega/SwitchyOmega LevelDB；已正确则不写盘；缺失或笔误才修补。圆形感叹号悬停有中文说明。

**限制：** 浏览器占用 LevelDB 时写入会失败——请先**完全退出** Chrome/Chromium/Brave/Edge（含托盘图标），再点「配置代理插件绕过」；写入后重启浏览器；不覆盖非 Omega 扩展。

#### 各平台浏览器数据根（含 `Default` / `Profile *`）

1. 安装 **ZeroOmega**（Chrome 商店 ID `pfnededegaaopdmhkdmcofjmoldfiped`；Edge 商店 `dmaldhchmoafliphkijbfhaomcgglmgd`）或旧版 SwitchyOmega；打开过一次扩展设置以生成存储。  
2. **完全退出**浏览器后再点「配置代理插件绕过」。  
3. 工具扫描的常见根目录：
   - **Windows：** `%LOCALAPPDATA%\Google\Chrome\User Data`、`Chrome Beta`；`Microsoft\Edge\User Data`；`Chromium\User Data`；`BraveSoftware\Brave-Browser\User Data`
   - **macOS：** `~/Library/Application Support/Google/Chrome`（含 Beta/Canary）、`Chromium`、`Microsoft Edge`、`BraveSoftware/Brave-Browser`
   - **Linux：** `~/.config/google-chrome`、`google-chrome-stable`、`chromium`、`BraveSoftware/Brave-Browser`、`microsoft-edge`；以及 Flatpak `~/.var/app/…`、Snap `~/snap/…/common/…`
4. 情景模式：托管 5 域规则 → **`direct`**（放在宽规则 `*google*` 之前）；或 FixedProfile **bypassList**。

### 浏览器仍打不开翻译？

1. **关闭安全 DNS / DoH**（绕过 hosts）  
2. 确认 5 域 DIRECT/bypass  
3. 收藏 `.com` / `.com.hk`；勿依赖 `.cn`  
4. flushdns + 浏览器 DNS 缓存清理；解析结果应等于 hosts IP  
5. bypass 笔误 `ranslate.…` 改回 `translate.…`  

心智模型：**海外页 → Socks5**；**翻译 5 域 → DIRECT + hosts**。

## 开机自启动

| 平台 | 机制 |
|------|------|
| Windows | HKCU Run；NSIS 可选组件；设置页 |
| macOS | LaunchAgent plist |
| Linux | `~/.config/autostart/*.desktop` |

## 环境 / 从源码构建

- Go 1.23+（Wails v2.17 会拉 toolchain 1.26）
- Node.js + npm
- Wails CLI：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Windows：WebView2；（可选）NSIS
- Linux：GTK3 + WebKitGTK + AppIndicator（构建见 `build/linux/README.md`；**发布 AppImage 已捆绑 WebKit/GTK**，见 `packages/README.md`）
- macOS：Xcode CLT / Apple Silicon（见 `build/darwin/README.md`）

```powershell
wails doctor
```

### Windows（本机）

```powershell
cd d:\Code\Go\GoogleTranslate_refreshes_daily
wails dev
go test ./...
wails build
powershell -ExecutionPolicy Bypass -File .\scripts\package-windows.ps1
```

| 路径 | 说明 |
|------|------|
| `build/bin/GoogleTranslateHostsUpdater.exe` | 主程序（中间产物） |
| `packages/*-portable.zip` | Windows 便携包 |
| `packages/*installer*.exe` | NSIS 安装包（需 makensis） |
| `packages/*.dmg` | **macOS Apple Silicon** 磁盘映像 |
| `packages/*.AppImage` | **Linux x86_64** 单文件包（**不是 DMG**） |

### Linux amd64 / macOS arm64

GUI **需要对应平台 + CGO**，无法在 Windows 上交叉出可用托盘包：

```bash
# Linux 桌面 / ubuntu CI → packages/*.AppImage
bash scripts/package-linux.sh

# Apple Silicon Mac / macos CI → packages/*.dmg
bash scripts/package-darwin.sh
```

**格式约定：** macOS = **`.dmg`**；Linux = **`.AppImage`**（DMG 仅为 macOS 格式）。详见 [`packages/README.md`](packages/README.md)。

## 文档

- [`docs/user-flow.md`](docs/user-flow.md) — 端到端使用流程  
- [`docs/hosts-updater-plan.md`](docs/hosts-updater-plan.md)  
- [`packages/README.md`](packages/README.md) — 发布包格式与 Linux AppImage 运行说明  
- [`build/linux/README.md`](build/linux/README.md) / [`build/darwin/README.md`](build/darwin/README.md)  
- [`NOTICE`](NOTICE) / [`LICENSE`](LICENSE)  

## 已知缺口

- 本机无 NSIS 时安装包需另装 makensis  
- macOS 公证未做（`.dmg` 已产出）；首次打开可能需「仍要打开」；真机托盘细调留待硬件  
- CIDR 深度扫描未做  
