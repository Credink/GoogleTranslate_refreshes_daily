package main

import (
	"path/filepath"
	"strings"
	"testing"

	"GoogleTranslate_refreshes_daily/internal/hostsfile"
)

func TestPatchPathAllowed(t *testing.T) {
	ok := filepath.Join(string(filepath.Separator), "home", "u", ".config", "GoogleTranslateHostsUpdater", "elevate", "hosts-123.patch")
	if !patchPathAllowed(filepath.Clean(ok)) {
		t.Fatalf("expected allow %s", ok)
	}
	tempOK := filepath.Join(string(filepath.Separator), "tmp", "gthu-hosts-xyz", "hosts-1.patch")
	if !patchPathAllowed(filepath.Clean(tempOK)) {
		t.Fatalf("expected allow temp %s", tempOK)
	}
	bad := []string{
		filepath.Join(string(filepath.Separator), "etc", "passwd"),
		filepath.Join(string(filepath.Separator), "tmp", "hosts-1.patch"),
		filepath.Join(string(filepath.Separator), "home", "u", ".config", "GoogleTranslateHostsUpdater", "elevate", "evil.txt"),
		filepath.Join(string(filepath.Separator), "home", "u", ".config", "OtherApp", "elevate", "hosts-1.patch"),
	}
	for _, p := range bad {
		if patchPathAllowed(filepath.Clean(p)) {
			t.Fatalf("expected deny %s", p)
		}
	}
}

func TestValidateApplyHostsContent(t *testing.T) {
	good := hostsfile.BeginMarker + "\n1.2.3.4 translate.googleapis.com\n" + hostsfile.EndMarker + "\n"
	if err := validateApplyHostsContent([]byte(good)); err != nil {
		t.Fatal(err)
	}
	if err := validateApplyHostsContent([]byte("no markers")); err == nil {
		t.Fatal("expected marker error")
	}
	if err := validateApplyHostsContent([]byte("a\x00b" + hostsfile.BeginMarker + hostsfile.EndMarker)); err == nil {
		t.Fatal("expected NUL error")
	}
	huge := []byte(strings.Repeat("x", maxApplyHostsBytes+1))
	if err := validateApplyHostsContent(huge); err == nil {
		t.Fatal("expected size error")
	}
}
