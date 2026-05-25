package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/mitchellrj/timeflip-desktop/internal/services"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

func Run() {
	controller, bus, err := BuildController(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	wailsApp := application.New(application.Options{
		Name:        "TimeFlip Desktop",
		Description: "Local TimeFlip2 task tracking",
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(os.DirFS("frontend/dist")),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
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
	if runtime.GOOS == "darwin" {
		systemTray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else {
		systemTray.SetLabel("TimeFlip")
	}

	rebuildMenu := func() {
		systemTray.SetMenu(buildControlCentreMenu(wailsApp, controller, window, systemTray))
	}

	rebuildMenu()
	systemTray.AttachWindow(window).WindowOffset(6)

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
		systemTray.SetMenu(buildControlCentreMenu(wailsApp, controller, window, systemTray))
	})

	deviceID, paused, hasDevice := controlCentreDevice(state)
	toggleLabel := "Pause Tracking"
	if paused {
		toggleLabel = "Resume Tracking"
	}
	menu.Add(toggleLabel).SetEnabled(hasDevice).OnClick(func(*application.Context) {
		if !hasDevice {
			return
		}
		if err := controller.SetPaused(deviceID, !paused); err != nil {
			wailsApp.Event.Emit("shell.error", err.Error())
			return
		}
		wailsApp.Event.Emit("shell.refresh")
		systemTray.SetMenu(buildControlCentreMenu(wailsApp, controller, window, systemTray))
	})

	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(*application.Context) {
		wailsApp.Quit()
	})
	return menu
}

func controlCentreStatusLabel(state services.AppState) string {
	if state.CurrentSession != nil {
		return "Tracking: " + state.CurrentSession.TaskLabelSnapshot
	}
	if _, paused, ok := controlCentreDevice(state); ok && paused {
		return "Paused"
	}
	if len(state.States) > 0 {
		return fmt.Sprintf("%s, facet %d", state.States[0].ConnectionState, state.States[0].CurrentFacet)
	}
	if len(state.Devices) > 0 {
		return "No active session"
	}
	return "No device configured"
}

func controlCentreDevice(state services.AppState) (string, bool, bool) {
	if len(state.States) > 0 {
		return state.States[0].DeviceID, state.States[0].Paused, true
	}
	if len(state.Devices) > 0 {
		return state.Devices[0].ID, false, true
	}
	return "", false, false
}
