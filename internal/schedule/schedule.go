// Package schedule decides when checks should run (on-app-start / midnight / interval + missed window).
package schedule

import (
	"time"

	"GoogleTranslate_refreshes_daily/internal/state"
)

// NeedsCheck reports whether a timer-driven check should run now.
// ModeOnAppStart always returns false here — refresh is triggered only from app startup
// (see ShouldCheckOnStartup), not by the background schedule loop.
func NeedsCheck(now time.Time, cfg state.Config, st state.State) bool {
	if cfg.ScheduleMode == state.ModeOnAppStart {
		return false
	}
	if st.LastCheckAt.IsZero() || st.LastStatus == state.StatusNever {
		return true
	}
	switch cfg.ScheduleMode {
	case state.ModeIntervalHours:
		hours := cfg.IntervalHours
		if hours <= 0 {
			hours = 24
		}
		return now.Sub(st.LastCheckAt) >= time.Duration(hours)*time.Hour
	default: // daily_midnight
		ly, lm, ld := st.LastCheckAt.In(now.Location()).Date()
		ny, nm, nd := now.Date()
		return ly != ny || lm != nm || ld != nd
	}
}

// ShouldCheckOnStartup reports whether a check should run when the process starts.
func ShouldCheckOnStartup(now time.Time, cfg state.Config, st state.State) bool {
	if cfg.ScheduleMode == state.ModeOnAppStart {
		return true
	}
	return NeedsCheck(now, cfg, st)
}

// NextRun returns the next scheduled time after now.
// For ModeOnAppStart there is no wall-clock next run; callers should treat the
// returned time as a long poll wake (settings may change) and skip the refresh
// via NeedsCheck.
func NextRun(now time.Time, cfg state.Config) time.Time {
	loc := now.Location()
	switch cfg.ScheduleMode {
	case state.ModeOnAppStart:
		// Re-read config periodically; NeedsCheck stays false so no spurious refresh.
		return now.Add(24 * time.Hour)
	case state.ModeIntervalHours:
		hours := cfg.IntervalHours
		if hours <= 0 {
			hours = 24
		}
		return now.Add(time.Duration(hours) * time.Hour)
	default:
		// Next local midnight
		y, m, d := now.Date()
		midnightToday := time.Date(y, m, d, 0, 0, 0, 0, loc)
		if now.After(midnightToday) || now.Equal(midnightToday) {
			return midnightToday.Add(24 * time.Hour)
		}
		return midnightToday
	}
}
