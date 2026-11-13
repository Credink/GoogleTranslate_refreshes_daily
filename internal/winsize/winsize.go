// Package winsize picks a default Wails window size from the primary/current
// screen work area (logical pixels — DPI-aware on Windows/macOS/Linux).
package winsize

// Fallback used when screen metrics are unavailable.
const (
	FallbackWidth  = 540
	FallbackHeight = 820
	AbsMinWidth    = 420
	AbsMinHeight   = 560
	AbsMaxWidth    = 960
	AbsMaxHeight   = 1280
	// Preferred content-fit size for the settings UI at 100% scale.
	ContentWidth  = 540
	ContentHeight = 820
)

// Size is a window geometry suggestion.
type Size struct {
	Width     int
	Height    int
	MinWidth  int
	MinHeight int
}

// Fit returns width/height/min sizes for a screen work area in logical pixels.
// Uses a fraction of available space, prefers content-fit when the display allows,
// and clamps to absolute min/max so common DPI setups are not cropped.
func Fit(screenW, screenH int) Size {
	if screenW <= 0 {
		screenW = 1280
	}
	if screenH <= 0 {
		screenH = 800
	}

	// Fractions of the work area (logical). Height-biased: form is taller than wide.
	w := int(float64(screenW) * 0.36)
	h := int(float64(screenH) * 0.82)

	// Prefer content-fit when the screen is large enough.
	if screenW >= ContentWidth+48 && w < ContentWidth {
		w = ContentWidth
	}
	if screenH >= ContentHeight+72 && h < ContentHeight {
		h = ContentHeight
	}

	maxW := int(float64(screenW) * 0.92)
	maxH := int(float64(screenH) * 0.90)
	if maxW < AbsMinWidth {
		maxW = screenW
	}
	if maxH < AbsMinHeight {
		maxH = screenH
	}

	w = clamp(w, AbsMinWidth, min(AbsMaxWidth, maxW))
	h = clamp(h, AbsMinHeight, min(AbsMaxHeight, maxH))

	minW := clamp(int(float64(screenW)*0.28), AbsMinWidth, w)
	minH := clamp(int(float64(screenH)*0.48), AbsMinHeight, h)
	if minW > w {
		minW = w
	}
	if minH > h {
		minH = h
	}

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
