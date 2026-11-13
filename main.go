package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"GoogleTranslate_refreshes_daily/internal/elevate"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// Injected at link time: -ldflags "-X main.BuildCommit=... -X main.BuildTime=..."
var (
	BuildCommit = "unknown"
	BuildTime   = "unknown"
)

// BuildInfo returns a short baked identity string for logs / verification.
func BuildInfo() string {
	return fmt.Sprintf("%s %s/%s commit=%s built=%s",
		"GoogleTranslateHostsUpdater", runtime.GOOS, runtime.GOARCH, BuildCommit, BuildTime)
}

// setupFileLog mirrors logs to %AppData%\GoogleTranslateHostsUpdater\app.log
// so windowsgui builds remain diagnosable without a console.
func setupFileLog() {
	base, err := os.UserConfigDir()
	if err != nil {
		return
	}
	dir := filepath.Join(base, "GoogleTranslateHostsUpdater")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	// windowsgui builds have no usable stderr; write file-only.
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

func main() {
	// energye/systray locks the importing OS thread in init(); undo that so
	// Wails can own the main thread while tray.Start locks a dedicated one.
	runtime.UnlockOSThread()

	if len(os.Args) >= 2 && os.Args[1] == "--version" {
		fmt.Println(BuildInfo())
		return
	}
	if len(os.Args) >= 3 && os.Args[1] == "--apply-hosts" {
		os.Exit(runApplyHosts(os.Args[2]))
	}
	if len(os.Args) >= 3 && os.Args[1] == "--elevated-relaunch" {
		// Signal the unelevated parent that UAC/pkexec succeeded and we started.
		elevate.MarkRelaunchDone(os.Args[2])
		// Fall through to normal GUI startup.
	}

	setupFileLog()
	log.Printf("main: %s", BuildInfo())

	app := NewApp()
	app.trayIcon = trayIcon
	app.trayTemplateIcon = trayTemplateIcon

	// Initial geometry is a content-fit fallback; OnStartup re-sizes from the
	// primary/current screen logical work area (DPI-aware) via winsize.Fit.
	opts := &options.App{
		Title:     "谷歌翻译 Hosts 更新器",
		Width:     540,
		Height:    980,
		MinWidth:  420,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 248, G: 250, B: 252, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:   false,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarDefault(),
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "谷歌翻译 Hosts 更新器",
				Message: "Hosts 可用性检测与自动更新",
			},
		},
		Linux: &linux.Options{
			WindowIsTranslucent: false,
		},
	}

	err := wails.Run(opts)
	if err != nil {
		log.Fatal(err)
	}
}
