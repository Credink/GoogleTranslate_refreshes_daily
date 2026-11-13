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
}

func TestFitSmallLaptop(t *testing.T) {
	s := Fit(1366, 768)
	screenH := 768.0
	limit := int(screenH * 0.92)
	if s.Height > limit {
		t.Fatalf("height too tall for 768p work area: %+v", s)
	}
	if s.Width < AbsMinWidth && s.Width != 1366 {
		// must still be usable
		t.Fatalf("width too small: %+v", s)
	}
}

func TestFitTiny(t *testing.T) {
	s := Fit(800, 600)
	if s.Width > 800 || s.Height > 600 {
		t.Fatalf("must fit tiny screen: %+v", s)
	}
	if s.MinWidth > s.Width || s.MinHeight > s.Height {
		t.Fatalf("min > size on tiny: %+v", s)
	}
}

func TestFitZeroFallsBackBase(t *testing.T) {
	s := Fit(0, 0)
	if s.Width < AbsMinWidth || s.Height < AbsMinHeight {
		t.Fatalf("zero screen: %+v", s)
	}
}
