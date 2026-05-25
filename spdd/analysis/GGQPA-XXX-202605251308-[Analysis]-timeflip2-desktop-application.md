# SPDD Analysis: Timeflip2 Desktop Application

## Original Business Requirement
# TimeFlip2 desktop application

## Background

I have a TimeFlip2 device which is a piece of hardware that is used to track time spent on tasks. The hardware itself is a regular dodecahedron shape containing accelerometers that detect the orientation of the device, and when it is phsyically tapped against a hard surface.

The device normally communicates over BLE (Bluetooth Low Energy). The app then interprets the orientation and movements of the device as signals to start tracking time, stop tracking time, and what task to track time against. The user can label each face of the dodecahedron with labels or symbols that indicate different tasks, e.g. "documentation", "coding", "admin". The face (facet) that is facing up is considered active.

I have built a [Golang module](https://github.com/mitchellrj/timeflip-go/) which supports the TimeFlip device and interactions with it, now I want to build a desktop app to use instead of the default mobile app.

Consider [the description of the mobile app and its features](https://timeflip.io/quickstartguide) as inspiration for this project.

Use the demo CLI app in the `timeflip-go` module as an example of how to interface with the module and form low-level user journeys.

## Decision drivers

1. Independence from cloud APIs and mobile app.
2. Platform and architecture portability (Mac first)
3. Quality of user experience.

## Scope In

* Pairing a new device.
* Unpairing a device.
* Assigning tasks to facets (with labels, icons, and colours).
* Assigning facets to Pomodoro with configurable time.
* Showing the current state of the device's time tracking features: active facet, paused / unpaused, locked / unlocked.
* Pausing / unpausing task tracking.
* Viewing the history of the device.
* Local storage of any configuration and history in a SQLite database.
* A control centre icon (on Mac) or task bar icon (Windows) indicating current state and which can launch the full app window.
* Selecting a UI framework (Wails, QT, etc).

## Scope Out

* Cloud storage & API integrations.
* App installer.
* Running on startup / as a daemon.
* User notifications.
* Windows / Linux implementation for now.

## Acceptance Criteria (ACs)

1. User can pair supported devices in range.
   **Given** A TimeFlip2 device is in range.
   **When** A user initiates pairing by selecting a device from a list of those in range.
   **Then** The pairing process is executed, and error states revealed and recovered from where possible.

2. User can unpair supported devices.
   **Given** A TimeFlip2 device is already paired (not necessarily in range).
   **When** A user initiates unpairing by device ID.
   **Then** The unpairing process (specific to the TimeFlip2 device where appropriate) is initiated, and following stages can be executed, and error states revealed and recovered from where possible.

3. User can view current configuration of each facet.
   **Given** A TimeFlip2 device is paired and in range.
   **When** A consuming app requests to read data by device ID.
   **Then** The consuming app receives details of the requested data, or an appropriate error.

4. User can view current state of the device.
   **Given** A TimeFlip2 device is paired and in range.
   **When** A user wishes to view the current state of the device in detail.
   **Then** The app displays this in a user-friendly way.

5. User can configure each facet.
   **Given** A TimeFlip2 device is paired and in range.
   **When** A consuming app requests to write configuration by device ID.
   **Then** The consuming app writes the configuration to the device, and receives either a success status after confirming the state or appropriate error.

6. User can view the history of activity.
   **Given** A TimeFlip2 device is paired and in range.
   **When** An event is emitted by the device (e.g. orientation change).
   **Then** The consuming app receives the event.

7. User can configure tap / pause behaviour.
   **Given** A TimeFlip2 device is paired and in range.
   **When** User configures the device.
   **Then** The desired configuration is reflected on the device.

8. Automatic connection.
   **Given** A configured TimeFlip2 device is paired.
   **When** The app is launched or the device comes into range.
   **Then** The app automatically connects to the device.

## Domain Concept Identification

### Existing Concepts (from codebase)
- SPDD Workflow Scaffolding: local prompt templates define the analysis, REASONS Canvas, generation, sync, and update workflow for turning this requirement into implementation prompts.
- Go Quality Envelope: local lint, pre-commit, Dependabot, CI, CodeQL, and release workflow files assume a Go module at the repository root, with Go formatting, tests, `golangci-lint`, `govulncheck`, and cross-platform checks as the project quality boundary.

### Existing Concepts (from referenced `timeflip-go` module)
- TimeFlip Client: platform-neutral library entrypoint for discovering devices, connecting sessions, pairing, and unpairing.
- BLE Transport: abstraction over scan, connect, OS pairing, and OS unpairing; the referenced module already provides a macOS CoreBluetooth-backed transport.
- Device and Session: a discovered BLE peripheral becomes an active session that can authorize, read state, write configuration, stream events, and close the connection.
- Pairing Workflow: staged connect, optional OS pairing, password authorization, optional password change, and verification flow with stage-level results and manual-action recovery.
- Unpairing Workflow: staged device-side reset and OS unpairing flow, including manual actions when OS-level support is not available.
- Device State: readable device information, battery, system state, tracker status, lock state, pause state, auto-pause setting, and current facet state.
- Facet and Task Parameters: low-level facet identifiers, assignment state, task mode, Pomodoro limit, elapsed seconds, and facet colour support.
- Tap Settings: low-level double-tap accelerometer configuration exposed by the library.
- History Entry: decoded device history containing event number, facet, pause flag, timestamp, duration, previous event number, and raw payload.
- Event Stream: technical device events over Go channels, including facet, double-tap, battery, system state, history, command result, connection state, and raw events.
- Manual Action: library-level indication that the caller or user must complete OS pairing or unpairing outside the direct API.
- Demo CLI Journey: existing command flow for list, select, pair, connect, authorize, read, write, stream, stop, close, unpair, and exit.

### New Concepts Required
- Desktop Application Shell: the primary user-facing application frame, navigation, lifecycle, and platform integration boundary.
- Device Profile: local record of a known TimeFlip2 device, including device ID, friendly name, pairing status, preferred protocol, stored TimeFlip password, and connection preferences.
- Connection Manager: app-owned orchestration for scanning, reconnecting, session lifecycle, authorization from stored app config, offline state, and recovery from lost BLE connections.
- Facet Assignment: user-facing mapping between a physical facet and local task metadata such as label, icon, colour, optional Pomodoro behaviour, and pause-side intent.
- Task: app-owned time-tracking category shown to users and associated with one or more facet assignments over time.
- Pomodoro Configuration: local and device-backed timing intent for a facet, including configured limit and behaviour when the Pomodoro side is active.
- Tracking State: interpreted user-facing state derived from device status, event stream, pause state, lock state, current facet, and local task mapping.
- Tracking Event Log: local event history normalized from device history and live events into task sessions first, with summary reporting deferred.
- Local SQLite Store: persistence boundary for device profiles, facet assignments, task metadata, settings, event history, and any sync checkpoints.
- Tray or Menu Bar Presence: lightweight platform status surface that reflects current task/state and can open the full app window.
- UI Framework Boundary: selected desktop framework that determines how Go services, BLE permissions, tray integration, and frontend state communicate.
- Error Recovery Model: user-facing states and actions for Bluetooth permission denial, device out of range, wrong password, unsupported OS action, stale connection, low battery, and manual OS pairing/unpairing.
- History Reconciliation Policy: rules for combining onboard device history, live events, and locally stored records without duplication or misleading durations.

### Key Business Rules
- Cloud independence: all configuration and history needed by the desktop app must be stored locally and must not depend on TimeFlip cloud APIs or the mobile app.
- Device ownership boundary: the `timeflip-go` library remains stateless; the desktop app owns persistence of devices, task labels, user preferences, sync checkpoints, and history.
- Password persistence: TimeFlip passwords are stored in app configuration for the initial implementation so automatic reconnection and authorization can work without repeated prompts.
- Local labels and icons: facet labels and icons are local-only app metadata; they must not be written to device task parameters unless a future requirement explicitly changes that boundary.
- Facet activation: the face currently reported as up maps to an active local task unless tracking is paused, locked, undefined, or mapped to an idle/pause behaviour.
- Pause side semantics: a pause side is represented as a local facet task assignment, not as a separate device mode.
- Pairing transparency: pairing and unpairing must expose staged progress and recoverable manual actions rather than collapsing BLE and OS-level failures into generic errors.
- Configuration confirmation: writes to device configuration should be confirmed by subsequent readback or an equivalent success state before the UI treats them as durable.
- Event interpretation: low-level events from the library are technical signals; the app must interpret them into user-facing tracking states using local task and facet mapping.
- History integrity: imported device history and live events must preserve event ordering and avoid duplicate time entries when reconnecting after offline use.
- Historical assignment stability: facet reassignment affects only new records; existing task-session history retains the task assignment that was active when those records were created.
- Mac-first portability: the first implementation should optimize for macOS while preserving architecture boundaries for later Windows and Linux work.
- No unsupported background promise: startup daemon behaviour, notifications, installer work, and Windows/Linux implementation are explicitly outside the current requirement.

## Strategic Approach

### Solution Direction
- Build a Mac-first Go desktop application around the existing `timeflip-go` module, treating BLE/device operations as a service boundary and local SQLite as the source of truth for user-facing configuration and history.
- Start with a vertical slice that mirrors the demo CLI journey in a desktop workflow: scan, select, pair, connect, authorize, read state, stream events, write basic facet/tap/pause settings, import history, and unpair.
- Keep device state and user configuration separate: device-facing values come from `timeflip-go`, while labels, icons, task naming, stored TimeFlip passwords, local history views, and UI preferences live in SQLite-backed app configuration.
- Model the app as event-driven: BLE scans and sessions produce status updates, read results, write confirmations, and live events that feed a connection manager and persistent event log.
- Use the official mobile app behaviour as product inspiration only: local desktop UX should cover task assignment, Pomodoro, pause/play, lock awareness, auto-pause/tap settings, history, and settings without adopting cloud-backed report/export assumptions.

### Key Design Decisions
- UI framework selection: Wails offers a natural Go backend plus web frontend path and fits the existing Go quality envelope; Qt/Fyne may provide more native widgets but increase packaging, binding, and cross-platform complexity. Recommendation: prefer Wails unless a specific native menu-bar or Bluetooth-permission constraint blocks it during REASONS Canvas validation.
- Persistence boundary: store app-owned configuration and history in SQLite rather than relying on device memory or external services. Trade-off: the app must define migrations, reconciliation, and backup semantics, but gains cloud independence and a durable desktop experience.
- Device library ownership: consume `timeflip-go` as an external module rather than copying BLE logic into this repo. Trade-off: upstream API changes need version management, but this preserves a clean separation between protocol/device concerns and desktop product behaviour.
- State derivation strategy: present a user-facing tracking model derived from device status, current facet, local assignments, pause/lock flags, and event stream. Trade-off: more app logic is required, but the UI can avoid exposing raw protocol concepts to users.
- History strategy: combine live events with explicit history reads on connection and after reconnect. Trade-off: reconciliation is more complex, but it handles the device's offline/onboard memory model and avoids gaps when the app has not been running.
- Initial history view: render interpreted task sessions as the first history surface, with summary reporting treated as a later capability. This keeps the first release focused on trustable session boundaries rather than aggregate analytics.
- Reassignment policy: snapshot the active task assignment into each new task session so later facet reassignment does not rewrite history. Trade-off: this requires explicit session records, but it preserves user trust and auditability.
- Reconnection defaults: after launch or disconnect, scan immediately, then retry every 15 seconds for the first 2 minutes, every 60 seconds until 15 minutes, and every 5 minutes thereafter while the app remains open. Mark the device as offline after 3 consecutive failed connection attempts or 2 minutes without a successful reconnect, but keep background retry active in the open app.
- Pairing and unpairing UX: expose stage-based progress and manual OS actions from the library directly in the UI. Trade-off: this is more verbose than a one-click wizard, but it improves recoverability for Bluetooth permission, OS pairing, wrong password, and out-of-range failures.
- Platform scope: implement macOS first and isolate platform-specific tray/menu-bar and BLE permission assumptions. Trade-off: Windows/Linux are deferred, but architectural portability remains credible.

### Alternatives Considered
- Reimplement BLE protocol in the desktop app: rejected because the referenced Go library already provides protocol orchestration, macOS transport, examples, tests, and staged pairing/unpairing semantics.
- Treat the device as the sole history store: rejected because the requirement explicitly calls for local SQLite storage and because offline/onboard history still needs reconciliation and user-facing interpretation.
- Build a cloud-compatible clone of the official app: rejected because cloud storage and API integrations are out of scope and conflict with the independence decision driver.
- Implement the tray/menu-bar app before the full window: rejected as a first slice because pairing, configuration, and history require richer workflows; the lightweight status surface should follow the core connection and tracking model.
- Start with Windows/Linux portability work: rejected because Windows/Linux implementation is currently out of scope, while macOS support is directly enabled by the referenced module.

## Risk & Gap Analysis

### Requirement Ambiguities
- UI framework is intentionally undecided; the next SPDD phase must choose one and define the backend/frontend boundary, build tooling, and tray/menu-bar support.
- The requirement says "assigning tasks to facets with labels, icons, and colours"; labels and icons are now explicitly local-only app concepts, while colour may be both local metadata and a device-backed facet colour when supported.
- Pomodoro behaviour needs clarification: the device has task mode and Pomodoro limit concepts, while the desktop UX must decide whether Pomodoro is purely device-backed, locally displayed, or both.
- Pause, lock, and auto-pause interaction rules need to be made explicit, especially when a locally assigned pause side, tap gesture, app pause button, and lock state overlap.
- History display starts as interpreted task sessions; summary reporting, day totals, statistics, and editable entries are later concerns unless pulled into the first release.
- The acceptance criteria use "consuming app" language from a library requirement; the REASONS Canvas should translate this into desktop-user workflows and UI outcomes.
- Password handling is clarified for the initial implementation: store TimeFlip passwords in app config, with later hardening such as macOS Keychain left outside the first design unless security requirements change.

### Edge Cases
- No devices found, unsupported devices found, multiple supported devices found, weak RSSI, duplicate advertised names, or selected device no longer in range.
- macOS Bluetooth permission denied, Bluetooth powered off, OS pairing unsupported by adapter, or manual OS pairing/unpairing required.
- Device has a non-default password, wrong password attempts fail authorization, or a user changes the password but the app loses connection before verification.
- Device reports undefined facet, accelerometer error, low/depleted batteries, reset/system issue, sync required, stale advertised name, or protocol v3/v4 differences.
- Event stream drops while history import continues, duplicate history appears after reconnect, or local clock/display time differs from device event timestamps.
- A facet is reassigned after historical entries exist; existing task sessions must retain their original task assignment and only new records should use the updated assignment.
- Configuration write succeeds at command level but readback differs or is delayed; the UI needs pending, failed, and confirmed states.
- User unpairs a device that is offline: OS/manual unpairing may be possible, while device-side reset is not.

### Technical Risks
- The local repository currently lacks `go.mod`, application source, scripts referenced by pre-commit, and scripts referenced by CI; the scaffold assumes a Go project that has not been created yet.
- Desktop framework choice may affect SQLite access, frontend bundling, code signing expectations, macOS Bluetooth permission prompts, menu-bar integration, and test strategy.
- `timeflip-go` is available as a pseudo-version rather than a tagged stable release at the time of analysis; the desktop app should pin a known version and decide when to update.
- BLE workflows are inherently asynchronous and failure-prone; connection manager design must prevent stale sessions, leaked event streams, overlapping writes, and UI state races.
- Local history storage must be idempotent and migration-friendly; otherwise reconnects and onboard-history imports can corrupt or duplicate tracked time.
- Device passwords and raw BLE trace data are sensitive; because passwords are stored in app configuration for now, logs, diagnostics, support flows, and config handling must avoid exposing password values or password characteristic bytes.
- Testing real BLE behaviour in CI is not feasible; the design should rely on library fake transports, app-level fakes, and narrow hardware smoke tests.
- The official quickstart describes onboard memory and offline operation, but the exact history capacity and protocol behaviour need to be validated against `timeflip-go` and real hardware during implementation.

### Acceptance Criteria Coverage
- AC1 Pair supported device in range: addressable through `timeflip-go` scan, selected device, staged `Pair`, manual-action display, and UI recovery flows.
- AC2 Unpair supported device: addressable through stored device ID, staged `Unpair`, optional device reset, manual OS action display, and offline-device handling.
- AC3 View current facet configuration: addressable through active session reads plus local SQLite facet/task metadata; labels and icons are local-only, while device-backed task parameters and colours are shown where supported.
- AC4 View current device state: addressable through device info, battery, system state, tracker status, current facet, pause, lock, and UI-derived tracking state.
- AC5 Configure each facet: addressable through local assignment persistence plus `timeflip-go` write capabilities for supported device-backed values; readback confirmation must be designed.
- AC6 View history/activity: addressable through event streaming and explicit history reads rendered initially as interpreted task sessions; summary reporting is deferred.
- AC7 Configure tap/pause behaviour: addressable through tap settings, pause, auto-pause, and local pause-side assignment; overlapping behaviour still needs detailed interaction rules.
- AC8 Automatic connection: addressable through a connection manager using persisted device profiles, stored passwords, and the default scan/retry cadence; launch/startup constraints remain limited by the out-of-scope daemon/startup requirement.

### Open Questions for REASONS Canvas
- Which desktop framework should be selected for the first implementation, and what are the minimum required tray/menu-bar capabilities?
- What redaction rules and future migration path should protect stored TimeFlip passwords if the app later moves from app config to OS keychain storage?
- What exact session-boundary rules should convert raw facet, pause, tap, and reconnect events into task sessions?
- How should summary reporting be staged after task-session history is reliable?
- What is the minimum hardware smoke-test checklist before calling pairing, configuration writes, history sync, and event streaming complete?
