package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if len(os.Args) >= 3 && os.Args[1] == "--apply-hosts" {
		os.Exit(runApplyHosts(os.Args[2]))
	}

	app := NewApp()
	app.trayIcon = trayIcon
	app.trayTemplateIcon = trayTemplateIcon

	opts := &options.App{
		Title:  "谷歌翻译 Hosts 更新器",
		Width:  520,
		Height: 680,
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
	}

	err := wails.Run(opts)
	if err != nil {
		log.Fatal(err)
	}
}
