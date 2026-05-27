# SPDD Analysis: UI Experience Improvements

## Original Business Requirement
/spdd-analysis Assess the UI design and experience. Recommend and then implement improvements to:
* Flow
* Widget selection
* Text and copy
* Use of icons, imagery, and colours

Consider different usage journeys:
* Brand new & pairing
* First time configuration
* Routine use and time tracking
* Later configuration
* Advanced settings, unpairing / reset

## Domain Concept Identification

#### Existing Concepts (from codebase)
- Device: A known or discovered TimeFlip2 unit that can be scanned, connected, paired, renamed, configured, and unpaired. It is the anchor for pairing, state, tap settings, LED settings, facet configuration, and task sessions.
- Device state: The live interpretation of connection, current facet, lock, pause, firmware, and battery state. It is the primary routine-use concept because it tells the user whether tracking is active and what is being tracked.
- Pairing workflow: A staged device authorization process that can involve OS pairing prompts, passwords, and manual recovery actions. It owns the brand-new and reset journeys.
- Facet configuration: The local assignment from one of twelve physical sides to a task, pomodoro task, or pause side, including label, icon, colour, and confirmation status.
- Task: A local tracking label with an icon and colour that can be reused across facets and sessions.
- Task session: A recorded interval derived from facet changes, including start, end, duration, paused time, and task snapshots. It supports routine review and later reporting.
- Tap settings: Device-backed tuning values for tap detection, with preview, presets, detection count, temporary application, confirmation, reset, and cancel states.
- LED settings: Device-backed brightness and blink configuration.
- App settings: Local communication timeout and reconnect policy values stored in local config.
- Password and unpair controls: Rare but high-risk controls for password updates, factory reset, and OS unpairing.

#### New Concepts Required
- Journey status: A user-facing summary that frames the current screen as setup, configuration, tracking, review, or advanced maintenance without adding new persistence.
- Setup progress: A lightweight visual grouping of scan, pair/connect, configure facets, and track steps. It clarifies the brand-new path while remaining useful for later configuration.
- Advanced settings disclosure: A presentation boundary that keeps low-frequency reconnect, password, and reset controls available but visually subordinate to daily tracking and configuration.
- Icon option set: A curated UI-level vocabulary for common task icons so users select recognizable symbols instead of typing opaque icon names.

#### Key Business Rules
- Local-first boundaries must remain visible: task labels, local icons, colours, passwords, settings, and history are app-owned; device confirmation applies only when a device is connected and writes succeed.
- Brand-new users must be able to discover a device before they understand every device setting.
- First-time configuration must prioritize facet assignment and task naming before advanced tuning.
- Routine users need the current task, connection, lock, pause, and elapsed session context above configuration forms.
- Later configuration must allow changing facets, tap feel, LED behaviour, and device name without forcing a full pairing flow.
- Unpairing, factory reset, and password updates must be present but visually separated from routine controls.

## Strategic Approach

#### Solution Direction
- Preserve the existing Wails/React single-page app and controller API boundaries while restructuring the UI into clearer journeys. Use frontend presentation changes first: stronger dashboard summary, setup-progress context, better section copy, more purposeful buttons, curated icon selection, status chips, compact session rows, and advanced disclosures.
- Keep the left navigation and local desktop tool feel, but make the first visible content answer "what is happening now?" and "what should I do next?".
- Reuse the existing `lucide-react` dependency for clearer action icons and task icon previews instead of introducing new assets or image dependencies.
- Use the existing colour values as a foundation but broaden the palette and role mapping so green is not the only dominant signal. Reserve green for successful/primary actions, amber for attention/unsaved states, red for destructive actions, and neutral blue/ink for navigation and information.

#### Key Design Decisions
- Single page versus wizard: A wizard would simplify first run but slow down routine use and later configuration. Keep the single page and add journey grouping plus contextual calls to action.
- Always-visible advanced controls versus disclosure: Always-visible password/unpair/retry values increase cognitive load and risk. Put rare or risky controls behind explicit advanced sections while keeping them accessible.
- Free-text icon entry versus curated selection: Free-text is flexible but unclear. Use a select with common task icons and render matching lucide previews, while retaining simple string storage semantics.
- Raw technical labels versus user-oriented copy: Values such as "register ticks" are accurate but unfriendly. Preserve units where needed, but add intent-led labels and helper copy around presets and preview flows.
- More imagery versus product-tool restraint: The app is an operational desktop tool for hardware tracking. Avoid decorative imagery and instead use the actual domain object in copy, iconography, colour swatches, and status presentation.

#### Alternatives Considered
- Full visual redesign: Rejected because this is already a functioning local desktop tool and the safest improvement is to clarify existing flows without destabilizing API behaviour.
- New routing model: Rejected for now because anchors and sections are adequate; the main weakness is hierarchy, not navigation infrastructure.
- Adding a design system package: Rejected because the existing app is small and already uses lucide icons; scoped CSS and components are sufficient.

## Risk & Gap Analysis

- The screenshots show real-device states, but automated verification may run without the Wails backend or hardware. Visual verification should focus on layout resilience and build correctness; hardware behaviour remains a manual validation item.
- Pairing and unpairing can require OS prompts and manual actions. Copy must avoid implying that the app can complete every recovery step silently.
- Facet colours are confirmed on-device only when connected; the UI must not hide the difference between local saved values and device-confirmed values.
- Tap tuning has temporary preview state; the UI must make it clear that preview, save, reset, and cancel have different permanence.
- Passwords are stored locally but not displayed. Password update copy must avoid revealing stored values and should not encourage unnecessary changes.
- Current settings names are technically precise but may be ambiguous to non-developers. Advanced disclosure reduces prominence but does not fully explain each reconnect parameter.
- The existing single-page layout can still become long on small screens. Responsive CSS must keep action groups, session metadata, and facet controls readable without overlapping.
- Acceptance criteria coverage: flow, widgets, copy, icons, colours, brand-new pairing, first configuration, routine use, later configuration, and advanced reset/unpair are all addressable through frontend changes within the current architecture.
