//go:build linux

package autostart

import (
	"os"
	"path/filepath"
	"strings"
)

// XDG Autostart: ~/.config/autostart/<name>.desktop

func supported() bool { return true }

func desktopFileName() string {
	return RegistryValueName + ".desktop"
}

func desktopPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "autostart", desktopFileName()), nil
}

func exePath() (string, error) {
	return os.Executable()
}

func isEnabled() (bool, error) {
	path, err := desktopPath()
	if err != nil {
		return false, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	exe, err := exePath()
	if err != nil {
		return false, err
	}
	body := string(b)
	return strings.Contains(body, "Exec="+exe) || strings.Contains(body, "Exec=\""+exe+"\""), nil
}

func enable() error {
	exe, err := exePath()
	if err != nil {
		return err
	}
	path, err := desktopPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(desktopEntry(exe)), 0644)
}

func disable() error {
	path, err := desktopPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
