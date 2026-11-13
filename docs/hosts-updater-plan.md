# 产品 / 技术计划（仓库副本）

权威副本：Project Context `docs/hosts-updater-plan.md`。

## 已锁定决策

1. **Hosts 域名：** `translate.googleapis.com` + `translate-pa.googleapis.com`
2. **在线 IP 库：** 不定时同步；**仅当当前 hosts IP 探测失败**时拉取远程列表再测速选优（`allow_online_on_failure`，默认开）
3. **IPv6：** 默认关；开启后每次检查先确认本机 IPv6 可用，否则本轮回退 IPv4；双栈都可用时**优先 IPv4**
4. **许可：** **Apache-2.0**（NOTICE 致谢 IPFinder 思路，不引入 GPL 源码）
5. **Windows 安装包：** `wails build -nsis`（可选「开机自启动」组件）+ `scripts/package-windows.ps1` 产出便携 zip
6. **Windows 开机自启：** HKCU Run 值 `GoogleTranslateHostsUpdater`；设置页 `launch_at_login`；卸载时清除
7. **macOS 托盘：** 仅做 build-tag / 模板图标 / 代码路径对齐；**无 Mac 硬件时不做完整 GUI 托盘验收**（见 `build/darwin/README.md`）

## 探测摘要

- URL：`https://{ip}/translate_a/element.js`，SNI/`Host` = `translate.googleapis.com`
- 当前 IP 仍可用 → 不写 hosts、不拉远程列表
- 调度：默认本地午夜；错过则启动补跑（只做可用性检查）

## 安装 / 自启（Windows）

```powershell
# 构建 exe +（若有 makensis）NSIS 安装包 + 便携 zip
powershell -ExecutionPolicy Bypass -File .\scripts\package-windows.ps1
```

- 安装包：勾选「开机自启动（当前用户）」即可写入登录启动项
- 已安装后：打开设置 → 勾选「开机自启动」→ 保存

## macOS 限制

本环境无 Mac：darwin 托盘用 `SetTemplateIcon` + 短标题结构化；全量 `wails`/托盘 GUI 验证留待真机。

## 后续

CIDR 深扫 UI、macOS 真机托盘验收与公证/DMG、Linux 托盘打磨。
