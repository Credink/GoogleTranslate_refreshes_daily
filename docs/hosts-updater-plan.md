# 产品 / 技术计划（仓库副本）

权威副本：Project Context `docs/hosts-updater-plan.md`。

## 已锁定决策

1. **Hosts 域名：** `translate.googleapis.com` + `translate-pa.googleapis.com`
2. **在线 IP 库：** 不定时同步；**仅当当前 hosts IP 探测失败**时拉取远程列表再测速选优（`allow_online_on_failure`，默认开）
3. **IPv6：** 默认关；开启后每次检查先确认本机 IPv6 可用，否则本轮回退 IPv4；双栈都可用时**优先 IPv4**
4. **许可：** **Apache-2.0**（NOTICE 致谢 IPFinder 思路，不引入 GPL 源码）

## 探测摘要

- URL：`https://{ip}/translate_a/element.js`，SNI/`Host` = `translate.googleapis.com`
- 当前 IP 仍可用 → 不写 hosts、不拉远程列表
- 调度：默认本地午夜；错过则启动补跑（只做可用性检查）

## 后续

正式安装包、CIDR 深扫 UI、macOS 托盘与 Wails 共存打磨。
