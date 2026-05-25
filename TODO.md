# TODO

This file tracks intentional first-pass gaps from `/spdd-generate` for `spdd/prompt/GGQPA-XXX-202605251333-[Feat]-desktop-timeflip2-app.md`.

## Device And Hardware

- Verify the `timeflip-go` adapter against a real TimeFlip2 device on macOS for scan, pair, authorize, state reads, facet writes, history import, event streaming, and unpair.
- Add a documented manual hardware smoke-test command or script once the first real-device pass has been completed.
- Expand device write readback handling so partial readback failures produce richer per-field confirmation state.

## Desktop Shell

- Manually QA the Wails v3 system tray control centre on macOS: tray icon visibility, menu refresh, normal app-switcher/window focus behaviour, close-to-hide behaviour, pause/resume action, and explicit Quit.
- Decide whether to keep tracking Wails v3 alpha releases directly or pin a reviewed upgrade cadence before packaging.
- Consider adding frontend-specific DTOs with string timestamps if Wails binding generation warnings for `time.Time` become noisy. The current generated bindings work and type these fields as `any`.

## Frontend

- Run a manual browser/Wails smoke test for the full frontend flow once a real or simulated device path is available: scan, pair, connect, facet save, pause/resume, password update, unpair, and settings save.
- Add broader component or browser automation coverage after the Wails v3 tray/window lifecycle is stable enough for repeatable app launch tests.

## Persistence And Security

- Decide whether app-config password storage should migrate to macOS Keychain before any broader release.
- Add explicit redaction tests for all public controller DTOs and logger fields.
- Add schema migrations for history checkpoints once real hardware confirms the most reliable checkpoint key.

## Tracking Semantics

- Refine session-boundary rules with real event streams, especially lock state, undefined facets, double taps, and reconnect timing.
- Add task-session conflict repair tooling if duplicate or overlapping sessions are observed during hardware testing.
- Stage summary reporting only after task-session reconciliation is stable.
