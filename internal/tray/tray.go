// Package tray provides system-tray stubs. Full systray wiring (e.g. energye/systray) is a follow-up.
package tray

import "fmt"

// StatusLine formats a short tray tooltip / menu status.
func StatusLine(lastMessage string, lastCheck string) string {
	if lastMessage == "" {
		return "尚未检查"
	}
	if lastCheck == "" {
		return lastMessage
	}
	return fmt.Sprintf("%s | %s", lastCheck, lastMessage)
}

// MenuLabels are Chinese tray menu entries for the MVP shell.
var MenuLabels = struct {
	Refresh  string
	Settings string
	Quit     string
}{
	Refresh:  "立即刷新",
	Settings: "打开设置",
	Quit:     "退出",
}
