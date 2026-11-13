# 谷歌翻译 Hosts 每日更新器

跨平台桌面工具：按计划检测 `translate.googleapis.com` 可用 IP，**仅在当前 hosts 映射失效时**拉取远程 IP 库、测速选优并更新系统 hosts。  
技术栈：**Go + Wails v2**（设置窗口 + 系统托盘）。许可：**Apache-2.0**。

## 已锁定行为

| 项 | 默认 |
|----|------|
| Hosts 域名 | 默认 5 条：API×2 + `jnn-pa` + `.com` + `.com.hk`；可选地区 ccTLD 包（默认关） |
| 在线 IP 库 | **不定时**；仅当前 IP 探测失败且允许时拉取 |
| IPv6 | **关**；开启后先测本机 IPv6，不可用则本轮 IPv4；双栈优先 IPv4 |
| 开机自启 | **关**；设置页可开（Windows HKCU Run） |
| 权限模式 | **默认普通用户**；写 hosts 时按次提权。可选「以管理员身份运行」长期静默更新 |
| 许可 | Apache-2.0（`LICENSE` + `NOTICE`） |

IP 探测思路参考 [GoogleTranslate_IPFinder](https://github.com/GoodCoder666/GoogleTranslate_IPFinder)（**不复制其 GPL 源码**）。

## 环境（Windows）

- Go 1.23+（Wails v2.17 会拉 toolchain 1.26）
- Node.js + npm
- Wails CLI：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- WebView2
- （可选）[NSIS](https://nsis.sourceforge.io/)：`makensis` 用于生成安装包

```powershell
wails doctor
```

## 运行 / 打包

```powershell
cd d:\Code\Go\GoogleTranslate_refreshes_daily

# 开发
wails dev

# 测试
go test ./...

# 仅构建 exe
wails build

# 构建 exe + 便携 zip +（若有 NSIS）安装包
powershell -ExecutionPolicy Bypass -File .\scripts\package-windows.ps1

# 或单独生成 NSIS 安装包
wails build -nsis
```

产物：

| 路径 | 说明 |
|------|------|
| `build/bin/GoogleTranslateHostsUpdater.exe` | 主程序 |
| `build/bin/*-installer.exe` | NSIS 安装包（需 makensis） |
| `build/dist/*-portable.zip` | 便携包 |

关闭窗口会**隐藏到托盘**；托盘可「立即刷新 / 打开设置 / 退出」。

### 两种权限模式

| 模式 | 行为 |
|------|------|
| **普通用户（默认）** | 日常以非管理员运行；真正改 hosts（+ flushdns）时再弹 UAC / 系统授权 |
| **管理员模式（可选）** | 设置页勾选「以管理员身份运行」→ 保存后提权重启；之后换 IP 可长期静默写 hosts |

- 管理员模式可长期静默更新 hosts（换 IP 时不再每次点授权）
- 普通模式更安全，但每次真正改 hosts 都需要点击授予权限
- 偏好保存在 `config.json` 的 `run_as_admin`；启动时若勾了管理员但当前未提权，会自动尝试提权重启（可取消授权并继续普通运行）
- **Windows**：提权用 `ShellExecute runas`；关闭管理员模式时尝试用关联令牌降权重启；失败则请完全退出后普通再开
- **macOS / Linux**：提权分别为 osascript / pkexec 的 best-effort；降权通常需手动退出后普通再开

写入 hosts 时若无权限，会通过 UAC 拉起同进程 `--apply-hosts` 助手。成功写入后会**刷新系统 DNS 缓存**（Windows：`ipconfig /flushdns`，优先在提权助手中执行；macOS/Linux 为 best-effort）。

配置目录：`%AppData%\GoogleTranslateHostsUpdater\`（`config.json` / `state.json`）。

## 重要：Hosts ≠ 代理路径（ZeroOmega / 系统代理）

改 hosts **只影响本机 DNS 解析**。若 Chrome 里开了 **ZeroOmega / SwitchyOmega**（或其它代理插件）把流量送去 `127.0.0.1:7890` 之类，请求**不会走 hosts 指向的优选 IP**，翻译仍可能失败——这是「解析对了、路径错了」。

### 需要直连（DIRECT / bypass）的 5 个托管域名

| 域名 | 用途 |
|------|------|
| `translate.googleapis.com` | API / 测速 |
| `translate-pa.googleapis.com` | Chrome 网页翻译主调用 |
| `jnn-pa.googleapis.com` | Chrome 内置翻译相关 |
| `translate.google.com` | 网页入口 |
| `translate.google.com.hk` | 网页入口 |

在 ZeroOmega / SwitchyOmega 中任选其一：

- **情景模式（auto switch）**：为上述域名添加规则 → **`direct`**（且放在 `*google*` / `*googleapis*` → proxy 之类宽规则**之前**）
- **代理情景（FixedProfile，如走 7890）**：把上述域名加入 **bypassList**（绕过条件）

其它代理扩展（Clash 浏览器插件、系统 PAC 等）需**自行**配置同样的 DIRECT/绕过；**本工具不声称覆盖所有代理扩展**。

### 本工具的自动配置（best-effort）

写入 hosts 成功后（以及首次检查、或设置里点「配置代理插件绕过」）会尝试：

1. 扫描本机 Chrome / Edge 用户目录下的 **ZeroOmega / SwitchyOmega** `Local Extension Settings`（LevelDB）
2. **先只读检查**：若托管域名已在 bypass/DIRECT 且无 `ranslate` 等笔误 → **不写盘**（幂等）
3. 仅当缺失或笔误时才修补 FixedProfile bypass / SwitchProfile `direct` 规则
4. 域名列表未变且上次已确认就绪时，自动流程会直接跳过（避免每次运行重复改写）；手动按钮仍会重新检查，但同样只在需要时写入
5. 在界面显示结果；设置区有 **圆形感叹号**，悬停可见中文说明

限制：

- 浏览器正在运行时 LevelDB 可能锁住 → 请**完全退出**后再点「配置代理插件绕过」
- 写入后需**重启浏览器**插件才会重载配置
- 格式无法识别时跳过，不会强行改坏未知存储
- **不能**自动配置非 Omega 类扩展

### 浏览器仍打不开翻译？

1. **关闭浏览器「安全 DNS / Secure DNS / DoH」**（Chrome：设置 → 隐私和安全 → 使用安全 DNS；Edge 类似）。DoH **绕过**系统 hosts。  
2. 确认代理插件已对上面 5 域 **DIRECT/bypass**（见上节）；宽规则 `*google*` / `*googleapis*` → proxy 会把翻译送进 7890。  
3. 默认写入同一优选 IP 到上述 5 域（设置里可改；可选地区 ccTLD 包）。探测仍只验证 API。请收藏 **[translate.google.com](https://translate.google.com/)** 或 **[translate.google.com.hk](https://translate.google.com.hk/)**；`translate.google.cn` 只是跳板。  
4. 管理员执行：`ipconfig /flushdns`；`Resolve-DnsName translate.googleapis.com` 应等于 hosts 中的 IP。Chrome：`chrome://net-internals/#dns` → Clear host cache。  
5. 若 bypass 列表里误写成 `ranslate.…`，改回 `translate.…`（本工具重写时会尝试自动修正）。

### 开机自启动

1. **安装时：** NSIS 组件勾选「开机自启动（当前用户）」  
2. **设置页：** 勾选「开机自启动」→ 保存（写入 `HKCU\...\Run\GoogleTranslateHostsUpdater`）  
3. 卸载安装包或取消勾选并保存即可清除

## macOS（无本机 Mac 时的说明）

- 代码已按 darwin 路径对齐：模板托盘图标、`SetTemplateIcon`、LaunchAgent 自启、Wails `Mac` 选项（见 `build/darwin/`）。
- **完整菜单栏托盘 GUI 验收需要真机 Mac**；Windows 上无法做有意义的 darwin CGO/托盘交叉验证。

## 文档

- [`docs/hosts-updater-plan.md`](docs/hosts-updater-plan.md)
- [`build/darwin/README.md`](build/darwin/README.md)
- [`NOTICE`](NOTICE) / [`LICENSE`](LICENSE)

## 已知缺口

- 本机若未装 NSIS，需另装后再跑 `wails build -nsis`（脚本会回退到便携 zip）
- macOS 真机托盘 / 公证 / DMG 未做
- CIDR 深度扫描未做
