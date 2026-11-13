package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"GoogleTranslate_refreshes_daily/internal/dnsflush"
	"GoogleTranslate_refreshes_daily/internal/hostsfile"
)

const maxApplyHostsBytes = 2 << 20 // 2 MiB — full hosts rewrite, not just the marker block

// runApplyHosts is the elevated helper entry: write patch file contents to system hosts.
// Only accepts patches under the app elevate dir (or gthu-hosts-* temp) and content
// that includes this app's managed markers — never an arbitrary path as root/admin.
func runApplyHosts(patchPath string) int {
	abs, err := filepath.Abs(patchPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "patch path: %v\n", err)
		return 1
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	if !patchPathAllowed(abs) {
		fmt.Fprintf(os.Stderr, "refusing --apply-hosts path outside app elevate/temp dirs: %s\n", abs)
		return 1
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read patch: %v\n", err)
		return 1
	}
	if err := validateApplyHostsContent(raw); err != nil {
		fmt.Fprintf(os.Stderr, "invalid patch: %v\n", err)
		return 1
	}
	path := hostsfile.Path()
	if _, err := hostsfile.BackupAndWrite(path, string(raw)); err != nil {
		if err2 := os.WriteFile(path, raw, 0644); err2 != nil {
			fmt.Fprintf(os.Stderr, "write hosts: %v / %v\n", err, err2)
			return 2
		}
	}
	// Elevated process can flush DNS on Windows where a normal user may not.
	dnsflush.AfterHostsWrite()
	_ = os.WriteFile(abs+".done", []byte("ok\n"), 0644)
	return 0
}

// patchPathAllowed restricts elevated reads to this app's patch locations so a
// crafted argv cannot point --apply-hosts at arbitrary files after UAC/pkexec.
func patchPathAllowed(abs string) bool {
	base := filepath.Base(abs)
	if !strings.HasPrefix(base, "hosts-") || !strings.HasSuffix(base, ".patch") {
		return false
	}
	dir := filepath.Dir(abs)
	if filepath.Base(dir) == "elevate" && filepath.Base(filepath.Dir(dir)) == "GoogleTranslateHostsUpdater" {
		return true
	}
	// updater.writeHosts fallback: os.MkdirTemp("", "gthu-hosts-*")
	if strings.HasPrefix(filepath.Base(dir), "gthu-hosts-") {
		return true
	}
	return false
}

func validateApplyHostsContent(raw []byte) error {
	if len(raw) == 0 {
		return fmt.Errorf("empty")
	}
	if len(raw) > maxApplyHostsBytes {
		return fmt.Errorf("too large (%d bytes)", len(raw))
	}
	if strings.ContainsRune(string(raw), 0) {
		return fmt.Errorf("contains NUL")
	}
	s := string(raw)
	if !strings.Contains(s, hostsfile.BeginMarker) || !strings.Contains(s, hostsfile.EndMarker) {
		return fmt.Errorf("missing managed hosts markers")
	}
	return nil
}
