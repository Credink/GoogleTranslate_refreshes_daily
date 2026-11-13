package main

import (
	"context"
	"log"
	"sync"
	"time"

	"GoogleTranslate_refreshes_daily/internal/autostart"
	"GoogleTranslate_refreshes_daily/internal/chromeproxy"
	"GoogleTranslate_refreshes_daily/internal/elevate"
	"GoogleTranslate_refreshes_daily/internal/hostsfile"
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

	// Preference is admin mode but this process is not elevated → UAC/pkexec relaunch once.
	if cfg.RunAsAdmin && !elevate.IsElevated() {
		log.Printf("app: run_as_admin set but not elevated; requesting elevated relaunch")
		if err := elevate.RelaunchElevated(); err != nil {
			log.Printf("app: elevated relaunch declined or failed: %v — continuing as normal user", err)
		} else {
			go func() {
				time.Sleep(200 * time.Millisecond)
				a.requestQuit()
			}()
			return
		}
	}

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

	ProxyBypassAt      string `json:"proxy_bypass_at"`
	ProxyBypassStatus  string `json:"proxy_bypass_status"`
	ProxyBypassMessage string `json:"proxy_bypass_message"`
	ProxyBypassTooltip string `json:"proxy_bypass_tooltip"`
}

func (a *App) GetStatus() StatusDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	checkAt := ""
	if !a.st.LastCheckAt.IsZero() {
		checkAt = a.st.LastCheckAt.Local().Format("2006-01-02 15:04:05")
	}
	next := schedule.NextRun(time.Now(), a.cfg).Local().Format("2006-01-02 15:04:05")
	bypassAt := ""
	if !a.st.ProxyBypassAt.IsZero() {
		bypassAt = a.st.ProxyBypassAt.Local().Format("2006-01-02 15:04:05")
	}
	return StatusDTO{
		LastCheckAt:        checkAt,
		LastStatus:         string(a.st.LastStatus),
		LastMessage:        a.st.LastMessage,
		LastIP:             a.st.LastIP,
		LastLatency:        a.st.LastLatency,
		Busy:               a.busy,
		TrayLine:           tray.StatusLine(a.st.LastMessage, checkAt),
		NextRunAt:          next,
		ProxyBypassAt:      bypassAt,
		ProxyBypassStatus:  a.st.ProxyBypassStatus,
		ProxyBypassMessage: a.st.ProxyBypassMessage,
		ProxyBypassTooltip: chromeproxy.TooltipZH,
	}
}

type SettingsDTO struct {
	ScheduleMode         string   `json:"schedule_mode"`
	IntervalHours        int      `json:"interval_hours"`
	AllowOnlineOnFailure bool     `json:"allow_online_on_failure"`
	IncludeIPv6          bool     `json:"include_ipv6"`
	LaunchAtLogin        bool     `json:"launch_at_login"`
	AutostartSupported   bool     `json:"autostart_supported"`
	RunAsAdmin           bool     `json:"run_as_admin"`
	IsElevated           bool     `json:"is_elevated"`
	NeedsAdminRelaunch   bool     `json:"needs_admin_relaunch"`
	ElevationLabel       string   `json:"elevation_label"`
	ElevationExplain     string   `json:"elevation_explain"`
	ElevationPlatform    string   `json:"elevation_platform"`
	IncludeRegionalPack  bool     `json:"include_regional_pack"`
	Workers              int      `json:"workers"`
	TimeoutMS            int      `json:"timeout_ms"`
	Repeat               int      `json:"repeat"`
	Domains              []string `json:"domains"`
	RegionalDomains      []string `json:"regional_domains"`
	EffectiveDomains     []string `json:"effective_domains"`
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
	elev := elevate.IsElevated()
	return SettingsDTO{
		ScheduleMode:         string(a.cfg.ScheduleMode),
		IntervalHours:        a.cfg.IntervalHours,
		AllowOnlineOnFailure: a.cfg.AllowOnlineOnFailure,
		IncludeIPv6:          a.cfg.IncludeIPv6,
		LaunchAtLogin:        launch,
		AutostartSupported:   autostart.Supported(),
		RunAsAdmin:           a.cfg.RunAsAdmin,
		IsElevated:           elev,
		NeedsAdminRelaunch:   a.cfg.RunAsAdmin && !elev,
		ElevationLabel:       elevate.CurrentModeLabel(),
		ElevationExplain:     elevate.ModeExplanation,
		ElevationPlatform:    elevate.PlatformNotes(),
		IncludeRegionalPack:  a.cfg.IncludeRegionalPack,
		Workers:              a.cfg.Workers,
		TimeoutMS:            a.cfg.TimeoutMS,
		Repeat:               a.cfg.Repeat,
		Domains:              a.cfg.Domains,
		RegionalDomains:      append([]string{}, hostsfile.RegionalDomains...),
		EffectiveDomains:     state.EffectiveDomains(a.cfg),
	}
}

