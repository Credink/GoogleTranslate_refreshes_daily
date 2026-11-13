//go:build !windows && !darwin

package dnsflush

import (
	"fmt"
	"os"
	"os/exec"
)

// flush: best-effort systemd-resolved / nscd / resolvectl on Linux and friends.
func flush() error {
	var errs []string
	try := func(name string, args ...string) {
		if _, err := exec.LookPath(name); err != nil {
			return
		}
		if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v (%s)", name, err, string(out)))
		}
	}
	try("resolvectl", "flush-caches")
	try("systemd-resolve", "--flush-caches")
	try("nscd", "-i", "hosts")
	// Some distros expose a drop-in script.
	if _, err := os.Stat("/etc/init.d/nscd"); err == nil {
		try("/etc/init.d/nscd", "restart")
	}
	if len(errs) == 0 {
		// No known flusher present — not an error for hosts apply.
		return nil
	}
	// If at least one tool existed but all failed, surface the first.
	return fmt.Errorf("%s", errs[0])
}
