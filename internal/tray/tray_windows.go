//go:build windows

package tray

import "github.com/energye/systray"

func platformTitle() string {
	return "谷歌翻译 Hosts"
}

func applyIcon(h Hooks) {
	if len(h.Icon) > 0 {
		systray.SetIcon(h.Icon)
	}
}

// wireClicks: Windows left-click restores the window; right-click opens the tray menu.
func wireClicks(onLeftClick func()) {
	systray.SetOnClick(func(menu systray.IMenu) {
		_ = menu // menu available; left-click restores UI instead of popping the menu
		if onLeftClick != nil {
			onLeftClick()
		}
	})
	systray.SetOnDClick(func(menu systray.IMenu) {
		_ = menu
		if onLeftClick != nil {
			onLeftClick()
		}
	})
	// Explicit RClick handler required: once set, energye will not auto-show the menu.
	systray.SetOnRClick(func(menu systray.IMenu) {
		if menu != nil {
			_ = menu.ShowMenu()
		}
	})
}
