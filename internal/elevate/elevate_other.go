//go:build !windows

package elevate

func elevateWindows(exe, patchFile string) error {
	_ = exe
	_ = patchFile
	return ErrNeedElevation
}
