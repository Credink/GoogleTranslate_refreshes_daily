// Package probe implements HTTPS availability checks for Google Translate API IPs.
// Approach inspired by GoodCoder666/GoogleTranslate_IPFinder (GPL-3.0): reimplemented in Go.
package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultHost     = "translate.googleapis.com"
	DefaultTemplate = "https://%s/translate_a/element.js"
	DefaultTimeout  = 800 * time.Millisecond
	DefaultRepeat   = 2
	DefaultWorkers  = 32
	// RepeatGap between stable re-probes; keep short so daily single-IP checks stay snappy.
	RepeatGap = 250 * time.Millisecond
)

// Result is a successful probe of one IP.
type Result struct {
	IP      string
	Latency time.Duration
}

// Config controls a single-IP or pool speedtest.
type Config struct {
	Host     string
	Template string
	Timeout  time.Duration
	Repeat   int
	Workers  int
	// StopAfterSuccess cancels the pool scan after N successes (0 = probe entire pool).
	StopAfterSuccess int
	// PriorityKeep leaves the first N IPs unshuffled (e.g. last-known-good).
	PriorityKeep int
}

func (c Config) withDefaults() Config {
	if c.Host == "" {
		c.Host = DefaultHost
	}
	if c.Template == "" {
		c.Template = DefaultTemplate
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.Repeat <= 0 {
		c.Repeat = DefaultRepeat
	}
	if c.Workers <= 0 {
		c.Workers = DefaultWorkers
	}
	if c.Workers > 64 {
		c.Workers = 64
	}
	return c
}

func bracketIP(ip string) string {
	if strings.Contains(ip, ":") && !strings.HasPrefix(ip, "[") {
		return "[" + ip + "]"
	}
	return ip
}

// TestIP performs one HTTPS GET to template filled with ip, using SNI/Host = cfg.Host.
// Returns latency on success.
func TestIP(ctx context.Context, cfg Config, ip string) (time.Duration, error) {
	cfg = cfg.withDefaults()
	url := fmt.Sprintf(cfg.Template, bracketIP(ip))

	dialer := &net.Dialer{Timeout: cfg.Timeout}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			addr := net.JoinHostPort(ip, "443")
			return dialer.DialContext(ctx, "tcp", addr)
		},
		TLSClientConfig: &tls.Config{
			ServerName: cfg.Host,
			MinVersion: tls.VersionTLS12,
		},
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          0,
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: cfg.Timeout,
		IdleConnTimeout:       cfg.Timeout,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Host = cfg.Host
	req.Header.Set("Host", cfg.Host)

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	return latency, nil
}

// TestIPStable runs Repeat probes with a gap; fails if any attempt fails.
func TestIPStable(ctx context.Context, cfg Config, ip string) (time.Duration, error) {
	cfg = cfg.withDefaults()
	var total time.Duration
	for i := 0; i < cfg.Repeat; i++ {
		lat, err := TestIP(ctx, cfg, ip)
		if err != nil {
			return 0, err
		}
		total += lat
		if i+1 < cfg.Repeat {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(RepeatGap):
			}
		}
	}
	return total / time.Duration(cfg.Repeat), nil
}
