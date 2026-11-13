// Package tray runs a system tray via energye/systray (Wails-friendly external loop).
package tray

import (
	"fmt"
	"sync"
	"time"

	"github.com/energye/systray"
)

// Hooks are callbacks into the Wails app (must be thread-safe).
type Hooks struct {
	// Icon is the platform tray image (.ico on Windows, .png elsewhere).
	Icon []byte
	// TemplateIcon is a macOS menu-bar template (black + transparent PNG).
	// Ignored on Windows/Linux when empty; when set on darwin, SetTemplateIcon is used.
	TemplateIcon []byte
	OnRefresh      func()
	OnShowSettings func()
	OnQuit         func()
	StatusProvider func() string
}

var (
	statusItem *systray.MenuItem
	mu         sync.Mutex
)

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

// MenuLabels are Chinese tray menu entries.
var MenuLabels = struct {
	Refresh  string
	Settings string
	Quit     string
}{
	Refresh:  "立即刷新",
	Settings: "打开设置",
	Quit:     "退出",
}

// Start launches the tray with an external loop suitable for Wails OnStartup.
// Call the returned stop function on shutdown if needed.
func Start(h Hooks) (stop func()) {
	start, end := systray.RunWithExternalLoop(func() {
		onReady(h)
	}, func() {})
	start()
	return end
}

func onReady(h Hooks) {
	applyIcon(h)

	// macOS: short title next to menu-bar icon; Windows ignores SetTitle.
	systray.SetTitle(platformTitle())
	systray.SetTooltip("谷歌翻译 Hosts 更新器")

	statusItem = systray.AddMenuItem("上次: 尚未检查", "最近一次检查结果")
	statusItem.Disable()
	systray.AddSeparator()
	mRefresh := systray.AddMenuItem(MenuLabels.Refresh, "立即检测并更新 hosts")
	mSettings := systray.AddMenuItem(MenuLabels.Settings, "打开设置窗口")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(MenuLabels.Quit, "退出程序")

	mRefresh.Click(func() {
		if h.OnRefresh != nil {
			go h.OnRefresh()
		}
	})
	mSettings.Click(func() {
		if h.OnShowSettings != nil {
			h.OnShowSettings()
		}
	})
	mQuit.Click(func() {
		if h.OnQuit != nil {
			h.OnQuit()
		}
		systray.Quit()
	})

	wireClicks()

	if h.StatusProvider != nil {
		go func() {
			t := time.NewTicker(3 * time.Second)
			defer t.Stop()
			for range t.C {
				UpdateStatus(h.StatusProvider())
			}
		}()
		UpdateStatus(h.StatusProvider())
	}
}

// UpdateStatus refreshes the disabled status menu item and tooltip.
func UpdateStatus(line string) {
	if line == "" {
		line = "尚未检查"
	}
	mu.Lock()
	defer mu.Unlock()
	systray.SetTooltip("谷歌翻译 Hosts — " + line)
	if statusItem != nil {
		statusItem.SetTitle("上次: " + truncate(line, 48))
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
