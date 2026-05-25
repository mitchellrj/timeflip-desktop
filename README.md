# TimeFlip Desktop

Local-first desktop tracking for TimeFlip2 devices.

This app pairs with a TimeFlip2, stores local task/facet configuration, tracks task sessions from facet changes, and provides a Wails desktop shell with a system tray control centre.

## Status

This is a first-pass desktop application generated from the SPDD prompt in `spdd/prompt/GGQPA-XXX-202605251333-[Feat]-desktop-timeflip2-app.md`.

The core app shell, persistence layer, controller API, frontend forms, and Wails v3 tray menu are in place. Real-device verification is still required before treating the app as release-ready. Remaining gaps are tracked in `TODO.md`.

## Tech Stack

- Go backend with local SQLite persistence via `modernc.org/sqlite`.
- TimeFlip device integration via `github.com/mitchellrj/timeflip-go`.
- Wails v3 alpha desktop shell and system tray menus.
- React/Vite frontend with generated Wails v3 bindings.

## Repository Layout

- `internal/app/` - Wails runner, controller, bootstrap, and event bridge.
- `internal/device/` - TimeFlip device client adapter.
- `internal/services/` - device, tracking, task, config, history, and reconnect services.
- `internal/store/` - SQLite store and migrations.
- `frontend/src/` - React application.
- `frontend/bindings/` - generated Wails v3 frontend bindings.
- `docs/desktop-shell.md` - Wails desktop shell and tray-menu notes.
- `TODO.md` - remaining first-pass tasks and manual QA.

## Prerequisites

- Go matching the module version in `go.mod`.
- Node.js and npm.
- macOS for the current native BLE path and tray QA.

Wails is pinned as a Go module dependency (`github.com/wailsapp/wails/v3 v3.0.0-alpha.95`), and bindings are generated through the local module cache.

## Development

Install/build frontend assets:

```sh
scripts/dev/frontend-build.sh
```

Regenerate Wails bindings after changing exported controller methods or DTOs:

```sh
scripts/dev/wails-generate.sh
```

Run frontend tests:

```sh
npm --prefix frontend test
```

Run Go tests:

```sh
go test ./...
```

Build all Go packages:

```sh
go build ./...
```

## Running

Build frontend assets first, then run either app entry point:

```sh
scripts/dev/frontend-build.sh
go run .
```

or:

```sh
go run ./cmd/timeflip-desktop
```

The Wails app serves `frontend/dist`, so refresh the frontend build after UI changes.

Bluetooth tracing can be enabled with the same flag shape as the demo CLI:

```sh
go run . -trace-ble trace.log
```

Use `-trace-ble -` to write the trace to stderr. BLE traces include raw reads, writes, notifications, and password bytes, so treat trace files as sensitive.

## Control Centre

The desktop control centre uses Wails v3 system tray menus. The tray menu shows current status and provides:

- Open Window
- Refresh
- Pause Tracking / Resume Tracking
- Quit

The main window behaves like a normal macOS app window and hides on close. Details are in `docs/desktop-shell.md`.

## Data And Security Notes

- App data is stored under the user config directory in `timeflip-desktop/timeflip-desktop.sqlite`.
- TimeFlip passwords are stored in the local app config/database for now.
- Stored passwords are not returned through public controller DTOs and are not displayed in the frontend.
- Migration to macOS Keychain remains a tracked follow-up in `TODO.md`.

## Manual QA Still Needed

Before release, verify with a real TimeFlip2 device on macOS:

- Scan, pair, authorize, connect, disconnect, and unpair.
- Facet writes and readback behaviour.
- Event streaming and history import.
- Task-session reconciliation from real device events.
- Tray icon visibility, menu refresh, normal app switching/focus behaviour, close-to-hide, pause/resume, and explicit Quit.
