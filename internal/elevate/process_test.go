package elevate

import "testing"

func TestModeExplanation(t *testing.T) {
	if ModeExplanation == "" {
		t.Fatal("ModeExplanation empty")
	}
	if CurrentModeLabel() == "" {
		t.Fatal("CurrentModeLabel empty")
	}
	if PlatformNotes() == "" {
		t.Fatal("PlatformNotes empty")
	}
	n := NewRelaunchNonce()
	if n == "" {
		t.Fatal("nonce empty")
	}
}
