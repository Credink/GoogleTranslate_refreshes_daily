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
	Show     string
	Refresh  string
	Settings string
	Quit     string
}{
	Show:     "显示窗口",
	Refresh:  "立即刷新",
	Settings: "打开设置",
	Quit:     "退出程序",
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

	// Build menu before wiring clicks so ShowMenu always has items.
	statusItem = systray.AddMenuItem("上次: 尚未检查", "最近一次检查结果与时间")
	statusItem.Disable()
	systray.AddSeparator()
	mShow := systray.AddMenuItem(MenuLabels.Show, "显示或还原主窗口")
	mRefresh := systray.AddMenuItem(MenuLabels.Refresh, "立即检测并更新 hosts")
	mSettings := systray.AddMenuItem(MenuLabels.Settings, "打开设置窗口")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(MenuLabels.Quit, "结束程序（不只是隐藏到托盘）")

	showWindow := func() {
		if h.OnShowSettings != nil {
			h.OnShowSettings()
		}
	}

	mShow.Click(func() {
		showWindow()
	})
	mRefresh.Click(func() {
		if h.OnRefresh != nil {
			go h.OnRefresh()
		}
	})
	mSettings.Click(func() {
		showWindow()
	})
	mQuit.Click(func() {
		// OnQuit must tear down tray + Wails; do not only systray.Quit (process would linger).
		if h.OnQuit != nil {
			go h.OnQuit()
		}
	})

	wireClicks(showWindow)

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
