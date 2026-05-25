# Timeflip2 desktop application

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
