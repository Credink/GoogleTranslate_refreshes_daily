# 谷歌翻译 Hosts 每日更新器

跨平台桌面工具：按计划检测 `translate.googleapis.com` 可用 IP，**仅在当前 hosts 映射失效时**拉取远程 IP 库、测速选优并更新系统 hosts。  
技术栈：**Go + Wails v2**（设置窗口 + 系统托盘）。许可：**Apache-2.0**。

## 已锁定行为

| 项 | 默认 |
|----|------|
| Hosts 域名 | `translate.googleapis.com`、`translate-pa.googleapis.com` |
| 在线 IP 库 | **不定时**；仅当前 IP 探测失败且允许时拉取 |
| IPv6 | **关**；开启后先测本机 IPv6，不可用则本轮 IPv4；双栈优先 IPv4 |
| 开机自启 | **关**；设置页可开（Windows HKCU Run） |
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

关闭窗口会**隐藏到托盘**；托盘可「立即刷新 / 打开设置 / 退出」。写入 hosts 时若无权限，会通过 UAC 拉起同进程 `--apply-hosts` 助手。成功写入后会**刷新系统 DNS 缓存**（Windows：`ipconfig /flushdns`，优先在提权助手中执行；macOS/Linux 为 best-effort）。

配置目录：`%AppData%\GoogleTranslateHostsUpdater\`（`config.json` / `state.json`）。

### 浏览器仍打不开翻译？

1. **关闭浏览器「安全 DNS / Secure DNS / DoH」**（Chrome：设置 → 隐私和安全 → 使用安全 DNS；Edge 类似）。DoH **绕过**系统 hosts。  
2. 本工具只写入 **API** 域名：`translate.googleapis.com`、`translate-pa.googleapis.com`。网页入口请优先用 **[translate.google.cn](https://translate.google.cn/)**；`translate.google.com` 前端 IP/证书与 API 不同，**不要**把 API IP 硬绑到该域名。  
3. 可手动管理员执行：`ipconfig /flushdns`，再用 `Resolve-DnsName translate.googleapis.com` 确认结果为 hosts 中的 IP。

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
