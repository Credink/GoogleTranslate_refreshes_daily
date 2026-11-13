//go:build darwin

package tray

import "github.com/energye/systray"

func platformTitle() string {
	// Compact label beside the menu-bar icon (macOS only).
	return "Hosts"
}

func applyIcon(h Hooks) {
	regular := h.Icon
	template := h.TemplateIcon
	if len(template) > 0 && len(regular) > 0 {
		systray.SetTemplateIcon(template, regular)
		return
	}
	if len(template) > 0 {
		systray.SetTemplateIcon(template, template)
		return
	}
	if len(regular) > 0 {
		systray.SetIcon(regular)
	}
}

func wireClicks(onLeftClick func()) {
	// macOS menu-bar: click surfaces the menu; "显示窗口" item still restores UI.
	_ = onLeftClick
	systray.SetOnClick(func(menu systray.IMenu) {
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
