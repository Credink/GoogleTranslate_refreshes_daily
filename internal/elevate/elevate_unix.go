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
		// Linux: never relaunch the full GUI/tray as root — energye/systray D-Bus
		// panics with a nil Conn, and AppImage+FUSE is hostile to root. Hosts
		// writes elevate only via --apply-hosts (pkexec).
		_ = exe
		_ = args
		return fmt.Errorf("%w: Linux 不支持以 root 运行完整界面/托盘；请保持普通用户启动，写入 hosts 时会弹出 pkexec 授权。%s",
			ErrNeedElevation, Hint())
	}
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
