// Package schedule decides when checks should run (midnight / interval + missed window).
package schedule

import (
	"time"

	"GoogleTranslate_refreshes_daily/internal/state"
)

// NeedsCheck reports whether a check should run now given config and last state.
func NeedsCheck(now time.Time, cfg state.Config, st state.State) bool {
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

// NextRun returns the next scheduled time after now.
func NextRun(now time.Time, cfg state.Config) time.Time {
	loc := now.Location()
	switch cfg.ScheduleMode {
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
