package main

import (
	"context"
	"log"
	"sync"
	"time"

	"GoogleTranslate_refreshes_daily/internal/autostart"
	"GoogleTranslate_refreshes_daily/internal/schedule"
	"GoogleTranslate_refreshes_daily/internal/state"
	"GoogleTranslate_refreshes_daily/internal/tray"
	"GoogleTranslate_refreshes_daily/internal/updater"
	"GoogleTranslate_refreshes_daily/internal/winsize"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails backend API.
type App struct {
	ctx context.Context

	mu               sync.Mutex
	cfg              state.Config
	st               state.State
	busy             bool
	quitting         bool // when true, beforeClose allows real process exit
	cancel           context.CancelFunc
	trayIcon         []byte
	trayTemplateIcon []byte
	trayStop         func()
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Printf("app: startup build=%s", BuildInfo())
	a.applyScreenAwareWindowSize()
	// Screen metrics can be incomplete on the first OnStartup tick; re-fit shortly after.
	go func() {
		time.Sleep(350 * time.Millisecond)
		a.applyScreenAwareWindowSize()
	}()

	cfg, _ := state.LoadConfig()
	st, _ := state.LoadState()
	a.mu.Lock()
	a.cfg = cfg
	a.st = st
	a.mu.Unlock()

	a.trayStop = tray.Start(tray.Hooks{
		Icon:         a.trayIcon,
		TemplateIcon: a.trayTemplateIcon,
		OnRefresh: func() {
			a.runRefresh(true)
			a.pushTrayStatus()
		},
		OnShowSettings: func() {
			a.showMainWindow()
		},
		OnQuit: func() {
			a.requestQuit()
		},
		StatusProvider: func() string {
			s := a.GetStatus()
			return s.TrayLine
		},
	})

	// Re-apply persisted launch-at-login preference (safe if already registered).
	if cfg.LaunchAtLogin && autostart.Supported() {
		_ = autostart.Enable()
	}

	if schedule.NeedsCheck(time.Now(), cfg, st) {
		go func() {
			a.runRefresh(false)
			a.pushTrayStatus()
		}()
	}
	go a.scheduleLoop()
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	a.mu.Lock()
	quitting := a.quitting
	a.mu.Unlock()
	if quitting {
		// Tray 「退出程序」 (or Quit API): allow Wails to tear down and end the process.
		return false
	}
	// Close button: hide to tray instead of quitting.
	wailsruntime.WindowHide(ctx)
	return true
}

// requestQuit tears down tray + in-flight work and exits the Wails process.
func (a *App) requestQuit() {
	a.mu.Lock()
	if a.quitting {
		a.mu.Unlock()
		return
	}
	a.quitting = true
	cancel := a.cancel
	stop := a.trayStop
	a.trayStop = nil
	ctx := a.ctx
	a.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if stop != nil {
		stop()
	}
	if ctx != nil {
		wailsruntime.Quit(ctx)
	}
}

func (a *App) showMainWindow() {
	if a.ctx == nil {
		return
	}
	wailsruntime.WindowShow(a.ctx)
	wailsruntime.WindowUnminimise(a.ctx)
	// Bring to foreground after hide-to-tray on Windows.
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, true)
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, false)
}

// applyScreenAwareWindowSize sizes the window from the current/primary screen
// logical work area (DPI-aware via Wails Screen.Size) with min clamps; user may still resize.
func (a *App) applyScreenAwareWindowSize() {
	if a.ctx == nil {
		return
	}
	sz := winsize.Fallback()
	sw, sh := 0, 0
	if screens, err := wailsruntime.ScreenGetAll(a.ctx); err == nil && len(screens) > 0 {
		chosen := screens[0]
		for _, s := range screens {
			if s.IsCurrent {
				chosen = s
				break
			}
			if s.IsPrimary {
				chosen = s
			}
		}
		sw, sh = chosen.Size.Width, chosen.Size.Height
		if sw <= 0 || sh <= 0 {
			sw, sh = chosen.Width, chosen.Height
		}
		sz = winsize.Fit(sw, sh)
	} else if err != nil {
		log.Printf("app: ScreenGetAll failed: %v — using fallback", err)
	}
	log.Printf("app: %s", winsize.Describe(sz, sw, sh))
	wailsruntime.WindowSetMinSize(a.ctx, sz.MinWidth, sz.MinHeight)
	wailsruntime.WindowSetSize(a.ctx, sz.Width, sz.Height)
	wailsruntime.WindowCenter(a.ctx)
}

func (a *App) pushTrayStatus() {
	s := a.GetStatus()
	tray.UpdateStatus(s.TrayLine)
}

func (a *App) scheduleLoop() {
	for {
		a.mu.Lock()
		cfg := a.cfg
		a.mu.Unlock()
		next := schedule.NextRun(time.Now(), cfg)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-a.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			a.mu.Lock()
			cfg = a.cfg
			st := a.st
			a.mu.Unlock()
			if schedule.NeedsCheck(time.Now(), cfg, st) {
				a.runRefresh(false)
				a.pushTrayStatus()
			}
		}
	}
}