func (a *App) SaveSettings(s SettingsDTO) string {
	a.mu.Lock()
	prevAdmin := a.cfg.RunAsAdmin
	a.cfg.ScheduleMode = state.ScheduleMode(s.ScheduleMode)
	if a.cfg.ScheduleMode == "" {
		a.cfg.ScheduleMode = state.ModeDailyMidnight
	}
	a.cfg.IntervalHours = s.IntervalHours
	a.cfg.AllowOnlineOnFailure = s.AllowOnlineOnFailure
	a.cfg.IncludeIPv6 = s.IncludeIPv6
	a.cfg.LaunchAtLogin = s.LaunchAtLogin
	a.cfg.RunAsAdmin = s.RunAsAdmin
	a.cfg.IncludeRegionalPack = s.IncludeRegionalPack
	a.cfg.Workers = s.Workers
	a.cfg.TimeoutMS = s.TimeoutMS
	a.cfg.Repeat = s.Repeat
	if len(s.Domains) > 0 {
		a.cfg.Domains = s.Domains
	} else {
		a.cfg.Domains = state.DefaultConfig().Domains
	}
	cfg := a.cfg
	a.mu.Unlock()

	if err := state.SaveConfig(cfg); err != nil {
		return "保存失败: " + err.Error()
	}
	if autostart.Supported() {
		if err := autostart.Apply(s.LaunchAtLogin); err != nil {
			return "设置已保存，但开机自启更新失败: " + err.Error()
		}
	}

	// Dual elevation mode transitions: persist first, then relaunch + quit.
	if s.RunAsAdmin && !prevAdmin {
		if elevate.IsElevated() {
			return "设置已保存（已在管理员模式下运行）"
		}
		if err := elevate.RelaunchElevated(); err != nil {
			return "设置已保存，但提权重启失败（可稍后点「立即以管理员身份重启」）: " + err.Error()
		}
		a.quitSoon()
		return "设置已保存，正在以管理员身份重新启动…"
	}
	if !s.RunAsAdmin && prevAdmin {
		if !elevate.IsElevated() {
			return "设置已保存（当前为普通用户）"
		}
		if err := elevate.RelaunchUnelevated(); err != nil {
			return "设置已保存。无法自动降权：" + err.Error()
		}
		a.quitSoon()
		return "设置已保存，正在以普通用户重新启动…"
	}
	if s.RunAsAdmin && !elevate.IsElevated() {
		return "设置已保存。偏好为管理员模式，但当前未提权 — 请点「立即以管理员身份重启」或重新打开本程序。"
	}
	return "设置已保存"
}

// RelaunchAsAdmin triggers an elevated relaunch when the user prefers admin mode
// but the current process is still unelevated (e.g. UAC was cancelled at startup).
func (a *App) RelaunchAsAdmin() string {
	a.mu.Lock()
	want := a.cfg.RunAsAdmin
	a.mu.Unlock()
	if elevate.IsElevated() {
		return "当前已是管理员模式"
	}
	if !want {
		return "请先勾选「以管理员身份运行」并保存"
	}
	if err := elevate.RelaunchElevated(); err != nil {
		return "提权重启失败: " + err.Error()
	}
	a.quitSoon()
	return "正在以管理员身份重新启动…"
}

func (a *App) quitSoon() {
	go func() {
		time.Sleep(400 * time.Millisecond)
		a.requestQuit()
	}()
}

func (a *App) RefreshNow() StatusDTO {
	a.runRefresh(true)
	a.pushTrayStatus()
	return a.GetStatus()
}

// ConfigureProxyBypass manually re-checks ZeroOmega/SwitchyOmega DIRECT/bypass rules
// for the current effective managed domains. Still idempotent: writes only if missing/typo.
// Best-effort; not all proxy extensions covered.
func (a *App) ConfigureProxyBypass() StatusDTO {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	domains := state.EffectiveDomains(cfg)
	pr := chromeproxy.ConfigureBypassOpts(domains, chromeproxy.Options{Force: true})
	a.mu.Lock()
	a.st.ProxyBypassTried = true
	a.st.ProxyBypassAt = pr.At
	a.st.ProxyBypassStatus = pr.Status
	a.st.ProxyBypassMessage = pr.Message
	if pr.DomainsKey != "" {
		a.st.ProxyBypassDomainsKey = pr.DomainsKey
	}
	st := a.st
	a.mu.Unlock()
	_ = state.SaveState(st)
	return a.GetStatus()
}

// ProxyBypassTooltip returns the Chinese hint for the hosts/proxy warning icon.
func (a *App) ProxyBypassTooltip() string {
	return chromeproxy.TooltipZH
}

func (a *App) TrayMenuLabels() map[string]string {
	return map[string]string{
		"show":     tray.MenuLabels.Show,
		"refresh":  tray.MenuLabels.Refresh,
		"settings": tray.MenuLabels.Settings,
		"quit":     tray.MenuLabels.Quit,
		"note": "系统托盘：左键显示窗口，右键打开菜单。关闭窗口隐藏到托盘；「退出程序」会结束进程。浏览器请关闭「安全 DNS/DoH」，否则会绕过 hosts。",
	}
}

// GetBuildInfo returns the baked commit/time for verifying the running binary.
func (a *App) GetBuildInfo() string {
	return BuildInfo()
}

func (a *App) Quit() {
	a.requestQuit()
}
