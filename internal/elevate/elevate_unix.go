//go:build !windows

package elevate

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

// IsElevated reports whether the process is running as root (euid 0).
func IsElevated() bool {
	return os.Geteuid() == 0
}

func relaunchElevated(exe, args string) error {
	switch runtime.GOOS {
	case "darwin":
		cmdPart := shellQuote(exe)
		if args != "" {
			cmdPart += " " + args
		}
		// Background so osascript returns after auth instead of waiting on the GUI.
		script := fmt.Sprintf(`do shell script "%s >/dev/null 2>&1 &" with administrator privileges`, cmdPart)
		cmd := exec.Command("osascript", "-e", script)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %v (%s); %s", ErrNeedElevation, err, string(out), Hint())
		}
		return nil
	default:
		argv := []string{exe}
		if args != "" {
			// args is "--elevated-relaunch <nonce>"
			fields := splitArgs(args)
			argv = append(argv, fields...)
		}
		cmd := exec.Command("pkexec", argv...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("%w: %v; %s", ErrNeedElevation, err, Hint())
		}
		return nil
	}
}

func splitArgs(s string) []string {
	var out []string
	var cur string
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case c == ' ' && !inQuote:
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
		default:
			cur += string(c)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func relaunchUnelevated(exe string) error {
	// Dropping root privileges for a GUI relaunch is unreliable across
	// desktop environments. Ask the user to restart normally.
	_ = exe
	return fmt.Errorf("当前平台无法从管理员自动降权重启")
}

func startDetached(exe string) error {
	cmd := exec.Command(exe)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
