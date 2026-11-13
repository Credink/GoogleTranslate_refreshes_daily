//go:build windows

// Tray smoke test: starts energye tray the same way as the app and keeps it
// alive so an operator (or automation) can right-click the icon.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"GoogleTranslate_refreshes_daily/internal/tray"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	fmt.Println("tray-smoke: starting; right-click the tray icon to verify Chinese menu")
	stop := tray.Start(tray.Hooks{
		OnRefresh: func() { log.Println("smoke: refresh") },
		OnShowSettings: func() {
			log.Println("smoke: show")
		},
		OnQuit: func() {
			log.Println("smoke: quit")
			os.Exit(0)
		},
		StatusProvider: func() string { return "smoke-ok" },
	})
	defer stop()

	// Give the dedicated pump time to register the icon.
	time.Sleep(800 * time.Millisecond)
	log.Println("tray-smoke: icon should be visible; waiting for Ctrl+C or menu 退出程序")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
	log.Println("tray-smoke: interrupt")
}
