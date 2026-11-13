//go:build !windows && !darwin

package main

import _ "embed"

//go:embed build/appicon.png
var trayIcon []byte

var trayTemplateIcon []byte
