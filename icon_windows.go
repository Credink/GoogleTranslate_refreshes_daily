//go:build windows

package main

import _ "embed"

//go:embed build/windows/icon.ico
var trayIcon []byte

// macOS-only template; empty on Windows.
var trayTemplateIcon []byte
