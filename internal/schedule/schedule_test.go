package schedule

import (
	"testing"
	"time"

	"GoogleTranslate_refreshes_daily/internal/state"
)

func TestNeedsCheckMissedMidnight(t *testing.T) {
	loc := time.Local
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, loc)
	st := state.State{
		LastCheckAt: time.Date(2026, 11, 12, 0, 5, 0, 0, loc),
		LastStatus:  state.StatusOK,
	}
	cfg := state.DefaultConfig()
	cfg.ScheduleMode = state.ModeDailyMidnight
	if !NeedsCheck(now, cfg, st) {
		t.Fatal("expected missed midnight to need check")
	}
	st.LastCheckAt = time.Date(2026, 11, 13, 0, 1, 0, 0, loc)
	if NeedsCheck(now, cfg, st) {
		t.Fatal("same local day should not need check")
	}
}

func TestNeedsCheckNever(t *testing.T) {
	cfg := state.DefaultConfig()
	cfg.ScheduleMode = state.ModeDailyMidnight
	st := state.State{LastStatus: state.StatusNever}
	if !NeedsCheck(time.Now(), cfg, st) {
		t.Fatal("never checked should need check")
	}
}

func TestOnAppStartNeedsCheckFalse(t *testing.T) {
	cfg := state.DefaultConfig()
	if cfg.ScheduleMode != state.ModeOnAppStart {
		t.Fatalf("factory default want on_app_start, got %q", cfg.ScheduleMode)
	}
	st := state.State{LastStatus: state.StatusNever}
	if NeedsCheck(time.Now(), cfg, st) {
		t.Fatal("on_app_start must not trigger timer-driven NeedsCheck")
	}
	st.LastCheckAt = time.Now().Add(-time.Hour)
	st.LastStatus = state.StatusOK
	if NeedsCheck(time.Now(), cfg, st) {
		t.Fatal("on_app_start timer path must stay idle after a prior check")
	}
}

func TestShouldCheckOnStartup(t *testing.T) {
	cfg := state.DefaultConfig()
	st := state.State{
		LastCheckAt: time.Now(),
		LastStatus:  state.StatusOK,
	}
	if !ShouldCheckOnStartup(time.Now(), cfg, st) {
		t.Fatal("on_app_start should always check on software start")
	}

	cfg.ScheduleMode = state.ModeDailyMidnight
	loc := time.Local
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, loc)
	st.LastCheckAt = time.Date(2026, 11, 13, 0, 1, 0, 0, loc)
	if ShouldCheckOnStartup(now, cfg, st) {
		t.Fatal("same-day daily mode should not check again on start")
	}
	st.LastCheckAt = time.Date(2026, 11, 12, 0, 1, 0, 0, loc)
	if !ShouldCheckOnStartup(now, cfg, st) {
		t.Fatal("missed day should check on start for daily mode")
	}
}
