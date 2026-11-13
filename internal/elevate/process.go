package elevate

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

// ModeExplanation is the Chinese copy shown near the admin-mode toggle.
const ModeExplanation = "管理员模式可长期静默更新 hosts（换 IP 时不再每次点授权）。普通模式更安全，但每次真正改 hosts 都需要点击授予权限。"

// CurrentModeLabel returns a short Chinese label for the process elevation state.
func CurrentModeLabel() string {
	if IsElevated() {
		return "当前：已提权（管理员）"
	}
	return "当前：普通用户"
}

// PlatformNotes describes OS-specific limits for dual elevation modes.
func PlatformNotes() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows：开启管理员模式会弹出 UAC；关闭时可尝试降权重启。若降权失败，请完全退出后以普通方式再打开。"
	case "darwin":
		return "macOS：开启管理员模式会尝试通过系统授权重启；关闭管理员模式时若无法自动降权，请完全退出后普通再开。"
	default:
		return "Linux：开启管理员模式会尝试 pkexec 重启；关闭时若无法自动降权，请完全退出后普通再开。"
	}
}

// RelaunchElevated starts a new elevated instance of this executable (GUI) and
// waits until that instance signals it is running (so UAC cancel does not quit us).
// The caller should then quit.
func RelaunchElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	nonce := NewRelaunchNonce()
	args := "--elevated-relaunch " + nonce
	if err := relaunchElevated(exe, args); err != nil {
		return err
	}
	if err := waitRelaunchDone(nonce, 2*time.Minute); err != nil {
		return err
	}
	return nil
}

// RelaunchUnelevated starts a new normal-user instance when possible, then the
// caller should quit. On some platforms this is best-effort.
func RelaunchUnelevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if !IsElevated() {
		return startDetached(exe)
	}
	if err := relaunchUnelevated(exe); err != nil {
		return fmt.Errorf("%w；请完全退出本程序后，再以普通方式重新打开", err)
	}
	return nil
}
