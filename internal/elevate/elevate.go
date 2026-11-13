// Package elevate handles privileged hosts writes across platforms.
package elevate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		return "写入 /etc/hosts 需要管理员权限。请在 pkexec/polkit 对话框中确认；不要用 sudo 运行整个 AppImage。"
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
	exe, err := SelfExecutable()
	if err != nil {
		return err
	}
	patchFile, err = filepath.Abs(patchFile)
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
		return elevateLinux(exe, "--apply-hosts", patchFile)
	}
}

// elevateLinux prefers pkexec (polkit GUI), then graphical sudo helpers, then sudo.
// For AppImage, re-exec $APPIMAGE (not the FUSE-mounted binary) so root can read it.
func elevateLinux(exe string, args ...string) error {
	target := exe
	if ai := strings.TrimSpace(os.Getenv("APPIMAGE")); ai != "" {
		if abs, err := filepath.Abs(ai); err == nil {
			target = abs
		} else {
			target = ai
		}
	} else if looksLikeAppImageMount(exe) {
		return fmt.Errorf("%w: 当前可执行文件在 AppImage FUSE 挂载内，root/pkexec 无法读取；请直接运行磁盘上的 .AppImage（保留 APPIMAGE 环境变量）; %s",
			ErrNeedElevation, Hint())
	}
	if abs, err := filepath.Abs(target); err == nil {
		target = abs
	}

	var errs []string
	if err := runWithPkexec(target, args...); err == nil {
		return nil
	} else if err != nil {
		errs = append(errs, err.Error())
	}

	for _, name := range []string{"lxqt-sudo", "gksudo", "kdesudo", "sudo"} {
		if _, err := exec.LookPath(name); err != nil {
			continue
		}
		cmd := exec.Command(name, append([]string{target}, args...)...)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v (%s)", name, err, strings.TrimSpace(string(out))))
	}
	if len(errs) == 0 {
		return fmt.Errorf("%w: no pkexec/sudo found; %s", ErrNeedElevation, Hint())
	}
	return fmt.Errorf("%w: %s; %s", ErrNeedElevation, strings.Join(errs, "; "), Hint())
}

// runWithPkexec runs: pkexec /usr/bin/env KEY=VAL… <target> <args…>
// so the polkit dialog can appear on the user session display and AppImage
// re-exec keeps APPIMAGE / EXTRACT_AND_RUN.
func runWithPkexec(target string, args ...string) error {
	pk, err := exec.LookPath("pkexec")
	if err != nil {
		return err
	}
	envPath, err := exec.LookPath("env")
	if err != nil {
		envPath = "/usr/bin/env"
	}
	cmd := exec.Command(pk, buildPkexecArgv(envPath, target, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pkexec: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// buildPkexecArgv builds arguments after "pkexec" (env binary + KEY=VAL + target + args).
func buildPkexecArgv(envPath, target string, args ...string) []string {
	out := []string{envPath}
	for _, key := range []string{
		"DISPLAY", "XAUTHORITY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR",
		"LANG", "LC_ALL", "LANGUAGE",
	} {
		if v := os.Getenv(key); v != "" {
			out = append(out, key+"="+v)
		}
	}
	if ai := strings.TrimSpace(os.Getenv("APPIMAGE")); ai != "" {
		out = append(out, "APPIMAGE="+ai)
		for _, key := range []string{"APPDIR", "OWD", "ARGV0"} {
			if v := os.Getenv(key); v != "" {
				out = append(out, key+"="+v)
			}
		}
		// Elevated re-exec: extract to a real dir so root is not blocked by FUSE.
		out = append(out, "APPIMAGE_EXTRACT_AND_RUN=1")
	}
	out = append(out, target)
	out = append(out, args...)
	return out
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
