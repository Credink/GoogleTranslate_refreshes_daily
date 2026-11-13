package winsize

import "testing"

func TestFit1080p(t *testing.T) {
	s := Fit(1920, 1080)
	if s.Width < ContentWidth || s.Height < ContentHeight {
		t.Fatalf("1080p should fit content, got %+v", s)
	}
	if s.Width > 1920 || s.Height > 1080 {
		t.Fatalf("exceeds screen: %+v", s)
	}
	if s.MinWidth > s.Width || s.MinHeight > s.Height {
		t.Fatalf("min > size: %+v", s)
	}
	t.Log(Describe(s, 1920, 1080))
}

func TestFitSmallLaptop(t *testing.T) {
	screenH := 768
	s := Fit(1366, screenH)
	limit := int(float64(screenH) * 0.94)
	if s.Height > limit {
		t.Fatalf("height too tall for 768p work area: %+v (limit %d)", s, limit)
	}
	if s.Width < AbsMinWidth && s.Width != 1366 {
		t.Fatalf("width too small: %+v", s)
	}
	// Must still leave room for scrolling UI when content is taller than work area.
	if s.Height >= ContentHeight {
		t.Fatalf("768p work area cannot fit full content height; got %+v", s)
	}
	t.Log(Describe(s, 1366, 768))
}

func TestFitTiny(t *testing.T) {
	s := Fit(800, 600)
	if s.Width > 800 || s.Height > 600 {
		t.Fatalf("must fit tiny screen: %+v", s)
	}
	if s.MinWidth > s.Width || s.MinHeight > s.Height {
		t.Fatalf("min > size on tiny: %+v", s)
	}
	t.Log(Describe(s, 800, 600))
}

func TestFitZeroFallsBackBase(t *testing.T) {
	s := Fit(0, 0)
	if s.Width < AbsMinWidth || s.Height < AbsMinHeight {
		t.Fatalf("zero screen: %+v", s)
	}
	t.Log(Describe(s, 0, 0))
}
