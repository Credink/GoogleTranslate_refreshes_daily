//go:build darwin

package main

import _ "embed"

//go:embed build/darwin/icon.png
var trayIcon []byte

//go:embed build/darwin/trayTemplate.png
var trayTemplateIcon []byte
