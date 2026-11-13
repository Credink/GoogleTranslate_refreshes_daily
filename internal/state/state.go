// Package state persists config and last-check status under the user config directory.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const appDirName = "GoogleTranslateHostsUpdater"

type ScheduleMode string

const (
	ModeDailyMidnight ScheduleMode = "daily_midnight"
	ModeIntervalHours ScheduleMode = "interval_hours"
)

type Config struct {
	ScheduleMode         ScheduleMode `json:"schedule_mode"`
	IntervalHours        int          `json:"interval_hours"`
	AllowOnlineOnFailure bool         `json:"allow_online_on_failure"`
	IncludeIPv6          bool         `json:"include_ipv6"`
	Workers              int          `json:"workers"`
	TimeoutMS            int          `json:"timeout_ms"`
	Repeat               int          `json:"repeat"`
	Domains              []string     `json:"domains"`
}

func DefaultConfig() Config {
	return Config{
		ScheduleMode:         ModeDailyMidnight,
		IntervalHours:        24,
		AllowOnlineOnFailure: true,
		IncludeIPv6:          false,
		Workers:              64,
		TimeoutMS:            1500,
		Repeat:               3,
		Domains: []string{
			"translate.googleapis.com",
			"translate-pa.googleapis.com",
		},
	}
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
