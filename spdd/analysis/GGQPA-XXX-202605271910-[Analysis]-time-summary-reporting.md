# SPDD Analysis: Time Summary Reporting

## Original Business Requirement
# Time Summary Reporting

Issue: https://github.com/mitchellrj/timeflip-desktop/issues/6

## Background

TimeFlip Desktop currently records local task sessions from TimeFlip2 facet changes and shows a short list of recent sessions in the Track view. The existing history panel lists individual sessions with start time, duration, paused time, and end time, but it does not answer higher-level questions such as how much time was spent per task today, yesterday, this week, this month, or across a custom date range.

The current implementation already has the core data required for this feature:

- `domain.TaskSession` stores task snapshots, start/end times, duration seconds, paused seconds, and open pause state.
- `domain.TaskSessionFilter` supports device, task, facet, from, and to fields.
- `HistoryService.ListTaskSessions` delegates session listing to the store.
- `SQLiteStore.ListTaskSessions` filters sessions and orders them newest first.
- `Controller.GetAppState` currently loads all sessions into the frontend state.
- `frontend/src/main.jsx` renders the current Task Sessions panel and limits display to the first 12 sessions.

Affected area: tracking and history reporting across the Go domain/store/service/controller layers, generated Wails bindings for any exported controller changes, and the React Track view session/reporting UI.

## Scope In

- Add a summary experience above the detailed task-session history in the Track view.
- Support preset reporting periods for today, yesterday, this week, and this month.
- Support a custom reporting period with explicit start and end date/time fields.
- Aggregate sessions by task for the selected reporting period.
- Exclude paused time from reported task totals.
- Show task rows ordered by most active time to least active time.
- Show each task row with the task icon, task color, abbreviated active time, and a proportional bar using the task color.
- Show an empty state when no tracked task time exists for the selected period.
- Reject invalid custom periods where the start is after the end or the period has zero duration.
- Keep the visible inline history concise by limiting the default number of entries shown.
- Add a way to open a paginated detailed-history dialog filtered between two date/times.
- Ensure reporting uses live query/aggregation rather than storing precomputed aggregate rows.
- Include tests for period validation, active-time aggregation, paused-time exclusion, empty-period behavior, and filtered history listing.

## Scope Out

- No stored aggregate reporting tables or cached summaries.
- No export to CSV, spreadsheet, calendar, or external reporting tools.
- No charts beyond the requested proportional bars.
- No cross-device account sync or cloud reporting.
- No edits to historical sessions.
- No changes to how device events are imported or reconciled into task sessions except where needed to make reporting boundaries correct.
- No redesign of task/facet configuration outside the reporting and history surfaces.

## Acceptance Criteria

- Given I have tracked time on a single task during a day, when I view the report panel for that day, then I see only that task with the correct active time.
- Given I have tracked time on multiple tasks during a day, when I view the report panel for that day, then I see each task with its correct active time.
- Given I have tracked tasks over several days, when I select a preset or custom reporting period, then only sessions overlapping that period contribute to the totals.
- Given a tracked session includes paused time, when I view a report period covering that session, then the reported total excludes paused time.
- Given a report period contains no tracked task time, when I view the report panel, then it shows an empty state.
- Given I enter a custom period with a start after the end, when I try to apply it, then the period is rejected and the previous valid report remains available.
- Given I enter a custom period with zero duration, when I try to apply it, then the period is rejected.
- Given there are many detailed tracking entries, when I view the Track page, then only a concise recent subset is shown inline.
- Given I need more detail, when I open the detailed-history dialog and choose a date/time range, then entries are filtered to that range and paginated.
- Given task labels, icons, or colors have changed after a session was recorded, when I view the summary, then the recorded task snapshots remain usable for historical reporting.

## Domain Concept Identification

### Existing Concepts (from codebase)
- TaskSession: durable historical record of tracked work, including task snapshots, facet, session timing, elapsed duration, paused duration, and any open pause state; it is already created by tracking flows and displayed in the Track view.
- TaskSessionFilter: user-independent filtering concept for narrowing session history by device, task, facet, and time bounds; it is the existing bridge between history requests and stored sessions.
- Task: local time-tracking category with label, icon, color, archive state, and timestamps; it remains the current task catalog while historical reports rely on session snapshots for past labels, icons, and colors.
- FacetAssignment: mapping between a TimeFlip facet and local task metadata; it is the source of snapshots copied into new task sessions, preserving historical meaning after reassignment.
- Tracking State: current interpreted device state, including pause and active-session information; it matters because open sessions and open pauses can affect live reporting if the selected period includes the present.
- HistoryService: service boundary for imported/reconciled history and session listing; it is the natural business boundary for reporting behavior that remains based on live session data.
- SQLite Local Store: local persistence source for task sessions, tasks, device state, and configuration; it already owns session storage and filtering and must remain the source for report calculations.
- Controller and Wails Bindings: application bridge from Go services to the React frontend; any new report or detailed-history request visible to the UI must flow through this boundary.
- Track View Session UI: existing user surface for tracking status and task-session history; the summary report belongs here because it answers higher-level questions directly above the detailed entries.
- Frontend Formatting and State Helpers: existing frontend conventions for duration labels, date/time display, error messages, selected device state, and testable helper utilities.

