//go:build darwin

package dnsflush

import (
	"fmt"
	"os/exec"
)

// flush: dscacheutil + mDNSResponder (best-effort; may need privileges).
func flush() error {
	if out, err := exec.Command("dscacheutil", "-flushcache").CombinedOutput(); err != nil {
		return fmt.Errorf("dscacheutil -flushcache: %w (%s)", err, string(out))
	}
	// Ignore killall failure — older/newer macOS variants differ.
	_ = exec.Command("killall", "-HUP", "mDNSResponder").Run()
	return nil
}
