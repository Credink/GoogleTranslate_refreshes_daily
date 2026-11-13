// Package updater orchestrates discover → probe → hosts apply for one check cycle.
package updater

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"GoogleTranslate_refreshes_daily/internal/discover"
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
// the current hosts IP fails (and AllowOnlineOnFailure is true).
func Refresh(ctx context.Context, cfg state.Config, st state.State) (Report, state.State) {
	now := time.Now()
	rep := Report{CheckedAt: now, Status: state.StatusFailed}

	pcfg := probe.Config{
		Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond,
		Repeat:  cfg.Repeat,
		Workers: cfg.Workers,
	}

	useIPv6 := false
	if cfg.IncludeIPv6 {
		useIPv6 = netcheck.IPv6Usable(ctx)
		if !useIPv6 {
			// Keep going on IPv4 for this run.
		}
	}
	rep.UsedIPv6 = useIPv6

	hostsPath := hostsfile.Path()
	if content, err := hostsfile.Read(hostsPath); err == nil {
		cur := hostsfile.CurrentIPFor(content, probe.DefaultHost)
		if cur != "" {
			if lat, err := probe.TestIPStable(ctx, pcfg, cur); err == nil {
				rep.Status = state.StatusOK
				rep.BestIP = cur
				rep.Latency = lat.Round(time.Millisecond).String()
				rep.Message = fmt.Sprintf("当前 hosts IP %s 可用 (%s)，无需更新", cur, rep.Latency)
				st.LastIP = cur
				st = persist(st, rep)
				return rep, st
			}
		}
	}

	// Current mapping failed (or missing) → build pool; online only if allowed.
	online := cfg.AllowOnlineOnFailure
	rep.Online = online
	pool, err := discover.BuildPool(ctx, discover.PoolOptions{
		Online:        online,
		IncludeIPv6:   useIPv6,
		LastKnownGood: st.LastIP,
	})
	if err != nil {
		rep.Message = "构建 IP 候选失败: " + err.Error()
		st = persist(st, rep)
		return rep, st
	}

	result, err := probe.Speedtest(ctx, pcfg, pool)
	if err != nil && !errors.Is(err, context.Canceled) {
		rep.Message = "测速失败: " + err.Error()
		st = persist(st, rep)
		return rep, st
	}
	best, bestLat := result.BestIPPreferV4()
	if best == "" {
		rep.Message = fmt.Sprintf("未找到可用 IP（失败 %d 个候选）", result.Failed)
		st = persist(st, rep)
		return rep, st
	}
	rep.BestIP = best
	rep.Latency = bestLat.Round(time.Millisecond).String()

	domains := cfg.Domains
	if len(domains) == 0 {
		domains = hostsfile.DefaultDomains
	}

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

	rep.Status = state.StatusUpdated
	rep.Updated = true
	msg := fmt.Sprintf("已更新 hosts → %s (%s)", best, rep.Latency)
	if online {
		msg += "（已按需拉取远程 IP 库）"
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
	st = persist(st, rep)
	return rep, st
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
