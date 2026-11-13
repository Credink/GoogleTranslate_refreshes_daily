# 谷歌翻译 Hosts 每日更新器

跨平台桌面工具：按计划检测 `translate.googleapis.com` 可用 IP，**仅在当前 hosts 映射失效时**拉取远程 IP 库、测速选优并更新系统 hosts。  
技术栈：**Go + Wails v2**（设置窗口 + 系统托盘）。许可：**Apache-2.0**。

## 已锁定行为

| 项 | 默认 |
|----|------|
| Hosts 域名 | `translate.googleapis.com`、`translate-pa.googleapis.com` |
| 在线 IP 库 | **不定时**；仅当前 IP 探测失败且允许时拉取 |
| IPv6 | **关**；开启后先测本机 IPv6，不可用则本轮 IPv4；双栈优先 IPv4 |
| 许可 | Apache-2.0（`LICENSE` + `NOTICE`） |

IP 探测思路参考 [GoogleTranslate_IPFinder](https://github.com/GoodCoder666/GoogleTranslate_IPFinder)（**不复制其 GPL 源码**）。

## 环境（Windows）

- Go 1.23+（Wails v2.17 会拉 toolchain 1.26）
- Node.js + npm
- Wails CLI：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- WebView2

```powershell
wails doctor
```

## 运行

```powershell
cd d:\Code\Go\GoogleTranslate_refreshes_daily

# 开发
wails dev

# 测试
go test ./...

# 构建
wails build
```

产物：`build/bin/GoogleTranslateHostsUpdater.exe`。  
关闭窗口会**隐藏到托盘**；托盘可「立即刷新 / 打开设置 / 退出」。写入 hosts 时若无权限，会通过 UAC 拉起同进程 `--apply-hosts` 助手。

配置目录：`%AppData%\GoogleTranslateHostsUpdater\`（`config.json` / `state.json`）。

## 文档

- [`docs/hosts-updater-plan.md`](docs/hosts-updater-plan.md)
- [`NOTICE`](NOTICE) / [`LICENSE`](LICENSE)

## 已知缺口

- macOS 上 energye/systray 与 Wails 可能冲突（Windows 优先验证）
- 安装包 / 自动启动项未做
- CIDR 深度扫描未做