### New Concepts Required
- Reporting Period: a user-selected time window, either preset or custom, that determines which task-session time contributes to the report.
- Reporting Preset: named period choices for today, yesterday, this week, and this month that translate user intent into concrete local date/time windows.
- Custom Reporting Period: user-entered start and end date/time values with validation and a "previous valid period remains available" behavior.
- Task Time Summary: aggregate row representing active tracked time for one historical task identity within a reporting period.
- Active Time: reportable task time after excluding paused time, including only the portion of a session that belongs to the selected reporting period.
- Empty Report State: explicit user-facing state for a valid period with no tracked active time.
- Detailed History Dialog: expanded history surface opened from the Track view, filtered by date/time range and paginated so the inline page remains concise.
- History Page: one slice of detailed session entries for a selected date/time filter, separate from the default concise inline list.
- Proportional Time Bar: visual representation of each task row's share relative to the most active task in the selected period.

### Key Business Rules
- Reporting must be derived from current session records at query time; no precomputed aggregate rows or cached summary tables may become a new source of truth.
- Summary totals are grouped by historical task identity and display the task snapshot recorded with each session, so later task label, icon, or color changes do not rewrite history.
- Only sessions that overlap the selected reporting period may contribute to totals; sessions outside the period must not affect summary rows or detailed filtered history.
- For sessions that traverse a reporting-period boundary, only the portion of the session that falls within the selected period contributes to that period's total.
- Paused time is excluded from active totals; an open pause can reduce active time when reporting includes a still-running session.
- Task rows are ordered from most active time to least active time to prioritize the user's main question.
- Tasks representing less than 1.5% of the total active time within the selected period are grouped into a visible "Other" summary row.
- Reports include tracked time from all devices by default.
- Week presets use locale-based week boundaries by default, with an app preference able to override the week-start behavior.
- Calendar period calculations must account for the locale's daylight-saving-time rules rather than assuming fixed 24-hour days.
- Summary reports update live while time is actively being tracked.
- Compact duration labels start with a simple abbreviated format: seconds for sub-minute totals, minutes for sub-hour totals, hours plus minutes for same-day totals, and days plus hours for multi-day totals.
- A valid period with no active tracked time must be shown as an empty report state rather than a blank or error state.
- Invalid custom periods where the start is after the end, or where start and end are equal, must be rejected without losing the previous valid report.
- The inline Track history remains a concise recent subset, while the detailed-history dialog provides larger filtered and paginated access with 20 entries per page.
- Reporting and detailed-history filters should respect local user date/time intent while stored session data remains durable and consistent across application restarts.

## Strategic Approach

### Solution Direction
- Build reporting as an extension of the existing local history model, using stored task sessions as the authoritative record and keeping aggregate results query-derived.
- Keep reporting concerns behind the existing Go service/controller boundary so the React Track view can request summary data and detailed history without owning persistence semantics.
- Treat the Track view as the user-facing reporting home: show period controls and task summaries above the existing recent-session list, and move larger history exploration into a focused dialog.
- Preserve the current historical snapshot model by displaying summary rows from session snapshots instead of resolving historical rows through the current mutable task catalog.
- Extend the existing test pattern across Go service/store behavior and frontend formatting/validation helpers so period validation, active-time math, empty states, and filtered history can be verified close to their responsibilities.

### Key Design Decisions
- Query-time aggregation vs. stored reports: query-time aggregation follows the requirement and avoids introducing migration-heavy derived data; the trade-off is that reporting must remain efficient and well-scoped for growing local history. Recommendation: derive summaries from task sessions on demand and optimize filtering before considering any future cache.
- Backend aggregation vs. frontend-only aggregation: backend aggregation centralizes overlap, pause exclusion, and historical-snapshot semantics; frontend-only aggregation would be quicker to prototype but risks duplicating business rules and loading excessive history into app state. Recommendation: place business reporting behavior in the Go history/reporting boundary and keep the frontend focused on controls and presentation.
- Period filtering semantics: the existing session filter concept is start-time oriented, while reporting requires overlap-oriented reasoning and boundary clipping. Recommendation: make the reporting concept explicit so period reports include only the active session portions that fall inside the requested window.
- Open-session handling: including open sessions makes the report feel live and matches the app's existing current-session display; excluding them simplifies calculations but makes "today" under-report active work while tracking is running. Recommendation: include open sessions where their active time overlaps the selected period, with tests covering current pause state.
- Inline history vs. dialog history: preserving a short inline list keeps the Track page scannable; putting filtered pagination in a dialog avoids turning the main page into a dense reporting table. Recommendation: keep inline history concise and add a dedicated detailed-history dialog for range and 20-entry page navigation.
- Preset period locality: presets such as today, yesterday, week, and month are user-facing calendar concepts, not raw UTC intervals. Recommendation: anchor preset selection in the user's locale, honor an app-level week-start preference when present, and account for locale daylight-saving-time transitions when deriving concrete period windows.
- Summary grouping identity: grouping by current task ID alone is simple but can blur historical label/icon/color changes; grouping purely by labels can split renamed tasks unpredictably. Recommendation: keep historical snapshot display primary and ensure grouping preserves the recorded task context users expect from past sessions.
- Minor task visibility: omitting very small task totals keeps the report scannable but can make totals appear incomplete; showing every tiny row can overwhelm the panel. Recommendation: group tasks below the 1.5% threshold into a visible "Other" row so the total remains explainable.
- Duration formatting: compact report labels should prioritize readability over precision. Recommendation: start with a sensible abbreviated format using the largest meaningful units, such as seconds for sub-minute values, minutes for sub-hour values, hours plus minutes for same-day totals, and days plus hours for multi-day totals.

