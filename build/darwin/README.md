# macOS build notes

## Tray structure (compile-ready on Windows CI)

- Icons: `icon.png` (regular) + `trayTemplate.png` (black/transparent menu-bar template).
- Code paths: `internal/tray/tray_darwin.go` uses `systray.SetTemplateIcon` + short `SetTitle("Hosts")`.
- App wiring: `icon_darwin.go` embeds both images; `main.go` sets Wails `Mac` options.
- Autostart: `internal/autostart/autostart_darwin.go` writes a LaunchAgent plist under `~/Library/LaunchAgents/`.

## Limits (no Mac hardware in this environment)

- Full `wails build` for darwin needs macOS + Xcode/CGO; **cannot claim GUI tray QA** from Windows.
- Cross-compile of packages that pull `energye/systray` darwin ObjC **requires** a Mac (CGO).
- Verify later on a Mac: menu-bar icon (light/dark), click→menu, hide-to-tray, LaunchAgent login item.

## Suggested Mac commands

```bash
wails build
# app bundle under build/bin/
open build/bin/GoogleTranslateHostsUpdater.app
```
