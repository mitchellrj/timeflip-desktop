# Desktop Shell

## Wails Tooling

Wails bindings are generated with:

```sh
scripts/dev/wails-generate.sh
```

The generated Wails v3 bindings live under `frontend/bindings/` and are imported by `frontend/src/main.jsx`.

The frontend build is the source of truth for `frontend/dist/`:

```sh
scripts/dev/frontend-build.sh
```

The Wails app serves `frontend/dist` from `internal/app/runner.go`.

## Control Centre

The control centre uses Wails v3 system tray menus:

- Source: <https://v3.wails.io/features/menus/systray/>
- Current module: `github.com/wailsapp/wails/v3 v3.0.0-alpha.95`
- Shell entry point: `internal/app/runner.go`

The tray is created with `app.SystemTray.New()`, uses a plain sandtimer template icon while disconnected, switches to play/pause sandtimer variants while connected, and installs a dynamic menu with:

- Current tracking or connection status.
- Open Window.
- Refresh.
- Pause Tracking / Resume Tracking for the first known device.
- Quit.

The app uses `ActivationPolicyRegular` on macOS so the main window behaves like an ordinary application window: it appears in app switching, participates in normal focus ordering, and is not attached to the tray as a floating popover. Tray clicks open the control-centre menu. Window close hides the window instead of terminating the app; Quit remains explicit from the tray menu.

Because Wails v3 is still alpha, packaging and manual macOS tray QA remain tracked in `TODO.md`.
