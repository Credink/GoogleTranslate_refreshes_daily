//go:build windows

package autostart

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

func supported() bool { return true }

func exePath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return p, nil
}

func quotedExe() (string, error) {
	p, err := exePath()
	if err != nil {
		return "", err
	}
	// Quote path for CommandLineToArgvW-safe Run value.
	return `"` + p + `"`, nil
}

func isEnabled() (bool, error) {
	want, err := quotedExe()
	if err != nil {
		return false, err
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	defer k.Close()
	val, _, err := k.GetStringValue(RegistryValueName)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	return normalize(val) == normalize(want), nil
}

func enable() error {
	cmd, err := quotedExe()
	if err != nil {
		return err
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer k.Close()
	if err := k.SetStringValue(RegistryValueName, cmd); err != nil {
		return fmt.Errorf("set Run value: %w", err)
	}
	return nil
}

func disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	defer k.Close()
	err = k.DeleteValue(RegistryValueName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

func normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
