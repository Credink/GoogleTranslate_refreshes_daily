// Package updater orchestrates discover → probe → hosts apply for one check cycle.
package updater

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"GoogleTranslate_refreshes_daily/internal/chromeproxy"
	"GoogleTranslate_refreshes_daily/internal/discover"
	"GoogleTranslate_refreshes_daily/internal/dnsflush"
	"GoogleTranslate_refreshes_daily/internal/elevate"
	"GoogleTranslate_refreshes_daily/internal/hostsfile"
	"GoogleTranslate_refreshes_daily/internal/netcheck"
	"GoogleTranslate_refreshes_daily/internal/probe"
	"GoogleTranslate_refreshes_daily/internal/state"
)

// Report is the outcome of Refresh.
type Report struct {
	Status    state.Status `json:"status"`
	Message   string       `json:"message"`
	BestIP    string       `json:"best_ip"`
	Latency   string       `json:"latency"`
	Updated   bool         `json:"updated"`
	CheckedAt time.Time    `json:"checked_at"`
	UsedIPv6  bool         `json:"used_ipv6"`
	Online    bool         `json:"online_fetch"`
}

// Refresh runs one availability check. Online IP list fetch happens only after
// the current hosts IP fails (and AllowOnlineOnFailure is true), and only if the
// local seed pool cannot yield a working IP.
func Refresh(ctx context.Context, cfg state.Config, st state.State) (Report, state.State) {
	now := time.Now()
	rep := Report{CheckedAt: now, Status: state.StatusFailed}

	stableCfg := probe.Config{
		Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond,
		Repeat:  cfg.Repeat,
		Workers: cfg.Workers,
	}
	// Pool scan: single attempt per IP, early-exit on first success, keep last-known first.
	scanCfg := probe.Config{
		Timeout:          time.Duration(cfg.TimeoutMS) * time.Millisecond,
		Repeat:           1,
		Workers:          cfg.Workers,
		StopAfterSuccess: 1,
		PriorityKeep:     1,
	}

	useIPv6 := false
	if cfg.IncludeIPv6 {
		useIPv6 = netcheck.IPv6Usable(ctx)
	}
	rep.UsedIPv6 = useIPv6

	hostsPath := hostsfile.Path()
	if content, err := hostsfile.Read(hostsPath); err == nil {
		cur := hostsfile.CurrentIPFor(content, probe.DefaultHost)
		if cur != "" {
			if lat, err := probe.TestIPStable(ctx, stableCfg, cur); err == nil {
				rep.Status = state.StatusOK
				rep.BestIP = cur
				rep.Latency = lat.Round(time.Millisecond).String()
				rep.Message = fmt.Sprintf("当前 hosts IP %s 可用 (%s)，无需更新", cur, rep.Latency)
				st.LastIP = cur
				// First-run / never-tried: still attempt Omega bypass config once.
				if !st.ProxyBypassTried {
					st = applyProxyBypass(st, state.EffectiveDomains(cfg), &rep)
				}
				st = persist(st, rep)
				return rep, st
			}
		}
	}

	// Current mapping failed (or missing) → local seed first (no remote sync).
	best, bestLat, failedLocal, err := findBest(ctx, scanCfg, discover.PoolOptions{
		Online:        false,
		IncludeIPv6:   useIPv6,
		LastKnownGood: st.LastIP,
	})
	if err != nil {
		rep.Message = "构建 IP 候选失败: " + err.Error()
		st = persist(st, rep)
		return rep, st
	}

	online := false
	failed := failedLocal
	if best == "" && cfg.AllowOnlineOnFailure {
		online = true
		rep.Online = true
		var failedOnline int
		best, bestLat, failedOnline, err = findBest(ctx, scanCfg, discover.PoolOptions{
			Online:        true,
			SkipSeed:      true, // local seed already exhausted
			IncludeIPv6:   useIPv6,
			LastKnownGood: st.LastIP,
		})
		failed += failedOnline
		if err != nil && best == "" {
			rep.Message = "构建/测速 IP 候选失败: " + err.Error()
			st = persist(st, rep)
			return rep, st
		}
	}

	if best == "" {
		rep.Message = fmt.Sprintf("未找到可用 IP（失败 %d 个候选）", failed)
		st = persist(st, rep)
		return rep, st
	}

	// Confirm winner once more before writing hosts.
	if lat, err := probe.TestIP(ctx, stableCfg, best); err == nil {
		bestLat = lat
	}

	rep.BestIP = best
	rep.Latency = bestLat.Round(time.Millisecond).String()

	domains := state.EffectiveDomains(cfg)

	content, err := hostsfile.Read(hostsPath)
	if err != nil {
		content = ""
	}
	newContent, err := hostsfile.Apply(content, hostsfile.Mapping{IP: best, Domains: domains})
	if err != nil {
		rep.Message = "生成 hosts 内容失败: " + err.Error()
		st = persist(st, rep)
		return rep, st
	}

	if err := writeHosts(hostsPath, newContent); err != nil {
		rep.Status = state.StatusFailed
		rep.Message = fmt.Sprintf("最佳 IP %s (%s)，但写入 hosts 失败: %v", best, rep.Latency, err)
		st.LastIP = best
		st = persist(st, rep)
		return rep, st
	}
	// Best-effort OS DNS cache flush (Windows: ipconfig /flushdns). Elevated
	// --apply-hosts also flushes; this covers non-elevated writes.
	dnsflush.AfterHostsWrite()

	rep.Status = state.StatusUpdated
	rep.Updated = true
	msg := fmt.Sprintf("已更新 hosts → %s (%s)", best, rep.Latency)
	if online {
		msg += "（本地候选不可用，已按需拉取远程 IP 库）"
	}
	if cfg.IncludeIPv6 && !useIPv6 {
		msg += "；本机 IPv6 不可用，本轮仅用 IPv4"
	}
	rep.Message = msg
	st.LastIP = best
	if st.LastIPs == nil {
		st.LastIPs = map[string]string{}
	}
	for _, d := range domains {
		st.LastIPs[d] = best
	}
	// After hosts write: auto-configure ZeroOmega/SwitchyOmega DIRECT/bypass (best-effort).
	st = applyProxyBypass(st, domains, &rep)
	st = persist(st, rep)
	return rep, st
}

