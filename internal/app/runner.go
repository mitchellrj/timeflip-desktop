package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/services"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func RunWithOptions(opts Options) {
	traceWriter, closeTrace, err := openBLETrace(opts.TraceBLEPath)
	if err != nil {
		log.Fatal(err)
	}
	defer closeTrace()

	controller, bus, err := BuildController(context.Background(), BootstrapOptions{BLETrace: traceWriter})
	if err != nil {
		log.Fatal(err)
	}

	wailsApp := application.New(application.Options{
		Name:        "TimeFlip Desktop",
		Description: "Local TimeFlip2 task tracking",
		Icon:        appIconPNG,
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(frontendAssets()),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyRegular,
		},
		Services: []application.Service{
			application.NewService(controller),
		},
	})

	bus.SetEmitter(func(name string, payload any) {
		wailsApp.Event.Emit(name, payload)
	})

	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "TimeFlip Desktop",
		Width:            1100,
		Height:           760,
		MinWidth:         940,
		MinHeight:        620,
		URL:              "/",
		BackgroundColour: application.NewRGB(245, 247, 250),
	})
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	configureControlCentre(wailsApp, controller, window)

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

func configureControlCentre(wailsApp *application.App, controller *Controller, window application.Window) {
	systemTray := wailsApp.SystemTray.New()
	systemTray.SetTooltip("TimeFlip Desktop")

	rebuildMenu := func() {
		setControlCentreIcon(systemTray, controller)
		systemTray.SetMenu(buildControlCentreMenu(wailsApp, controller, window, systemTray))
	}

	rebuildMenu()
	systemTray.OnClick(func() {
		systemTray.OpenMenu()
	})

	for _, eventName := range []string{
		"device.connection",
		"device.state",
		"tracking.session.started",
		"tracking.session.ended",
		"shell.refresh",
	} {
		wailsApp.Event.On(eventName, func(*application.CustomEvent) {
			rebuildMenu()
		})
	}
}

func setControlCentreIcon(systemTray *application.SystemTray, controller *Controller) {
	state, err := controller.GetAppState()
	target := controlCentreTarget{}
	if err == nil {
		target = controlCentreDevice(state)
	}
	if runtime.GOOS == "darwin" {
		systemTray.SetTemplateIcon(controlCentreTrayIcon(target))
		return
	}
	systemTray.SetIcon(controlCentreTrayIcon(target))
}

func controlCentreTrayIcon(target controlCentreTarget) []byte {
	if !target.connected {
		return trayPlainIconPNG
	}
	if target.paused {
		return trayPausedIconPNG
	}
	return trayRunningIconPNG
}

func buildControlCentreMenu(wailsApp *application.App, controller *Controller, window application.Window, systemTray *application.SystemTray) *application.Menu {
	state, err := controller.GetAppState()
	menu := wailsApp.NewMenu()
	menu.Add("TimeFlip Desktop").SetEnabled(false)
	if err != nil {
		menu.Add(fmt.Sprintf("State unavailable: %s", err)).SetEnabled(false)
	} else {
		menu.Add(controlCentreStatusLabel(state)).SetEnabled(false)
	}

	menu.AddSeparator()
	menu.Add("Open Window").OnClick(func(*application.Context) {
		window.Show().Focus()
	})
	menu.Add("Refresh").OnClick(func(*application.Context) {
		wailsApp.Event.Emit("shell.refresh")
		setControlCentreIcon(systemTray, controller)
		systemTray.SetMenu(buildControlCentreMenu(wailsApp, controller, window, systemTray))
	})

	target := controlCentreDevice(state)
	toggleLabel := "Pause Tracking"
	if target.paused {
		toggleLabel = "Resume Tracking"
	}
	menu.Add(toggleLabel).SetEnabled(target.ok).OnClick(func(*application.Context) {
		if !target.ok {
			return
		}
		if err := controller.SetPaused(target.deviceID, !target.paused); err != nil {
			wailsApp.Event.Emit("shell.error", err.Error())
			return
		}
		wailsApp.Event.Emit("shell.refresh")
		setControlCentreIcon(systemTray, controller)
		systemTray.SetMenu(buildControlCentreMenu(wailsApp, controller, window, systemTray))
	})
	lockLabel := "Lock Orientation"
	if target.locked {
		lockLabel = "Unlock Orientation"
	}
	menu.Add(lockLabel).SetEnabled(target.ok).OnClick(func(*application.Context) {
		if !target.ok {
			return
		}
		if err := controller.SetLocked(target.deviceID, !target.locked); err != nil {
			wailsApp.Event.Emit("shell.error", err.Error())
			return
		}
		wailsApp.Event.Emit("shell.refresh")
		setControlCentreIcon(systemTray, controller)
		systemTray.SetMenu(buildControlCentreMenu(wailsApp, controller, window, systemTray))
	})

	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(*application.Context) {
		wailsApp.Quit()
	})
	return menu
}

func controlCentreStatusLabel(state services.AppState) string {
	target := controlCentreDevice(state)
	if state.CurrentSession != nil {
		if target.locked {
			return "Locked: " + state.CurrentSession.TaskLabelSnapshot
		}
		return "Tracking: " + state.CurrentSession.TaskLabelSnapshot
	}
	if target.ok && target.paused && target.locked {
		return "Paused and locked"
	}
	if target.ok && target.paused {
		return "Paused"
	}
	if target.ok && target.locked {
		return "Locked"
	}
	if len(state.States) > 0 {
		return fmt.Sprintf("%s, facet %d", state.States[0].ConnectionState, state.States[0].CurrentFacet)
	}
	if len(state.Devices) > 0 {
		return "No active session"
	}
	return "No device configured"
}

type controlCentreTarget struct {
	deviceID  string
	paused    bool
	locked    bool
	connected bool
	ok        bool
}

func controlCentreDevice(state services.AppState) controlCentreTarget {
	if len(state.States) > 0 {
		return controlCentreTarget{
			deviceID:  state.States[0].DeviceID,
			paused:    state.States[0].Paused,
			locked:    state.States[0].Locked,
			connected: state.States[0].ConnectionState == domain.ConnectionConnected,
			ok:        true,
		}
	}
	if len(state.Devices) > 0 {
		return controlCentreTarget{deviceID: state.Devices[0].ID, ok: true}
	}
	return controlCentreTarget{}
}

func openBLETrace(path string) (io.Writer, func(), error) {
	if path == "" {
		return nil, func() {}, nil
	}
	if path == "-" {
		return os.Stderr, func() {}, nil
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return file, func() {
		_ = file.Close()
	}, nil
}
