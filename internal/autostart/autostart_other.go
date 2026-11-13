//go:build !windows && !darwin && !linux

package autostart

import "fmt"

func supported() bool { return false }

func isEnabled() (bool, error) { return false, nil }

func enable() error {
	return fmt.Errorf("autostart not implemented on this platform")
}

func disable() error { return nil }