func (a *App) runRefresh(manual bool) updater.Report {
	a.mu.Lock()
	if a.busy {
		a.mu.Unlock()
		return updater.Report{Status: state.StatusSkipped, Message: "已有检查在进行中"}
	}
	a.busy = true
	cfg := a.cfg
	st := a.st
	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	a.cancel = cancel
	a.mu.Unlock()

	defer func() {
		cancel()
		a.mu.Lock()
		a.busy = false
		a.cancel = nil
		a.mu.Unlock()
	}()

	_ = manual
	rep, newSt := updater.Refresh(ctx, cfg, st)
	a.mu.Lock()
	a.st = newSt
	a.mu.Unlock()
	return rep
}

// StatusDTO is exposed to the frontend.
type StatusDTO struct {
	LastCheckAt string `json:"last_check_at"`
	LastStatus  string `json:"last_status"`
	LastMessage string `json:"last_message"`
	LastIP      string `json:"last_ip"`
	LastLatency string `json:"last_latency"`
	Busy        bool   `json:"busy"`
	TrayLine    string `json:"tray_line"`
	NextRunAt   string `json:"next_run_at"`
}

func (a *App) GetStatus() StatusDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	checkAt := ""
	if !a.st.LastCheckAt.IsZero() {
		checkAt = a.st.LastCheckAt.Local().Format("2006-01-02 15:04:05")
	}
	next := schedule.NextRun(time.Now(), a.cfg).Local().Format("2006-01-02 15:04:05")
	return StatusDTO{
		LastCheckAt: checkAt,
		LastStatus:  string(a.st.LastStatus),
		LastMessage: a.st.LastMessage,
		LastIP:      a.st.LastIP,
		LastLatency: a.st.LastLatency,
		Busy:        a.busy,
		TrayLine:    tray.StatusLine(a.st.LastMessage, checkAt),
		NextRunAt:   next,
	}
}

type SettingsDTO struct {
	ScheduleMode         string   `json:"schedule_mode"`
	IntervalHours        int      `json:"interval_hours"`
	AllowOnlineOnFailure bool     `json:"allow_online_on_failure"`
	IncludeIPv6          bool     `json:"include_ipv6"`
	LaunchAtLogin        bool     `json:"launch_at_login"`
	AutostartSupported   bool     `json:"autostart_supported"`
	Workers              int      `json:"workers"`
	TimeoutMS            int      `json:"timeout_ms"`
	Repeat               int      `json:"repeat"`
	Domains              []string `json:"domains"`
}

func (a *App) GetSettings() SettingsDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	launch := a.cfg.LaunchAtLogin
	if autostart.Supported() {
		if enabled, err := autostart.IsEnabled(); err == nil {
			launch = enabled
			a.cfg.LaunchAtLogin = enabled
		}
	}
	return SettingsDTO{
		ScheduleMode:         string(a.cfg.ScheduleMode),
		IntervalHours:        a.cfg.IntervalHours,
		AllowOnlineOnFailure: a.cfg.AllowOnlineOnFailure,
		IncludeIPv6:          a.cfg.IncludeIPv6,
		LaunchAtLogin:        launch,
		AutostartSupported:   autostart.Supported(),
		Workers:              a.cfg.Workers,
		TimeoutMS:            a.cfg.TimeoutMS,
		Repeat:               a.cfg.Repeat,
		Domains:              a.cfg.Domains,
	}
}

func (a *App) SaveSettings(s SettingsDTO) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.ScheduleMode = state.ScheduleMode(s.ScheduleMode)
	if a.cfg.ScheduleMode == "" {
		a.cfg.ScheduleMode = state.ModeDailyMidnight
	}
	a.cfg.IntervalHours = s.IntervalHours
	a.cfg.AllowOnlineOnFailure = s.AllowOnlineOnFailure
	a.cfg.IncludeIPv6 = s.IncludeIPv6
	a.cfg.LaunchAtLogin = s.LaunchAtLogin
	a.cfg.Workers = s.Workers
	a.cfg.TimeoutMS = s.TimeoutMS
	a.cfg.Repeat = s.Repeat
	if len(s.Domains) > 0 {
		a.cfg.Domains = s.Domains
	}
	if err := state.SaveConfig(a.cfg); err != nil {
		return "保存失败: " + err.Error()
	}
	if autostart.Supported() {
		if err := autostart.Apply(s.LaunchAtLogin); err != nil {
			return "设置已保存，但开机自启更新失败: " + err.Error()
		}
	}
	return "设置已保存"
}

func (a *App) RefreshNow() StatusDTO {
	a.runRefresh(true)
	a.pushTrayStatus()
	return a.GetStatus()
}

func (a *App) TrayMenuLabels() map[string]string {
	return map[string]string{
		"show":     tray.MenuLabels.Show,
		"refresh":  tray.MenuLabels.Refresh,
		"settings": tray.MenuLabels.Settings,
		"quit":     tray.MenuLabels.Quit,
		"note":     "系统托盘：左键显示窗口，右键打开菜单。关闭窗口隐藏到托盘；「退出程序」会结束进程。",
	}
}

// GetBuildInfo returns the baked commit/time for verifying the running binary.
func (a *App) GetBuildInfo() string {
	return BuildInfo()
}

func (a *App) Quit() {
	a.requestQuit()
}
