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
