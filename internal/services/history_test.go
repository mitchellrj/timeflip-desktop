package services

import (
	"context"
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/store"
	_ "modernc.org/sqlite"
)

func TestReconcileEventsToSessionsSkipsPreviouslyImportedHistory(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	st := store.NewSQLiteStore(db)
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveFacetAssignment(ctx, domain.FacetAssignment{
		ID:                "a1",
		DeviceID:          "d1",
		Facet:             1,
		TaskID:            "task-1",
		TaskLabelSnapshot: "Coding",
		TaskIconSnapshot:  "code",
		TaskColorSnapshot: "#2B6CB0",
		EffectiveFrom:     time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveFacetAssignment(ctx, domain.FacetAssignment{
		ID:                "a2",
		DeviceID:          "d1",
		Facet:             2,
		TaskID:            "task-2",
		TaskLabelSnapshot: "Review",
		TaskIconSnapshot:  "file-search",
		TaskColorSnapshot: "#69D2A5",
		EffectiveFrom:     time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 5, 25, 9, 0, 0, 0, time.UTC)
	history := NewHistoryService(st, nil, NewTrackingService(st, fixedClock{t: start}, &MemoryEventBus{}))
	events := []domain.DeviceEventRecord{
		{DeviceID: "d1", Kind: "history", Facet: 1, EventNumber: 1, OccurredAt: start, Source: "device_history"},
		{DeviceID: "d1", Kind: "history", Facet: 2, EventNumber: 2, OccurredAt: start.Add(30 * time.Minute), Source: "device_history"},
	}

	imported, err := history.ReconcileEventsToSessions(ctx, "d1", events)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 2 {
		t.Fatalf("expected first import to reconcile 2 events, got %d", imported)
	}
	imported, err = history.ReconcileEventsToSessions(ctx, "d1", events)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 0 {
		t.Fatalf("expected second import to skip duplicate events, got %d", imported)
	}

	sessions, err := st.ListTaskSessions(ctx, domain.TaskSessionFilter{DeviceID: "d1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected one closed and one open session, got %#v", sessions)
	}
}

func TestBuildTimeReportAggregatesBoundaryClippedActiveTime(t *testing.T) {
	st, closeStore := newHistoryTestStore(t)
	defer closeStore()
	ctx := context.Background()
	start := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	if err := st.SaveTaskSession(ctx, reportSession("s1", "coding", "Coding", start.Add(-time.Hour), start.Add(time.Hour), 1800)); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveTaskSession(ctx, reportSession("s2", "review", "Review", start.Add(15*time.Minute), start.Add(45*time.Minute), 0)); err != nil {
		t.Fatal(err)
	}
	history := NewHistoryService(st, nil, nil)
	from := start
	to := start.Add(time.Hour)
	now := to

	report, err := history.BuildTimeReport(ctx, domain.TimeReportRequest{From: &from, To: &to, Now: &now})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalActiveSeconds != 4500 {
		t.Fatalf("expected total active 4500, got %#v", report)
	}
	if len(report.Rows) != 2 {
		t.Fatalf("expected two rows, got %#v", report.Rows)
	}
	if report.Rows[0].Label != "Coding" || report.Rows[0].ActiveSeconds != 2700 {
		t.Fatalf("expected clipped Coding active time first, got %#v", report.Rows)
	}
	if report.Rows[1].Label != "Review" || report.Rows[1].ActiveSeconds != 1800 {
		t.Fatalf("expected Review active time second, got %#v", report.Rows)
	}
}

func TestBuildTimeReportGroupsMinorTasksAsOther(t *testing.T) {
	st, closeStore := newHistoryTestStore(t)
	defer closeStore()
	ctx := context.Background()
	start := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	if err := st.SaveTaskSession(ctx, reportSession("major", "major", "Major", start, start.Add(9900*time.Second), 0)); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveTaskSession(ctx, reportSession("minor", "minor", "Minor", start, start.Add(100*time.Second), 0)); err != nil {
		t.Fatal(err)
	}
	history := NewHistoryService(st, nil, nil)
	from := start
	to := start.Add(3 * time.Hour)
	now := to

	report, err := history.BuildTimeReport(ctx, domain.TimeReportRequest{From: &from, To: &to, Now: &now})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != 2 {
		t.Fatalf("expected major plus other rows, got %#v", report.Rows)
	}
	if report.Rows[1].Label != "Other" || !report.Rows[1].Other || report.Rows[1].ActiveSeconds != 100 {
		t.Fatalf("expected minor task grouped into Other, got %#v", report.Rows)
	}
}

func TestBuildTimeReportHandlesEmptyAndInvalidPeriods(t *testing.T) {
	st, closeStore := newHistoryTestStore(t)
	defer closeStore()
	ctx := context.Background()
	history := NewHistoryService(st, nil, nil)
	from := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)

	report, err := history.BuildTimeReport(ctx, domain.TimeReportRequest{From: &from, To: &to, Now: &to})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalActiveSeconds != 0 || len(report.Rows) != 0 {
		t.Fatalf("expected empty report, got %#v", report)
	}
	_, err = history.BuildTimeReport(ctx, domain.TimeReportRequest{From: &to, To: &from, Now: &to})
	if err == nil {
		t.Fatal("expected invalid period error")
	}
}

func TestBuildTimeReportUpdatesOpenSessionWithNow(t *testing.T) {
	st, closeStore := newHistoryTestStore(t)
	defer closeStore()
	ctx := context.Background()
	start := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	open := reportSession("open", "coding", "Coding", start, start, 0)
	open.EndedAt = nil
	open.DurationSeconds = 0
	pauseStarted := start.Add(30 * time.Minute)
	open.PauseStartedAt = &pauseStarted
	if err := st.SaveTaskSession(ctx, open); err != nil {
		t.Fatal(err)
	}
	history := NewHistoryService(st, nil, nil)
	from := start
	to := start.Add(3 * time.Hour)
	firstNow := start.Add(time.Hour)
	secondNow := start.Add(2 * time.Hour)

	first, err := history.BuildTimeReport(ctx, domain.TimeReportRequest{From: &from, To: &to, Now: &firstNow})
	if err != nil {
		t.Fatal(err)
	}
	second, err := history.BuildTimeReport(ctx, domain.TimeReportRequest{From: &from, To: &to, Now: &secondNow})
	if err != nil {
		t.Fatal(err)
	}
	if first.TotalActiveSeconds != 1800 || second.TotalActiveSeconds != 1800 {
		t.Fatalf("expected open paused session to stay at pre-pause active time, got first=%#v second=%#v", first, second)
	}
}

func TestListTaskSessionPageIncludesOverlapsAndPaginates(t *testing.T) {
	st, closeStore := newHistoryTestStore(t)
	defer closeStore()
	ctx := context.Background()
	start := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 25; i++ {
		sessionStart := start.Add(time.Duration(i-1) * time.Minute)
		if err := st.SaveTaskSession(ctx, reportSession(strconv.Itoa(i), "task", "Task", sessionStart, sessionStart.Add(30*time.Minute), 0)); err != nil {
			t.Fatal(err)
		}
	}
	history := NewHistoryService(st, nil, nil)
	from := start
	to := start.Add(20 * time.Minute)

	page, err := history.ListTaskSessionPage(ctx, domain.DetailedHistoryRequest{From: &from, To: &to, Page: 0})
	if err != nil {
		t.Fatal(err)
	}
	if page.PageSize != 20 || len(page.Sessions) != 20 || page.TotalCount != 21 || !page.HasNext || page.HasPrevious {
		t.Fatalf("unexpected first page: %#v", page)
	}
	next, err := history.ListTaskSessionPage(ctx, domain.DetailedHistoryRequest{From: &from, To: &to, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Sessions) != 1 || next.HasNext || !next.HasPrevious {
		t.Fatalf("unexpected second page: %#v", next)
	}
}

func newHistoryTestStore(t *testing.T) (*store.SQLiteStore, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	st := store.NewSQLiteStore(db)
	if err := st.Migrate(context.Background()); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	return st, func() { _ = db.Close() }
}

func reportSession(id string, taskID string, label string, started time.Time, ended time.Time, paused uint32) domain.TaskSession {
	duration := uint32(ended.Sub(started).Seconds())
	return domain.TaskSession{
		ID:                id,
		DeviceID:          "d1",
		TaskID:            taskID,
		TaskLabelSnapshot: label,
		TaskIconSnapshot:  "code",
		TaskColorSnapshot: "#69d2a5",
		Facet:             1,
		StartedAt:         started,
		EndedAt:           &ended,
		DurationSeconds:   duration,
		PausedSeconds:     paused,
		Source:            "test",
	}
}
