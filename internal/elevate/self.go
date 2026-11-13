package elevate

import (
	"os"
	"path/filepath"
	"strings"
)

// SelfExecutable returns the path that should be re-executed for elevation.
// Under AppImage, os.Executable() points at the FUSE mount (user-only); root/pkexec
// cannot read that path. Prefer $APPIMAGE (the on-disk .AppImage) when set.
func SelfExecutable() (string, error) {
	if ai := strings.TrimSpace(os.Getenv("APPIMAGE")); ai != "" {
		if abs, err := filepath.Abs(ai); err == nil {
			return abs, nil
		}
		return ai, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if abs, err := filepath.Abs(exe); err == nil {
		exe = abs
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// looksLikeAppImageMount reports paths inside an AppImage FUSE mount that root
// typically cannot open (no allow_other).
func looksLikeAppImageMount(path string) bool {
	p := filepath.ToSlash(path)
	return strings.Contains(p, "/.mount_") || strings.Contains(p, "/tmp/.mount")
}