### Alternatives Considered
- Add aggregate reporting tables: rejected because the requirement explicitly excludes stored aggregate rows and because historical correction would become harder to reason about.
- Reuse the app-state session array for all reports: rejected because the controller currently loads all sessions broadly, and reporting plus paginated history should move toward scoped requests rather than larger global state.
- Replace the recent-session list with the report: rejected because detailed session review remains useful and the requirement asks for a summary above detailed history, not instead of it.
- Introduce charting beyond proportional bars: rejected because the requested visual language is task rows with bars, and exports/charts are explicitly out of scope.
- Edit or reconcile historical sessions to make reporting easier: rejected because historical editing is out of scope and session reconciliation should only change if reporting boundaries expose a correctness issue.

## Risk & Gap Analysis

### Requirement Ambiguities
- "This month" likely means the user's local calendar month, but the exact inclusivity at the end boundary should be defined before implementation.
- Detailed-history filtering should include overlapping sessions, but the row presentation needs to make clear when a listed session extends outside the selected range.
- The requirement says custom period fields use explicit date/time values but does not specify keyboard, validation-message, or apply/cancel behavior.

### Edge Cases
- A session starts before a period and ends inside it.
- A session starts inside a period and ends after it.
- A session fully contains the period.
- A session is still open when the period ends or when the period includes the current time.
- A pause starts before the period, ends inside it, or is still open.
- A session has duration data that is absent, zero, or inconsistent with start/end timestamps.
- Multiple historical sessions share a task ID but have different snapshots because the task was renamed or recolored.
- Very short sessions and all-paused sessions may produce zero active time and should not create misleading report rows.
- Tasks below the 1.5% visibility threshold should contribute to "Other" without hiding tracked time from the report total.
- Daylight-saving changes can make calendar periods shorter or longer than a fixed number of seconds and must be handled according to the selected locale and any app preference.
- Long local history could make all-session loading and frontend-side filtering slow if the feature expands the current app-state approach.

### Technical Risks
- The existing list filter is start-time based; reporting needs overlap-based inclusion, which is a correctness risk for sessions crossing period boundaries.
- Paused time is currently stored as a session-level total plus optional open pause state; excluding only the paused portion inside a period may require more precision than a single total can provide for historical sessions with pauses that cross report boundaries.
- Open sessions depend on current time and current pause state, so deterministic tests need a controllable clock or equivalent service boundary.
- Generated Wails bindings must stay aligned with any exported controller changes, otherwise the frontend can compile against stale or missing APIs.
- Controller app state currently loads all sessions; adding reporting without reducing broad loading could worsen startup or refresh cost as history grows.
- Frontend report controls and dialog state may become dense in the existing single-file React app, raising maintainability risk unless logic is carefully separated into testable helpers.
- Timezone handling can diverge between JavaScript date inputs and Go time handling if the boundary contract is not made explicit.
- Paginated filtered history must avoid implying that the inline recent list is complete.

### Acceptance Criteria Coverage
- Single-task daily report: addressable through reporting periods, task summaries, active-time calculation, and historical snapshots.
- Multi-task daily report: addressable through grouping and sorting task summaries by active time.
- Multi-day preset/custom filtering: addressable with boundary-clipped session contributions, locale-aware calendar boundaries, and daylight-saving-aware period calculations.
- Paused-time exclusion: addressable for whole-session pause totals, but period-sliced pause exclusion has a precision risk that must be handled deliberately.
- Empty period behavior: addressable through explicit empty report state after valid period evaluation.
- Custom start after end rejection: addressable through custom-period validation and previous-valid-period preservation.
- Custom zero-duration rejection: addressable through the same validation path.
- Concise inline history: already partially implemented with a visible subset and should be preserved as an explicit UI rule.
- Detailed filtered paginated history: addressable by extending the existing history-listing concept with overlapping-session filters and a 20-entry page size.
- Snapshot use after task metadata changes: strongly supported by existing TaskSession snapshot fields and should remain a protected rule in reporting.

### Open Questions for REASONS Canvas
- How should the UI indicate that an overlapping detailed-history session starts before or ends after the selected range?
- How should paused portions be attributed when only total paused seconds are known for an old session that partially overlaps the selected period?