// applyProxyBypass patches Omega chrome.storage LevelDB and records status on st/rep.
// Automatic path is idempotent: if last status was OK for the same domain set, skip
// entirely; otherwise read LevelDB and write only when bypass/DIRECT is missing or typo'd.
func applyProxyBypass(st state.State, domains []string, rep *Report) state.State {
	pr := chromeproxy.ConfigureBypassOpts(domains, chromeproxy.Options{
		Force:            false,
		PriorOK:          st.ProxyBypassStatus == chromeproxy.StatusOK,
		SkipIfDomainsKey: st.ProxyBypassDomainsKey,
	})
	st.ProxyBypassTried = true
	st.ProxyBypassAt = pr.At
	st.ProxyBypassStatus = pr.Status
	st.ProxyBypassMessage = pr.Message
	if pr.DomainsKey != "" {
		st.ProxyBypassDomainsKey = pr.DomainsKey
	}
	if pr.Message != "" && !pr.SkippedAuto {
		rep.Message = strings.TrimSpace(rep.Message + "；代理插件: " + pr.Message)
	} else if pr.SkippedAuto {
		// Keep hosts message clean on routine runs; status still stored.
	}
	return st
}

func findBest(ctx context.Context, scanCfg probe.Config, opt discover.PoolOptions) (best string, lat time.Duration, failed int, err error) {
	pool, err := discover.BuildPool(ctx, opt)
	if err != nil {
		return "", 0, 0, err
	}
	result, err := probe.Speedtest(ctx, scanCfg, pool)
	if err != nil && !errors.Is(err, context.Canceled) && len(result.Available) == 0 {
		return "", 0, result.Failed, err
	}
	ip, latency := result.BestIPPreferV4()
	return ip, latency, result.Failed, nil
}

func writeHosts(hostsPath, newContent string) error {
	if _, err := hostsfile.BackupAndWrite(hostsPath, newContent); err == nil {
		return nil
	} else if elevate.CanWriteHosts(hostsPath) {
		return elevate.ApplyHostsPatch(hostsPath, newContent)
	}

	dir, err := state.Dir()
	if err != nil {
		dir, err = os.MkdirTemp("", "gthu-hosts-*")
		if err != nil {
			return err
		}
	}
	patchDir := filepath.Join(dir, "elevate")
	_ = os.MkdirAll(patchDir, 0755)
	patch := filepath.Join(patchDir, fmt.Sprintf("hosts-%d.patch", time.Now().UnixNano()))
	if err := os.WriteFile(patch, []byte(newContent), 0600); err != nil {
		return err
	}
	err = elevate.ApplyElevated(patch)
	_ = os.Remove(patch)
	_ = os.Remove(patch + ".done")
	return err
}

func persist(st state.State, rep Report) state.State {
	st.LastCheckAt = rep.CheckedAt
	st.LastStatus = rep.Status
	st.LastMessage = rep.Message
	st.LastLatency = rep.Latency
	if rep.BestIP != "" {
		st.LastIP = rep.BestIP
	}
	_ = state.SaveState(st)
	return st
}
