//go:build !windows && !darwin

package dnsflush

import (
	"fmt"
	"os"
	"os/exec"
)

// flush: best-effort systemd-resolved / nscd / NetworkManager on Linux and friends.
func flush() error {
	var errs []string
	var ran bool
	try := func(name string, args ...string) {
		if _, err := exec.LookPath(name); err != nil {
			return
		}
		ran = true
		if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v (%s)", name, err, string(out)))
		}
	}
	try("resolvectl", "flush-caches")
	try("systemd-resolve", "--flush-caches")
	try("nscd", "-i", "hosts")
	// NetworkManager hosts cache (some distros).
	try("nmcli", "general", "reload")
	if _, err := os.Stat("/etc/init.d/nscd"); err == nil {
		try("/etc/init.d/nscd", "restart")
	}
	if !ran {
		// No known flusher present — not an error for hosts apply.
		return nil
	}
	if len(errs) == 0 {
		return nil
	}
	// If every invoked tool failed, surface the first.
	return fmt.Errorf("%s", errs[0])
}
