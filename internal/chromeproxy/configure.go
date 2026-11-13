// Package chromeproxy best-effort patches ZeroOmega / SwitchyOmega storage so
// managed Google Translate hosts use DIRECT (bypass local proxies like :7890).
//
// Coverage is intentionally incomplete: only known Omega-format chrome.storage
// LevelDB layouts are rewritten. Other proxy extensions must be configured manually.
//
// Idempotency: profiles are read first; LevelDB is written only when a managed
// domain is missing from bypass/DIRECT or a typo (e.g. ranslate) is found.
package chromeproxy

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Status codes for UI / state persistence.
const (
	StatusOK      = "ok"      // patched or already correct
	StatusSkipped = "skipped" // nothing found / unsupported / auto-skip
	StatusPartial = "partial" // some DBs ok, some failed
	StatusFailed  = "failed"  // found candidates but all failed
	StatusNever   = "never"
)

// Report is the aggregate result of ConfigureBypass.
type Report struct {
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	At          time.Time `json:"at"`
	Found       int       `json:"found"`
	Changed     int       `json:"changed"`
	Failed      int       `json:"failed"`
	Details     []string  `json:"details,omitempty"`
	RestartHint bool      `json:"restart_hint"`
	DomainsKey  string    `json:"domains_key,omitempty"`
	SkippedAuto bool      `json:"skipped_auto,omitempty"`
}

// Options controls ConfigureBypass behavior.
type Options struct {
	// Force always re-scans LevelDB (still idempotent: no write if already OK).
	Force bool
	// SkipIfDomainsKey, when non-empty and equal to DomainsKey(domains) and
	// PriorOK is true, skips opening any LevelDB (automatic path).
	SkipIfDomainsKey string
	// PriorOK means a previous run already reported StatusOK for SkipIfDomainsKey.
	PriorOK bool
}

// TooltipZH is shown near proxy/hosts settings (circle + exclamation).
const TooltipZH = "本软件已尝试自动配置 ZeroOmega 等 Chrome 代理插件，将托管的翻译域名设为直连（DIRECT/绕过代理）。若翻译仍失败，请在代理插件中手动把这些域名设为 bypass/DIRECT。本功能无法覆盖全部代理扩展。"

// DomainsKey is a stable fingerprint of the managed domain list (for skip-if-unchanged).
func DomainsKey(domains []string) string {
	return strings.Join(normalizeDomains(domains), ",")
}

// ConfigureBypass discovers Omega-like extensions and patches bypass / DIRECT rules.
func ConfigureBypass(domains []string) Report {
	return ConfigureBypassOpts(domains, Options{Force: true})
}

// ConfigureBypassOpts is ConfigureBypass with automatic-skip controls.
func ConfigureBypassOpts(domains []string, opt Options) Report {
	now := time.Now()
	rep := Report{At: now, Status: StatusSkipped}

	domains = normalizeDomains(domains)
	rep.DomainsKey = DomainsKey(domains)
	if len(domains) == 0 {
		rep.Message = "无托管域名，跳过代理插件配置"
		return rep
	}

	if !opt.Force && opt.PriorOK && opt.SkipIfDomainsKey != "" && opt.SkipIfDomainsKey == rep.DomainsKey {
		rep.Status = StatusOK
		rep.SkippedAuto = true
		rep.Message = "托管域名未变且上次已确认 bypass/DIRECT，跳过重复写入"
		return rep
	}

	hits := DiscoverOmegaExtensions()
	rep.Found = len(hits)
	if len(hits) == 0 {
		diag := DiagnoseDiscovery()
		rep.Message = formatNotFoundMessage(diag)
		if len(diag.RootsExisting) > 0 {
			rep.Details = append(rep.Details, "已找到浏览器配置目录:")
			rep.Details = append(rep.Details, diag.RootsExisting...)
		} else {
			rep.Details = append(rep.Details, "未找到任何 Chrome/Chromium/Brave/Edge 用户数据目录")
			if diag.HomeDir != "" {
				rep.Details = append(rep.Details, "探测 home: "+diag.HomeDir)
			}
		}
		if len(diag.ProfileDirs) > 0 {
			rep.Details = append(rep.Details, "已扫描配置档: "+strings.Join(shortPaths(diag.ProfileDirs, 6), "; "))
		}
		return rep
	}

	var details []string
	alreadyOK := 0
	for _, hit := range hits {
		label := fmt.Sprintf("%s/%s %s (%s)", hit.Browser, hit.Profile, hit.Name, shortID(hit.ExtensionID))
		res, err := patchStorageDir(hit.StorageDir, domains, false)
		if err != nil {
			rep.Failed++
			details = append(details, label+": "+err.Error())
			continue
		}
		if res.Skipped {
			details = append(details, label+": "+res.Message)
			continue
		}
		line := label + ": " + res.Message
		if len(res.Details) > 0 {
			line += " [" + strings.Join(res.Details, "; ") + "]"
		}
		details = append(details, line)
		if res.Changed {
			rep.Changed++
			rep.RestartHint = true
		} else {
			alreadyOK++
		}
	}

	rep.Details = details
	switch {
	case rep.Changed > 0 && rep.Failed == 0:
		rep.Status = StatusOK
		rep.Message = fmt.Sprintf("已修补 %d 个配置库（缺失/笔误）；另有 %d 个已是 DIRECT/bypass", rep.Changed, alreadyOK)
		if rep.RestartHint {
			rep.Message += "；请完全重启浏览器后生效"
		}
	case rep.Changed > 0 && rep.Failed > 0:
		rep.Status = StatusPartial
		rep.Message = fmt.Sprintf("部分成功：修补 %d，失败 %d（共发现 %d）", rep.Changed, rep.Failed, rep.Found)
	case rep.Failed > 0 && alreadyOK == 0:
		rep.Status = StatusFailed
		rep.Message = fmt.Sprintf("发现 %d 个配置库但未能访问（浏览器可能正在运行）", rep.Found)
	case rep.Failed > 0:
		rep.Status = StatusPartial
		rep.Message = fmt.Sprintf("已确认 %d 个就绪，%d 个无法访问", alreadyOK, rep.Failed)
	default:
		rep.Status = StatusOK
		rep.Message = fmt.Sprintf("已检查 %d 个配置：托管域名已在 bypass/DIRECT，未写入", rep.Found)
	}
	return rep
}

func normalizeDomains(domains []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:6] + "…"
}

func formatNotFoundMessage(diag DiscoveryDiagnostics) string {
	base := "未检测到 ZeroOmega / SwitchyOmega 的扩展存储（Local Extension Settings）"
	if len(diag.RootsExisting) == 0 {
		return base + "：未找到 Chrome/Chromium/Brave/Edge 配置目录。请确认浏览器已登录运行过，或使用 Flatpak/Snap 时已打开过扩展页；其他代理插件需手动设 DIRECT/绕过"
	}
	if len(diag.ProfileDirs) == 0 {
		return base + "：已找到浏览器目录但无 Default/Profile 配置档"
	}
	return base + "：已扫描 " + itoa(len(diag.ProfileDirs)) + " 个配置档。请安装 ZeroOmega（Chrome 商店 ID pfneded… / Edge 商店 dmaldh…），完全退出浏览器后点「配置代理插件绕过」；其他代理插件需手动设 DIRECT/绕过"
}

func shortPaths(paths []string, max int) []string {
	if max <= 0 || len(paths) <= max {
		return paths
	}
	out := append([]string{}, paths[:max]...)
	out = append(out, "…(+"+itoa(len(paths)-max)+")")
	return out
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
