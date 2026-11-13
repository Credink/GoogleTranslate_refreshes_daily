//go:build windows

package autostart

import "testing"

func TestSupported(t *testing.T) {
	if !Supported() {
		t.Fatal("windows should support autostart")
	}
}

func TestEnableDisableRoundTrip(t *testing.T) {
	was, err := IsEnabled()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if was {
			_ = Enable()
		} else {
			_ = Disable()
		}
	})

	if err := Enable(); err != nil {
		t.Fatalf("enable: %v", err)
	}
	ok, err := IsEnabled()
	if err != nil || !ok {
		t.Fatalf("expected enabled, ok=%v err=%v", ok, err)
	}
	if err := Disable(); err != nil {
		t.Fatalf("disable: %v", err)
	}
	ok, err = IsEnabled()
	if err != nil || ok {
		t.Fatalf("expected disabled, ok=%v err=%v", ok, err)
	}
}
