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
	st := state.State{LastStatus: state.StatusNever}
	if !NeedsCheck(time.Now(), cfg, st) {
		t.Fatal("never checked should need check")
	}
}
