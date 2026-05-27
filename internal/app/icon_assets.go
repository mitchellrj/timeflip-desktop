package app

import _ "embed"

//go:embed icon_assets/appicon.png
var appIconPNG []byte

//go:embed icon_assets/tray-plain.png
var trayPlainIconPNG []byte

//go:embed icon_assets/tray-running.png
var trayRunningIconPNG []byte

//go:embed icon_assets/tray-paused.png
var trayPausedIconPNG []byte
