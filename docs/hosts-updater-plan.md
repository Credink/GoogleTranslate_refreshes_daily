# 产品 / 技术计划（仓库副本）

权威副本：Project Context `docs/hosts-updater-plan.md`。

## 已锁定决策

1. **Hosts 域名（默认 5 条，同优选 IP）：** `translate.googleapis.com`、`translate-pa.googleapis.com`、`jnn-pa.googleapis.com`、`translate.google.com`、`translate.google.com.hk`。可选 `include_regional_pack`（默认关）追加 ccTLD 地区前端。探测仍仅 `translate.googleapis.com`。旧 2 域名配置加载时自动迁移到默认 5 条。详见 Context `docs/translate-domains.md`。
2. **在线 IP 库：** 不定时同步；**仅当当前 hosts IP 探测失败**时拉取远程列表再测速选优（`allow_online_on_failure`，默认开）
3. **IPv6：** 默认关；开启后每次检查先确认本机 IPv6 可用，否则本轮回退 IPv4；双栈都可用时**优先 IPv4**
4. **许可：** **Apache-2.0**（NOTICE 致谢 IPFinder 思路，不引入 GPL 源码）
5. **安装包产物：** 输出到 `packages/`。Windows=portable zip（可选 NSIS）；**macOS=`.dmg`**；**Linux=`.AppImage`**（DMG 仅为 macOS，Linux 用 AppImage 替代）
6. **开机自启：** Windows HKCU Run；macOS LaunchAgent；Linux XDG `~/.config/autostart/*.desktop`；设置页 `launch_at_login`
7. **macOS / Linux GUI：** 代码路径已齐；完整托盘验收需真机。Release 由 CI（`macos-14` / `ubuntu-latest`）构建（见 `build/darwin/README.md`、`build/linux/README.md`、`.github/workflows/release.yml`）
8. **双权限模式：** 默认普通用户（按次 `--apply-hosts` 提权）；可选 `run_as_admin` 长期提权静默写 hosts。Windows 完整 relaunch；macOS best-effort；**Linux 不支持整 GUI root**（只 `pkexec` 提权 `--apply-hosts`，AppImage 走 `$APPIMAGE`）
9. **代理插件 bypass（best-effort，幂等）：** 写入 hosts 后 / 首次检查 / GUI「配置代理插件绕过」会扫描 Chrome/Edge 的 ZeroOmega、SwitchyOmega LevelDB。**先检查是否已在 bypass/DIRECT**；已正确则跳过写入。仅缺失或 `ranslate` 等笔误时才修补。同域名集且上次已 OK 时自动流程不再重复改写。**不声称覆盖全部代理扩展**。详见 README「Hosts ≠ 代理路径」。

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

## macOS / Linux 构建

本环境无 Mac/Linux GUI 交叉能力：darwin/linux 托盘与自启代码路径已实现；全量 Wails GUI 由 CI 打 Release 包，真机验收留待硬件。端到端说明见 `docs/user-flow.md`。

## 权限模式（GUI）

- 设置页开关旁中文说明：管理员可长期静默更新；普通模式更安全但每次改 hosts 需授权
- 启动：`run_as_admin && !elevated` → 自动提权重启（握手文件防 UAC 取消误退出）
- Windows 降权：`TokenLinkedToken` + `CreateProcessAsUser`；失败则提示手动重启

## 后续

CIDR 深扫 UI、macOS 公证/DMG、Linux 发行版托盘宿主差异打磨；更多代理扩展的手动配置文档（仍不承诺自动覆盖）。
