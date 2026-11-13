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

// Speedtest probes all IPs concurrently (IPFinder-style warm-start worker pool).
// Each IP must pass Repeat successful probes; order of ips is shuffled.
func Speedtest(ctx context.Context, cfg Config, ips []string) (SpeedtestResult, error) {
	cfg = cfg.withDefaults()
	if len(ips) == 0 {
		return SpeedtestResult{}, nil
	}

	shuffled := append([]string(nil), ips...)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	sem := make(chan struct{}, cfg.Workers)
	var (
		mu   sync.Mutex
		out  SpeedtestResult
		wg   sync.WaitGroup
		errC = make(chan error, 1)
	)

	warmupDelay := 2 * cfg.Timeout / time.Duration(cfg.Workers)
	if warmupDelay < time.Millisecond {
		warmupDelay = time.Millisecond
	}

	for i, ip := range shuffled {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			lat, err := TestIPStable(ctx, cfg, ip)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				out.Failed++
				return
			}
			out.Available = append(out.Available, Result{IP: ip, Latency: lat})
		}(ip)

		if i < cfg.Workers {
			select {
			case <-ctx.Done():
			case <-time.After(warmupDelay):
			}
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		<-done
		return out, ctx.Err()
	case err := <-errC:
		return out, err
	}

	// Sort ascending by latency
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
