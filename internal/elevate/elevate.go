// Package elevate handles privileged hosts writes across platforms.
package elevate

import (
	"errors"
	"fmt"
	"os"
	"runtime"
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
		return "请以管理员身份重新运行本程序，或在 UAC 提示中确认授权后写入 hosts。"
	case "darwin":
		return "写入 /etc/hosts 需要管理员权限，请在系统提示中输入密码确认。"
	default:
		return "写入 /etc/hosts 需要管理员权限，请使用 pkexec/sudo 或授权后的提权助手。"
	}
}

// ApplyHostsPatch writes content to path if permitted; otherwise returns ErrNeedElevation.
// Future: spawn a one-shot elevated helper instead of requiring whole-app elevation.
func ApplyHostsPatch(path, content string) error {
	if CanWriteHosts(path) {
		return os.WriteFile(path, []byte(content), 0644)
	}
	return fmt.Errorf("%w: %s", ErrNeedElevation, Hint())
}
