// Package discover builds the candidate IP pool (seed + optional online sync + last-good).
// Online list URLs follow GoodCoder666/GoogleTranslate_IPFinder / gtdb conventions.
package discover

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"GoogleTranslate_refreshes_daily/internal/ipseed"
)

// Default IPv4 list URL groups (try in order within each group).
var DefaultIPv4URLGroups = [][]string{
	{
		"https://raw.githubusercontent.com/GoodCoder666/gtdb/main/src/ip.txt",
	},
	{
		"https://raw.githubusercontent.com/Ponderfly/GoogleTranslateIpCheck/master/src/GoogleTranslateIpCheck/GoogleTranslateIpCheck/ip.txt",
	},
}

// PoolOptions configures candidate collection.
type PoolOptions struct {
	Online        bool
	IncludeIPv6   bool
	LastKnownGood string
	URLGroups     [][]string
	HTTPTimeout   time.Duration
}

// BuildPool returns unique candidate IPs. Seed is always included; online merge is optional.
func BuildPool(ctx context.Context, opt PoolOptions) ([]string, error) {
	seen := make(map[string]struct{})
	var out []string
	add := func(ip string) {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			return
		}
		parsed := net.ParseIP(ip)
		if parsed == nil {
			return
		}
		if parsed.To4() == nil && !opt.IncludeIPv6 {
			return
		}
		key := parsed.String()
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}

	if opt.LastKnownGood != "" {
		add(opt.LastKnownGood)
	}
	for _, ip := range ipseed.IPs() {
		add(ip)
	}

	if opt.Online {
		groups := opt.URLGroups
		if len(groups) == 0 {
			groups = DefaultIPv4URLGroups
		}
		timeout := opt.HTTPTimeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		merged, err := FetchURLGroups(ctx, groups, timeout)
		if err != nil {
			// Online failure is non-fatal if seed exists.
			if len(out) == 0 {
				return nil, err
			}
		} else {
			for _, ip := range merged {
				add(ip)
			}
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("empty IP candidate pool")
	}
	return out, nil
}

// FetchURLGroups tries each URL in a group until one succeeds; merges all groups.
func FetchURLGroups(ctx context.Context, groups [][]string, timeout time.Duration) ([]string, error) {
	client := &http.Client{Timeout: timeout}
	var all []string
	var lastErr error
	okGroups := 0
	for _, urls := range groups {
		ips, err := fetchFirstOK(ctx, client, urls)
		if err != nil {
			lastErr = err
			continue
		}
		okGroups++
		all = append(all, ips...)
	}
	if okGroups == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("no IP list URLs configured")
	}
	return all, nil
}

func fetchFirstOK(ctx context.Context, client *http.Client, urls []string) ([]string, error) {
	var lastErr error
	for _, u := range urls {
		ips, err := fetchIPList(ctx, client, u)
		if err != nil {
			lastErr = err
			continue
		}
		return ips, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("empty url group")
}

func fetchIPList(ctx context.Context, client *http.Client, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: status %d", url, resp.StatusCode)
	}
	return ParseIPList(resp.Body)
}

// ParseIPList reads whitespace-separated IPs from r.
func ParseIPList(r io.Reader) ([]string, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var out []string
	for sc.Scan() {
		for _, f := range strings.Fields(sc.Text()) {
			if net.ParseIP(f) != nil {
				out = append(out, f)
			}
		}
	}
	return out, sc.Err()
}
