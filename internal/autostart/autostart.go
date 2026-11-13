// Package autostart toggles launch-at-login registration.
package autostart

const (
	// RegistryValueName is the Windows HKCU Run value / LaunchAgent label stem.
	RegistryValueName = "GoogleTranslateHostsUpdater"
)

// Supported reports whether this OS has an implemented autostart backend.
func Supported() bool {
	return supported()
}

// IsEnabled reports whether launch-at-login is currently registered for this executable.
func IsEnabled() (bool, error) {
	return isEnabled()
}

// Enable registers the current executable to start at user login.
func Enable() error {
	return enable()
}

// Disable removes the launch-at-login registration.
func Disable() error {
	return disable()
}

// Apply enables or disables according to want.
func Apply(want bool) error {
	if want {
		return Enable()
	}
	return Disable()
}
