//go:build windows

package elevate

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

func elevateWindows(exe, patchFile string) error {
	exe, _ = filepath.Abs(exe)
	patchFile, _ = filepath.Abs(patchFile)
	done := patchFile + ".done"
	_ = os.Remove(done)

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	args, _ := syscall.UTF16PtrFromString("--apply-hosts " + quoteWin(patchFile))
	cwd, _ := syscall.UTF16PtrFromString("")
	var showCmd int32 = 0 // SW_HIDE

	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("ShellExecuteW")
	r, _, err := proc.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(args)),
		uintptr(unsafe.Pointer(cwd)),
		uintptr(showCmd),
	)
	if r <= 32 {
		return fmt.Errorf("%w: ShellExecuteW=%d (%v); %s", ErrNeedElevation, r, err, Hint())
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

func quoteWin(s string) string {
	if s == "" {
		return `""`
	}
	return `"` + s + `"`
}
