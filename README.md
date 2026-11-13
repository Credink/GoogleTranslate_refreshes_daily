# 谷歌翻译 Hosts 每日更新器

跨平台桌面工具：按计划检测 `translate.googleapis.com` 可用 IP，**仅在当前 hosts 映射失效时**拉取远程 IP 库、测速选优并更新系统 hosts。  
技术栈：**Go + Wails v2**（设置窗口 + 系统托盘）。许可：**Apache-2.0**。

**完整使用流程（推荐先读）：[`docs/user-flow.md`](docs/user-flow.md)**

## 已锁定行为

| 项 | 默认 |
|----|------|
| Hosts 域名 | 默认 5 条：API×2 + `jnn-pa` + `.com` + `.com.hk`；可选地区 ccTLD 包（默认关） |
| 在线 IP 库 | **不定时**；仅当前 IP 探测失败且允许时拉取 |
| IPv6 | **关**；开启后先测本机 IPv6，不可用则本轮 IPv4；双栈优先 IPv4 |
| 开机自启 | **关**；设置页可开（Win Run / macOS LaunchAgent / Linux XDG） |
| 权限模式 | **默认普通用户**；写 hosts 时按次提权。可选「以管理员身份运行」长期静默更新 |
| 许可 | Apache-2.0（`LICENSE` + `NOTICE`） |

IP 探测思路参考 [GoogleTranslate_IPFinder](https://github.com/GoodCoder666/GoogleTranslate_IPFinder)（**不复制其 GPL 源码**）。

## 快速流程

1. 安装并运行（见下）→ 可选开机自启 / 管理员模式  
2. 「立即刷新」或等午夜计划 → 失效才换 IP，写入默认 5 域 + flushdns  
3. Chrome：**关闭安全 DNS(DoH)**；ZeroOmega 等对 5 域 **DIRECT**（工具可幂等自动配置）  
4. 海外原文页走 Socks5（如 `127.0.0.1:7890`）；**翻译域名走 DIRECT / hosts**  
5. 验证：系统 DNS = hosts IP；必要时 `chrome://net-internals/#dns` 清缓存  

目标重点是 **Chrome 右键/自动网页翻译**，不只是打开翻译网站。

## 环境

- Go 1.23+（Wails v2.17 会拉 toolchain 1.26）
- Node.js + npm
- Wails CLI：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Windows：WebView2；（可选）NSIS
- Linux：GTK3 + WebKitGTK + AppIndicator（见 `build/linux/README.md`）
- macOS：Xcode CLT / Apple Silicon（见 `build/darwin/README.md`）

```powershell
wails doctor
```

## 运行 / 打包

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

GUI **需要对应平台 + CGO**，无法在 Windows 上交叉出可用托盘包。请用脚本或 CI：

```bash
# 在 Linux 桌面 / ubuntu CI → packages/*.AppImage
bash scripts/package-linux.sh

# 在 Apple Silicon Mac / macos CI → packages/*.dmg
bash scripts/package-darwin.sh
```

**格式约定：** macOS 发 **`.dmg`**；Linux 发 **`.AppImage`**（DMG 仅为 macOS 格式，Linux 用 AppImage 作为对等的「单文件安装包」）。详见 [`packages/README.md`](packages/README.md)。

推送 tag（如 `v0.1.3`）后，[`.github/workflows/release.yml`](.github/workflows/release.yml) 会在三平台构建并发布 Release 资源到 [`packages/`](packages/) 同名约定。

## 两种权限模式

| 模式 | 行为 |
|------|------|
| **普通用户（默认）** | 日常非管理员；真正改 hosts（+ flushdns）时再提权 |
| **管理员模式（可选）** | 设置勾选 → 提权重启；之后换 IP 可静默写 hosts |

- **Windows**：`ShellExecute runas`；降权尝试关联令牌  
- **macOS / Linux**：osascript / pkexec（及 sudo 回退）best-effort；降权常需手动退出再开  

配置目录：Windows `%AppData%\…`；macOS `~/Library/Application Support/…`；Linux `~/.config/…`（均 `GoogleTranslateHostsUpdater`）。

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

限制：浏览器占用 DB 时请先退出；写入后重启浏览器；不覆盖非 Omega 扩展。

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

## 文档

- [`docs/user-flow.md`](docs/user-flow.md) — 端到端使用流程  
- [`docs/hosts-updater-plan.md`](docs/hosts-updater-plan.md)  
- [`build/linux/README.md`](build/linux/README.md) / [`build/darwin/README.md`](build/darwin/README.md)  
- [`NOTICE`](NOTICE) / [`LICENSE`](LICENSE)  

## 已知缺口

- 本机无 NSIS 时安装包需另装 makensis  
- macOS 公证未做（`.dmg` 已产出）；真机托盘细调留待硬件  
- Linux AppImage 未完整捆绑 WebKit，目标机仍需桌面依赖  
- CIDR 深度扫描未做  
