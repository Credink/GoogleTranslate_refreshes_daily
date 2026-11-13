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

// wireClicks: Windows left/double-click restore the window.
// Do NOT SetOnRClick — energye then defaults to ShowMenu on right-click,
// which is more reliable than a custom handler that can race the pump.
func wireClicks(onLeftClick func()) {
	systray.SetOnClick(func(menu systray.IMenu) {
		_ = menu
		if onLeftClick != nil {
			onLeftClick() // already dispatches via go in onReady
		}
	})
	systray.SetOnDClick(func(menu systray.IMenu) {
		_ = menu
		if onLeftClick != nil {
			onLeftClick()
		}
	})
}
