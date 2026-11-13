//go:build !windows && !darwin

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

func wireClicks() {
	systray.SetOnClick(func(menu systray.IMenu) {
		_ = menu.ShowMenu()
	})
	systray.SetOnRClick(func(menu systray.IMenu) {
		_ = menu.ShowMenu()
	})
}
