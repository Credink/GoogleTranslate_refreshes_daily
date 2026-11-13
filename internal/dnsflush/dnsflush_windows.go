//go:build windows

package dnsflush

import (
	"fmt"
	"os/exec"
	"strings"
)

// flush runs `ipconfig /flushdns`. May require elevation on some Windows setups;
// callers should prefer invoking this from the elevated --apply-hosts helper.
func flush() error {
	cmd := exec.Command("ipconfig", "/flushdns")
	out, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg != "" {
			return fmt.Errorf("ipconfig /flushdns: %w (%s)", err, msg)
		}
		return fmt.Errorf("ipconfig /flushdns: %w", err)
	}
	return nil
}
