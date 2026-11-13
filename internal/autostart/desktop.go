package autostart

import (
	"fmt"
	"strings"
)

// desktopEntry builds an XDG Autostart .desktop body for exe.
func desktopEntry(exe string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=谷歌翻译 Hosts 更新器
Comment=Google Translate hosts availability checker
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`, desktopExecEscape(exe))
}

// desktopExecEscape quotes Exec= values that contain whitespace or shell meta.
func desktopExecEscape(path string) string {
	if !strings.ContainsAny(path, " \t\"'\\$") {
		return path
	}
	r := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`$`, `\$`,
	)
	return `"` + r.Replace(path) + `"`
}
