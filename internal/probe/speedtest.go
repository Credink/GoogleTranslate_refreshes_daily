package probe

import (
	"context"
	"math/rand"
	"net"
	"sync"
	"time"
)

// SpeedtestResult aggregates available IPs sorted by latency (caller may sort).
type SpeedtestResult struct {
	Available []Result
	Failed    int
}

// Speedtest probes IPs concurrently with a bounded worker pool.
// The first PriorityKeep addresses stay at the front (unshuffled); the rest are shuffled.
// When StopAfterSuccess > 0, scanning cancels after that many successes (early-exit).
func Speedtest(ctx context.Context, cfg Config, ips []string) (SpeedtestResult, error) {
	cfg = cfg.withDefaults()
	if len(ips) == 0 {
		return SpeedtestResult{}, nil
	}

	ordered := orderCandidates(ips, cfg.PriorityKeep)

	runCtx := ctx
	var cancel context.CancelFunc
	if cfg.StopAfterSuccess > 0 {
		runCtx, cancel = context.WithCancel(ctx)
		defer cancel()
	}

	jobs := make(chan string)
	var (
		mu  sync.Mutex
		out SpeedtestResult
		wg  sync.WaitGroup
	)

	workerCount := cfg.Workers
	if workerCount > len(ordered) {
		workerCount = len(ordered)
	}
	if workerCount < 1 {
		workerCount = 1
	}

	worker := func() {
		defer wg.Done()
		for ip := range jobs {
			if runCtx.Err() != nil {
				return
			}
			lat, err := TestIPStable(runCtx, cfg, ip)
			mu.Lock()
			if err != nil {
				out.Failed++
				mu.Unlock()
				continue
			}
			out.Available = append(out.Available, Result{IP: ip, Latency: lat})
			n := len(out.Available)
			mu.Unlock()
			if cancel != nil && cfg.StopAfterSuccess > 0 && n >= cfg.StopAfterSuccess {
				cancel()
				return
			}
		}
	}

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}

sendLoop:
	for _, ip := range ordered {
		select {
		case <-runCtx.Done():
			break sendLoop
		case jobs <- ip:
		}
	}
	close(jobs)
	wg.Wait()

	if err := ctx.Err(); err != nil && len(out.Available) == 0 {
		return out, err
	}

	avail := out.Available
	for i := 0; i < len(avail); i++ {
		for j := i + 1; j < len(avail); j++ {
			if avail[j].Latency < avail[i].Latency {
				avail[i], avail[j] = avail[j], avail[i]
			}
		}
	}
	out.Available = avail
	return out, nil
}

func orderCandidates(ips []string, priorityKeep int) []string {
	out := append([]string(nil), ips...)
	if priorityKeep < 0 {
		priorityKeep = 0
	}
	if priorityKeep > len(out) {
		priorityKeep = len(out)
	}
	rest := out[priorityKeep:]
	rand.Shuffle(len(rest), func(i, j int) {
		rest[i], rest[j] = rest[j], rest[i]
	})
	return out
}

// BestIP returns the lowest-latency available IP, or empty string.
func (r SpeedtestResult) BestIP() string {
	if len(r.Available) == 0 {
		return ""
	}
	return r.Available[0].IP
}

// BestIPPreferV4 returns the best IPv4 when any IPv4 works; otherwise best IPv6.
func (r SpeedtestResult) BestIPPreferV4() (ip string, latency time.Duration) {
	var best4, best6 Result
	var has4, has6 bool
	for _, a := range r.Available {
		parsed := net.ParseIP(a.IP)
		if parsed == nil {
			continue
		}
		if parsed.To4() != nil {
			if !has4 || a.Latency < best4.Latency {
				best4, has4 = a, true
			}
			continue
		}
		if !has6 || a.Latency < best6.Latency {
			best6, has6 = a, true
		}
	}
	if has4 {
		return best4.IP, best4.Latency
	}
	if has6 {
		return best6.IP, best6.Latency
	}
	return "", 0
}
