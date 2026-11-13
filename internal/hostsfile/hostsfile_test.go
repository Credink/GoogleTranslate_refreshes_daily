package hostsfile

import (
	"strings"
	"testing"
)

func TestApplyMarkerBlock(t *testing.T) {
	in := "127.0.0.1 localhost\n1.2.3.4 translate.googleapis.com\n"
	out, err := Apply(in, Mapping{IP: "8.8.8.8", Domains: DefaultDomains})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, BeginMarker) || !strings.Contains(out, EndMarker) {
		t.Fatalf("missing markers: %s", out)
	}
	if !strings.Contains(out, "8.8.8.8 translate.googleapis.com") {
		t.Fatalf("missing mapping: %s", out)
	}
	if !strings.Contains(out, "# 1.2.3.4 translate.googleapis.com") {
		t.Fatalf("expected conflict commented: %s", out)
	}
}

func TestCurrentIPFor(t *testing.T) {
	content := "9.9.9.9 translate.googleapis.com\n"
	if got := CurrentIPFor(content, "translate.googleapis.com"); got != "9.9.9.9" {
		t.Fatalf("got %q", got)
	}
}
