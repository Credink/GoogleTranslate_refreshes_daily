// Package elevate handles privileged hosts writes across platforms.
package elevate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var ErrNeedElevation = errors.New("writing hosts requires elevation")

// CanWriteHosts reports whether the current process can write the hosts path.
func CanWriteHosts(path string) bool {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

// Hint returns a short Chinese message explaining how to elevate on this OS.
func Hint() string {
	switch runtime.GOOS {
	case "windows":
		return "请在 UAC 提示中确认，或以管理员身份运行。"
	case "darwin":
		return "写入 /etc/hosts 需要管理员权限，请在系统提示中输入密码确认。"
	default:
		return "写入 /etc/hosts 需要管理员权限（pkexec/sudo）。"
	}
}

// ApplyHostsPatch writes content to path if permitted; otherwise returns ErrNeedElevation.
func ApplyHostsPatch(path, content string) error {
	if CanWriteHosts(path) {
		return os.WriteFile(path, []byte(content), 0644)
	}
	return fmt.Errorf("%w: %s", ErrNeedElevation, Hint())
}

// ApplyElevated relaunches this executable with --apply-hosts <patchFile> elevated.
func ApplyElevated(patchFile string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return elevateWindows(exe, patchFile)
	case "darwin":
		script := fmt.Sprintf(`do shell script "%s --apply-hosts %s" with administrator privileges`,
			shellQuote(exe), shellQuote(patchFile))
		cmd := exec.Command("osascript", "-e", script)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %v (%s); %s", ErrNeedElevation, err, string(out), Hint())
		}
		return nil
	default:
		cmd := exec.Command("pkexec", exe, "--apply-hosts", patchFile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %v (%s); %s", ErrNeedElevation, err, string(out), Hint())
		}
		return nil
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
