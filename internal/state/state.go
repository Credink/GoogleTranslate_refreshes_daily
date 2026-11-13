// Package state persists config and last-check status under the user config directory.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"GoogleTranslate_refreshes_daily/internal/hostsfile"
)

const appDirName = "GoogleTranslateHostsUpdater"

type ScheduleMode string

const (
	ModeOnAppStart    ScheduleMode = "on_app_start" // 软件启动时刷新（出厂默认）
	ModeDailyMidnight ScheduleMode = "daily_midnight"
	ModeIntervalHours ScheduleMode = "interval_hours"
)

type Config struct {
	ScheduleMode         ScheduleMode `json:"schedule_mode"`
	IntervalHours        int          `json:"interval_hours"`
	AllowOnlineOnFailure bool         `json:"allow_online_on_failure"`
	IncludeIPv6          bool         `json:"include_ipv6"`
	LaunchAtLogin        bool         `json:"launch_at_login"`
	RunAsAdmin           bool         `json:"run_as_admin"` // prefer long-lived elevated process
	IncludeRegionalPack  bool         `json:"include_regional_pack"`
	Workers              int          `json:"workers"`
	TimeoutMS            int          `json:"timeout_ms"`
	Repeat               int          `json:"repeat"`
	Domains              []string     `json:"domains"`
}

func DefaultConfig() Config {
	return Config{
		ScheduleMode:         ModeOnAppStart,
		IntervalHours:        24,
		AllowOnlineOnFailure: true,
		IncludeIPv6:          false,
		LaunchAtLogin:        false,
		RunAsAdmin:           false, // default: normal user; elevate only for hosts write
		IncludeRegionalPack:  false,
		Workers:              32,
		TimeoutMS:            800,
		Repeat:               2,
		Domains:              cloneStrings(hostsfile.DefaultDomains),
	}
}

// EffectiveDomains returns the hosts list to write: base Domains plus optional regional pack.
func EffectiveDomains(c Config) []string {
	base := c.Domains
	if len(base) == 0 {
		base = hostsfile.DefaultDomains
	}
	out := cloneStrings(base)
	if !c.IncludeRegionalPack {
		return out
	}
	seen := make(map[string]struct{}, len(out))
	for _, d := range out {
		seen[strings.ToLower(d)] = struct{}{}
	}
	for _, d := range hostsfile.RegionalDomains {
		key := strings.ToLower(d)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, d)
	}
	return out
}

type Status string

const (
	StatusOK      Status = "ok"
	StatusUpdated Status = "updated"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
	StatusNever   Status = "never"
)

type State struct {
	LastCheckAt time.Time         `json:"last_check_at"`
	LastStatus  Status            `json:"last_status"`
	LastMessage string            `json:"last_message"`
	LastIP      string            `json:"last_ip"`
	LastLatency string            `json:"last_latency,omitempty"`
	LastIPs     map[string]string `json:"last_ips,omitempty"`

	// Best-effort ZeroOmega / SwitchyOmega bypass auto-config (chrome.storage LevelDB).
	ProxyBypassAt         time.Time `json:"proxy_bypass_at,omitempty"`
	ProxyBypassStatus     string    `json:"proxy_bypass_status,omitempty"`
	ProxyBypassMessage    string    `json:"proxy_bypass_message,omitempty"`
	ProxyBypassTried      bool      `json:"proxy_bypass_tried,omitempty"`
	ProxyBypassDomainsKey string    `json:"proxy_bypass_domains_key,omitempty"` // fingerprint; skip auto rewrite when unchanged + OK
}

func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, appDirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func configPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

func statePath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "state.json"), nil
}

func LoadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c := DefaultConfig()
			_ = SaveConfig(c)
			return c, nil
		}
		return DefaultConfig(), err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(b, &c); err != nil {
		return DefaultConfig(), err
	}
	// Migrate deprecated online_sync → allow_online_on_failure when new key absent.
	var raw map[string]json.RawMessage
	if json.Unmarshal(b, &raw) == nil {
		if _, hasNew := raw["allow_online_on_failure"]; !hasNew {
			if old, ok := raw["online_sync"]; ok {
				var v bool
				if json.Unmarshal(old, &v) == nil {
					c.AllowOnlineOnFailure = v
				}
			}
		}
	}
	if len(c.Domains) == 0 {
		c.Domains = DefaultConfig().Domains
	} else if sameDomainList(c.Domains, hostsfile.LegacyDefaultDomains) {
		// Old 2-domain installs pick up the expanded default set on load.
		c.Domains = DefaultConfig().Domains
		_ = SaveConfig(c)
	}
	return c, nil
}

func SaveConfig(c Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func LoadState() (State, error) {
	path, err := statePath()
	if err != nil {
		return State{LastStatus: StatusNever}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{LastStatus: StatusNever}, nil
		}
		return State{LastStatus: StatusNever}, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return State{LastStatus: StatusNever}, err
	}
	if s.LastStatus == "" {
		s.LastStatus = StatusNever
	}
	return s, nil
}

func SaveState(s State) error {
	path, err := statePath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func cloneStrings(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func sameDomainList(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(strings.TrimSpace(a[i]), strings.TrimSpace(b[i])) {
			return false
		}
	}
	return true
}
