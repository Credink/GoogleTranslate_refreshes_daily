// Package winsize picks a default Wails window size from the primary/current
// screen work area (logical pixels — DPI-aware on Windows/macOS/Linux).
package winsize

import "fmt"

// Fallback used when screen metrics are unavailable.
const (
	FallbackWidth  = 540
	FallbackHeight = 920
	AbsMinWidth    = 420
	AbsMinHeight   = 640
	AbsMaxWidth    = 960
	AbsMaxHeight   = 1400
	// Preferred content-fit size for the settings UI at 100% scale.
	ContentWidth  = 540
	ContentHeight = 920
)

// Size is a window geometry suggestion.
type Size struct {
	Width     int
	Height    int
	MinWidth  int
	MinHeight int
}

// Fit returns width/height/min sizes for a screen work area in logical pixels.
// Uses most of the available height so the settings form fits; clamps to the
// work area so small / high-DPI laptops are not cropped by an oversized frame.
func Fit(screenW, screenH int) Size {
	if screenW <= 0 {
		screenW = 1280
	}
	if screenH <= 0 {
		screenH = 800
	}

	// Prefer content-fit; otherwise take most of the work area.
	w := ContentWidth
	h := ContentHeight
	if screenW < ContentWidth+40 {
		w = int(float64(screenW) * 0.92)
	}
	if screenH < ContentHeight+80 {
		h = int(float64(screenH) * 0.92)
	}

	maxW := int(float64(screenW) * 0.95)
	maxH := int(float64(screenH) * 0.94)
	if maxW < 1 {
		maxW = screenW
	}
	if maxH < 1 {
		maxH = screenH
	}

	loW := AbsMinWidth
	if loW > maxW {
		loW = maxW
	}
	loH := AbsMinHeight
	if loH > maxH {
		loH = maxH
	}

	w = clamp(w, loW, min(AbsMaxWidth, maxW))
	h = clamp(h, loH, min(AbsMaxHeight, maxH))

	// Keep mins reachable on the chosen size; allow user resize down a bit.
	minW := clamp(AbsMinWidth, loW, w)
	minH := clamp(AbsMinHeight, loH, h)

	return Size{Width: w, Height: h, MinWidth: minW, MinHeight: minH}
}

// Fallback returns conservative defaults when screens cannot be queried.
func Fallback() Size {
	return Size{
		Width:     FallbackWidth,
		Height:    FallbackHeight,
		MinWidth:  AbsMinWidth,
		MinHeight: AbsMinHeight,
	}
}

// Describe is a short log line for the chosen geometry.
func Describe(sz Size, screenW, screenH int) string {
	return fmt.Sprintf("window=%dx%d min=%dx%d screen=%dx%d",
		sz.Width, sz.Height, sz.MinWidth, sz.MinHeight, screenW, screenH)
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
