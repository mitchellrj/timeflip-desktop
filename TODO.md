# TODO

This file tracks intentional first-pass gaps from `/spdd-generate` for `spdd/prompt/GGQPA-XXX-202605251333-[Feat]-desktop-timeflip2-app.md`.

## Device And Hardware

- Verify the `timeflip-go` adapter against a real TimeFlip2 device on macOS for scan, pair, authorize, state reads, facet writes, history import, event streaming, and unpair.
- Add a documented manual hardware smoke-test command or script once the first real-device pass has been completed.
- Expand device write readback handling so partial readback failures produce richer per-field confirmation state.

## Desktop Shell

- Replace the minimal menu-bar service boundary with real macOS menu-bar integration if Wails support is sufficient, or document the chosen plugin/native bridge.
- Generate Wails bindings as part of the normal frontend workflow once the Wails CLI is installed in the developer environment.
- Replace the static `frontend/dist/index.html` fallback with a built artifact from `frontend/src` during normal Wails builds.

## Frontend

- Wire all pairing, unpairing, facet editing, password update, and settings forms to the backend API. The current frontend is a first dashboard shell and state viewer.
- Add frontend tests or browser smoke tests after the Wails/Vite build path is settled.
- Add a polished password update flow that never displays the stored value.

## Persistence And Security

- Decide whether app-config password storage should migrate to macOS Keychain before any broader release.
- Add explicit redaction tests for all public controller DTOs and logger fields.
- Add schema migrations for history checkpoints once real hardware confirms the most reliable checkpoint key.

## Tracking Semantics

- Refine session-boundary rules with real event streams, especially lock state, undefined facets, double taps, and reconnect timing.
- Add task-session conflict repair tooling if duplicate or overlapping sessions are observed during hardware testing.
- Stage summary reporting only after task-session reconciliation is stable.

