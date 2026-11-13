# 谷歌翻译 Hosts 每日更新器

跨平台桌面工具：按计划检测 `translate.googleapis.com` 可用 IP，必要时安全更新系统 hosts。  
技术栈：**Go 核心 + Wails v2 GUI**（托盘/设置/状态；Windows 优先）。

## 功能（MVP）

- 启动时若错过本地午夜（或从未检查）→ 自动测速一次
- 默认每天本地午夜检查；可改为按小时间隔
- HTTPS 探测（SNI + `Host` + `/translate_a/element.js`），多并发、多轮确认
- Hosts 使用标记块写入，冲突行注释，写前备份
- 简体中文设置窗口；系统托盘菜单为预留桩（后续接入）

## IP 查找参考

探测与 IP 源策略**参考**（非复制源码）长期维护项目：

- [GoodCoder666/GoogleTranslate_IPFinder](https://github.com/GoodCoder666/GoogleTranslate_IPFinder)（GPL-3.0）
- IP 库：[gtdb](https://github.com/GoodCoder666/gtdb)、可选 Ponderfly 列表

本仓库以 Go **重实现**测速逻辑；详见 `NOTICE` 与产品计划。

## 环境要求（Windows）

- Go 1.23+（本机可用 Go 1.24 / toolchain 1.26）
- Node.js + npm（Wails 前端）
- [Wails CLI](https://wails.io) v2：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- WebView2（Win10/11 通常已带）

```powershell
wails doctor
```

## 构建与运行

```powershell
cd d:\Code\Go\GoogleTranslate_refreshes_daily

# 开发（热重载）
wails dev

# 仅测 Go 包
go test ./...

# 生产构建
wails build
```

产物在 `build/bin/`。**写入 hosts 需要管理员权限**；无权限时会提示提权（当前 MVP：请以管理员运行或手动复制 IP）。

## 配置与状态

用户目录：`%AppData%\GoogleTranslateHostsUpdater\`

- `config.json` — 频率、是否在线同步 IP、是否 IPv6 等
- `state.json` — 上次检查时间/结果/IP

## 仓库文档

- 产品/技术方案摘要：[`docs/hosts-updater-plan.md`](docs/hosts-updater-plan.md)
- 第三方致谢：[`NOTICE`](NOTICE)

## 待办（下一阶段）

- 完整系统托盘（Windows/macOS/Linux）
- 一键 UAC / `pkexec` 提权助手（无需整进程管理员）
- 可选 CIDR 深度扫描（IPFinder「扫描」模式，非每日默认）
- 正式安装包（NSIS / DMG / AppImage）

## 许可

应用代码拟采用宽松许可（待确认 MIT/Apache-2.0）。第三方参考与数据源见 `NOTICE`。
