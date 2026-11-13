# 使用流程（端到端）

面向日常使用的简要指南。

## 1. 安装与运行

| 平台 | 做法 |
|------|------|
| **Windows** | 解压便携 zip 运行 `GoogleTranslateHostsUpdater.exe`，或运行 NSIS 安装包 |
| **Linux (x86_64)** | 下载 **`.AppImage`**，`chmod +x` 后运行；需桌面环境（WebKitGTK）。**不是 DMG**（DMG 仅为 macOS） |
| **macOS (Apple Silicon)** | 打开 **`.dmg`**，拖到「应用程序」；首次若被拦截，在「系统设置 → 隐私与安全性」允许 |

关闭窗口 = **隐藏到托盘**；托盘「退出程序」才真正结束。托盘可：立即刷新 / 打开设置 / 退出。

配置目录：

- Windows：`%AppData%\GoogleTranslateHostsUpdater\`
- macOS：`~/Library/Application Support/GoogleTranslateHostsUpdater/`
- Linux：`~/.config/GoogleTranslateHostsUpdater/`

## 2. 权限模式

| 模式 | 行为 |
|------|------|
| **普通用户（默认）** | 改 hosts 时按次提权（Windows UAC / macOS 密码 / Linux pkexec） |
| **管理员模式（可选）** | 设置勾选「以管理员身份运行」→ 提权重启后，换 IP 可静默写 hosts |

关闭管理员模式后若无法自动降权：完全退出，再以普通方式打开。

## 3. 计划检查与手动刷新

- 默认：**每天本地午夜**检查一次；错过则启动时补跑
- 可改为按小时间隔
- 托盘或主界面「立即刷新」随时手动跑
- **仅当当前 hosts IP 探测失败**时才拉远程 IP 库并换优（不定时同步）

## 4. Hosts 域名与 flushdns

默认写入同一优选 IP 的 **5** 个域名：

1. `translate.googleapis.com`（探测 / API）
2. `translate-pa.googleapis.com`（Chrome 网页翻译主调用）
3. `jnn-pa.googleapis.com`
4. `translate.google.com`
5. `translate.google.com.hk`

可选「地区前端」ccTLD 包（默认关）。设置里可改域名列表。

写入成功后会刷新系统 DNS 缓存：

- Windows：`ipconfig /flushdns`（提权助手内优先执行）
- macOS：`dscacheutil -flushcache` + `mDNSResponder` HUP
- Linux：`resolvectl flush-caches` / `nscd` 等（best-effort）

## 5. Chrome：右键网页翻译 vs 翻译网站

| 目标 | 说明 |
|------|------|
| **右键 / 自动「翻译成中文」** | 依赖 hosts 里的 `translate-pa` / `translate.googleapis.com` 等；**这是本工具主要要修的路径** |
| **打开 translate.google.com** | 网页入口；默认也写入 `.com` / `.com.hk`。勿依赖 `translate.google.cn`（仅跳板） |

海外原文页打不开 ≠ 翻译坏了：原文页常需代理；翻译 API 应用 **DIRECT**（见下）。

## 6. ZeroOmega / 代理：DIRECT 绕过

改 hosts **只改本机解析**。若代理插件把流量送进 `127.0.0.1:7890`，请求**不会**走 hosts 优选 IP。

对上述 5 域配置 **DIRECT / bypass**（宽规则 `*google*` → proxy 之前）。

本工具会（best-effort、幂等）：

1. 扫描 Chrome/Edge 的 ZeroOmega / SwitchyOmega LevelDB
2. 已正确则**不写盘**；仅缺失或 `ranslate` 等笔误才修补
3. 界面圆形感叹号悬停可见中文说明；也可点「配置代理插件绕过」

限制：浏览器占用 LevelDB 时请先退出；写入后需重启浏览器；不覆盖非 Omega 类扩展。

推荐心智模型：

- **海外页面**：走 Socks5（如 `socks5://127.0.0.1:7890`）
- **翻译相关 5 域**：**DIRECT**（走 hosts 优选 IP）

## 7. 安全 DNS 与浏览器 DNS 缓存

1. **关闭** Chrome/Edge「使用安全 DNS / Secure DNS / DoH」——DoH 会**绕过**系统 hosts  
2. 系统侧已 flushdns 后，再在 Chrome：`chrome://net-internals/#dns` → Clear host cache  
3. 确认：`Resolve-DnsName` / `dig` / `getent hosts` 看到的 IP = hosts 中的优选 IP

## 8. 开机自启

| 平台 | 机制 |
|------|------|
| Windows | HKCU Run；NSIS 可选组件；设置页开关 |
| macOS | `~/Library/LaunchAgents/…plist` |
| Linux | `~/.config/autostart/*.desktop`（XDG） |

设置页勾选「开机自启动」→ 保存即可。

## 9. 注意事项（速查）

- IPv6 默认关；开了会先探本机 IPv6，双栈优先 IPv4  
- 无管理员权限时无法写 `/etc/hosts` 或 Windows hosts  
- 代理插件未 DIRECT → hosts「看起来对了」翻译仍失败  
- Secure DNS 开着 → hosts 无效  
- Linux/macOS 完整 GUI 托盘请用对应平台 Release 包；跨平台交叉编译 GUI 需 CI/真机
