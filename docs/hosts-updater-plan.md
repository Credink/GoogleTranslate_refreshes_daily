# 产品 / 技术计划（仓库副本）

权威副本也保存在 Project Context：`docs/hosts-updater-plan.md`。

## 摘要

- **目标：** 每日校验谷歌翻译 API IP，失效则测速选优并更新 hosts；Wails 轻量 GUI + 托盘。
- **IP 参考：** 启发自 [GoogleTranslate_IPFinder](https://github.com/GoodCoder666/GoogleTranslate_IPFinder)（GPL-3.0）——Go 重实现，不整库复制。
- **探测：** `https://{ip}/translate_a/element.js`，TLS SNI / HTTP Host = `translate.googleapis.com`；默认 timeout 1.5s、repeat 3、高并发测速。
- **Hosts 写入域名：** `translate.googleapis.com`、`translate-pa.googleapis.com`（标记块 + 备份）。
- **调度：** 默认本地午夜；错过窗口则启动补跑一次。
- **IP 源：** 内置 gtdb 快照 + 可选在线同步；每日路径 =「测速」而非全量 CIDR 扫描。

## 待用户确认

1. 上述默认写入域名是否 OK？
2. 在线同步 IP 库是否默认开启？
3. IPv6 是否默认关闭？
4. 应用许可选 MIT/Apache 还是跟随 GPL-3.0？

更多细节见 Project store 同名计划文档。
