package elevate

import (
	"fmt"
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
		return "Linux：请始终以普通用户运行（不要 sudo 整个 AppImage）。写入 hosts 时会弹出 pkexec/polkit 授权对话框，只提权 --apply-hosts 助手。不支持以 root 跑完整 GUI/托盘。"
	}
}

// SupportsProcessElevation reports whether this OS can relaunch the full GUI elevated.
// Linux only elevates the --apply-hosts helper via pkexec.
func SupportsProcessElevation() bool {
	return runtime.GOOS == "windows" || runtime.GOOS == "darwin"
}

// RelaunchElevated starts a new elevated instance of this executable (GUI) and
// waits until that instance signals it is running (so UAC cancel does not quit us).
// The caller should then quit.
func RelaunchElevated() error {
	if !SupportsProcessElevation() {
		return fmt.Errorf("%w: 当前系统不支持整进程管理员模式；写入 hosts 时会按次弹出 pkexec 授权", ErrNeedElevation)
	}
	exe, err := SelfExecutable()
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
	exe, err := SelfExecutable()
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

// RootGUIWarning is shown / logged when the GUI is started as root on Unix.
const RootGUIWarning = "请勿使用 sudo 运行本程序的完整界面。请以普通用户启动；写入 /etc/hosts 时会弹出 pkexec 授权对话框。系统托盘在 root 下已禁用，以免崩溃。"
