package services

import (
	"context"
	"database/sql"
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
	defer db.Close()
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
