package main

import (
	"context"
	"sync"
	"time"

	"GoogleTranslate_refreshes_daily/internal/autostart"
	"GoogleTranslate_refreshes_daily/internal/schedule"
	"GoogleTranslate_refreshes_daily/internal/state"
	"GoogleTranslate_refreshes_daily/internal/tray"
	"GoogleTranslate_refreshes_daily/internal/updater"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails backend API.
type App struct {
	ctx context.Context

	mu               sync.Mutex
	cfg              state.Config
	st               state.State
	busy             bool
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
			wailsruntime.WindowShow(a.ctx)
			wailsruntime.WindowUnminimise(a.ctx)
		},
		OnQuit: func() {
			wailsruntime.Quit(a.ctx)
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
	// Hide to tray instead of quitting when user closes the window.
	wailsruntime.WindowHide(ctx)
	return true
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
		"refresh":  tray.MenuLabels.Refresh,
		"settings": tray.MenuLabels.Settings,
		"quit":     tray.MenuLabels.Quit,
		"note":     "系统托盘已启用：可查看上次状态并立即刷新。关闭窗口会隐藏到托盘。",
	}
}

func (a *App) Quit() {
	if a.cancel != nil {
		a.cancel()
	}
	if a.trayStop != nil {
		a.trayStop()
	}
	if a.ctx != nil {
		wailsruntime.Quit(a.ctx)
	}
}
