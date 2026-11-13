// Package updater orchestrates discover → probe → hosts apply for one check cycle.
package updater

import (
	"context"
	"errors"
	"fmt"
	"time"

	"GoogleTranslate_refreshes_daily/internal/discover"
	"GoogleTranslate_refreshes_daily/internal/elevate"
	"GoogleTranslate_refreshes_daily/internal/hostsfile"
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
}

// Refresh runs one full check: build IP pool, speedtest, update hosts if needed.
func Refresh(ctx context.Context, cfg state.Config, st state.State) (Report, state.State) {
	now := time.Now()
	rep := Report{CheckedAt: now, Status: state.StatusFailed}

	pool, err := discover.BuildPool(ctx, discover.PoolOptions{
		Online:        cfg.OnlineSync,
		IncludeIPv6:   cfg.IncludeIPv6,
		LastKnownGood: st.LastIP,
	})
	if err != nil {
		rep.Message = "构建 IP 候选失败: " + err.Error()
		st = persist(st, rep)
		return rep, st
	}

	pcfg := probe.Config{
		Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond,
		Repeat:  cfg.Repeat,
		Workers: cfg.Workers,
	}

	// Fast path: if current hosts IP still works, skip rewrite.
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

	result, err := probe.Speedtest(ctx, pcfg, pool)
	if err != nil && !errors.Is(err, context.Canceled) {
		rep.Message = "测速失败: " + err.Error()
		st = persist(st, rep)
		return rep, st
	}
	best := result.BestIP()
	if best == "" {
		rep.Message = fmt.Sprintf("未找到可用 IP（失败 %d 个候选）", result.Failed)
		st = persist(st, rep)
		return rep, st
	}
	bestLat := result.Available[0].Latency
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

	if _, err := hostsfile.BackupAndWrite(hostsPath, newContent); err != nil {
		// BackupAndWrite may fail on write; try elevate path (same process write).
		if err2 := elevate.ApplyHostsPatch(hostsPath, newContent); err2 != nil {
			rep.Status = state.StatusFailed
			rep.Message = fmt.Sprintf("最佳 IP %s (%s)，但写入 hosts 失败: %v", best, rep.Latency, err2)
			st.LastIP = best
			st = persist(st, rep)
			return rep, st
		}
	}

	rep.Status = state.StatusUpdated
	rep.Updated = true
	rep.Message = fmt.Sprintf("已更新 hosts → %s (%s)", best, rep.Latency)
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
