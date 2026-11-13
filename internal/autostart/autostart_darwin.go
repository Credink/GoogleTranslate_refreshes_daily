//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LaunchAgent keeps a simple RunAtLoad plist for unbundled / pre-Ventura Macs.
// Real GUI tray QA on macOS still requires Apple hardware; this path is compile-ready only.

func supported() bool { return true }

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", RegistryValueName+".plist"), nil
}

func exePath() (string, error) {
	return os.Executable()
}

func isEnabled() (bool, error) {
	path, err := plistPath()
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
	return strings.Contains(string(b), exe), nil
}

func enable() error {
	exe, err := exePath()
	if err != nil {
		return err
	}
	path, err := plistPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`, RegistryValueName, xmlEscape(exe))
	return os.WriteFile(path, []byte(body), 0644)
}

func disable() error {
	path, err := plistPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func xmlEscape(s string) string {
	r := strings.NewReplacer(
		`&`, `&amp;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
		`"`, `&quot;`,
	)
	return r.Replace(s)
}
