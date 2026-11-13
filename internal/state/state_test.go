package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"GoogleTranslate_refreshes_daily/internal/hostsfile"
)

func TestDefaultConfigScheduleOnAppStart(t *testing.T) {
	c := DefaultConfig()
	if c.ScheduleMode != ModeOnAppStart {
		t.Fatalf("factory ScheduleMode want %q, got %q", ModeOnAppStart, c.ScheduleMode)
	}
}

func TestEffectiveDomainsRegionalOff(t *testing.T) {
	c := DefaultConfig()
	got := EffectiveDomains(c)
	if len(got) != len(hostsfile.DefaultDomains) {
		t.Fatalf("want %d domains, got %d: %v", len(hostsfile.DefaultDomains), len(got), got)
	}
	for i, d := range hostsfile.DefaultDomains {
		if got[i] != d {
			t.Fatalf("index %d: want %q got %q", i, d, got[i])
		}
	}
}

func TestEffectiveDomainsRegionalOn(t *testing.T) {
	c := DefaultConfig()
	c.IncludeRegionalPack = true
	got := EffectiveDomains(c)
	want := len(hostsfile.DefaultDomains) + len(hostsfile.RegionalDomains)
	if len(got) != want {
		t.Fatalf("want %d domains, got %d", want, len(got))
	}
	last := got[len(got)-1]
	if last != hostsfile.RegionalDomains[len(hostsfile.RegionalDomains)-1] {
		t.Fatalf("expected regional pack appended, last=%q", last)
	}
}

func TestMigrateLegacyTwoDomainConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "GoogleTranslateHostsUpdater")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	legacy := map[string]any{
		"schedule_mode":           "daily_midnight",
		"interval_hours":          24,
		"allow_online_on_failure": true,
		"domains":                 hostsfile.LegacyDefaultDomains,
	}
	b, _ := json.MarshalIndent(legacy, "", "  ")
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), b, 0644); err != nil {
		t.Fatal(err)
	}

	c, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !sameDomainList(c.Domains, hostsfile.DefaultDomains) {
		t.Fatalf("expected migrated defaults, got %v", c.Domains)
	}
	if c.IncludeRegionalPack {
		t.Fatal("regional pack should stay off after migration")
	}
}
