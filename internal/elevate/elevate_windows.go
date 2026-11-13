//go:build windows

package elevate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// IsElevated reports whether the current process token is elevated (admin).
func IsElevated() bool {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()
	return token.IsElevated()
}

func elevateWindows(exe, patchFile string) error {
	exe, _ = filepath.Abs(exe)
	patchFile, _ = filepath.Abs(patchFile)
	done := patchFile + ".done"
	_ = os.Remove(done)

	if err := shellExecuteRunAs(exe, "--apply-hosts "+quoteWin(patchFile), 0); err != nil {
		return err
	}

	// Wait for helper to create the .done marker (success) or timeout.
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(done); err == nil {
			_ = os.Remove(done)
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("%w: elevated helper timed out; %s", ErrNeedElevation, Hint())
}

func relaunchElevated(exe, args string) error {
	exe, _ = filepath.Abs(exe)
	if err := shellExecuteRunAs(exe, args, 1); err != nil { // SW_SHOWNORMAL
		return err
	}
	return nil
}

func relaunchUnelevated(exe string) error {
	exe, _ = filepath.Abs(exe)
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return err
	}
	defer token.Close()

	var linked struct {
		LinkedToken windows.Handle
	}
	var retLen uint32
	err = windows.GetTokenInformation(
		token,
		windows.TokenLinkedToken,
		(*byte)(unsafe.Pointer(&linked)),
		uint32(unsafe.Sizeof(linked)),
		&retLen,
	)
	if err != nil {
		return fmt.Errorf("获取普通用户令牌失败: %w", err)
	}
	linkedToken := windows.Token(linked.LinkedToken)
	defer linkedToken.Close()

	var si windows.StartupInfo
	var pi windows.ProcessInformation
	si.Cb = uint32(unsafe.Sizeof(si))
	cmdLine, err := windows.UTF16PtrFromString(quoteWin(exe))
	if err != nil {
		return err
	}
	err = windows.CreateProcessAsUser(linkedToken, nil, cmdLine, nil, nil, false, 0, nil, nil, &si, &pi)
	if err != nil {
		return fmt.Errorf("CreateProcessAsUser: %w", err)
	}
	_ = windows.CloseHandle(pi.Thread)
	_ = windows.CloseHandle(pi.Process)
	return nil
}

func startDetached(exe string) error {
	exe, _ = filepath.Abs(exe)
	cmd := exec.Command(exe)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	return cmd.Start()
}

func shellExecuteRunAs(exe, args string, showCmd int32) error {
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	argPtr, _ := syscall.UTF16PtrFromString(args)
	cwd, _ := syscall.UTF16PtrFromString("")

	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("ShellExecuteW")
	r, _, err := proc.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(argPtr)),
		uintptr(unsafe.Pointer(cwd)),
		uintptr(showCmd),
	)
	// ERROR_CANCELLED (1223) is returned when the user dismisses UAC; it is > 32.
	const errorCancelled = 1223
	if r <= 32 || r == errorCancelled {
		return fmt.Errorf("%w: ShellExecuteW=%d (%v); %s", ErrNeedElevation, r, err, Hint())
	}
	return nil
}

func quoteWin(s string) string {
	if s == "" {
		return `""`
	}
	return `"` + s + `"`
}
