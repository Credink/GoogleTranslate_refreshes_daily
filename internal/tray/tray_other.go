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

func wireClicks(onLeftClick func()) {
	systray.SetOnClick(func(menu systray.IMenu) {
		if onLeftClick != nil {
			onLeftClick()
			return
		}
		if menu != nil {
			_ = menu.ShowMenu()
		}
	})
	systray.SetOnRClick(func(menu systray.IMenu) {
		if menu != nil {
			_ = menu.ShowMenu()
		}
	})
}
