package autostart

import (
	"strings"
	"testing"
)

func TestDesktopExecEscape(t *testing.T) {
	if got := desktopExecEscape("/opt/app/bin"); got != "/opt/app/bin" {
		t.Fatalf("plain: %q", got)
	}
	if got := desktopExecEscape(`/opt/my app/bin`); got != `"/opt/my app/bin"` {
		t.Fatalf("space: %q", got)
	}
}

func TestDesktopEntryContainsExec(t *testing.T) {
	body := desktopEntry("/usr/local/bin/GoogleTranslateHostsUpdater")
	if !strings.Contains(body, "Exec=/usr/local/bin/GoogleTranslateHostsUpdater") {
		t.Fatalf("missing Exec:\n%s", body)
	}
	if !strings.Contains(body, "Type=Application") {
		t.Fatalf("missing Type:\n%s", body)
	}
	if !strings.Contains(body, "X-GNOME-Autostart-enabled=true") {
		t.Fatal("missing X-GNOME-Autostart-enabled")
	}
}
