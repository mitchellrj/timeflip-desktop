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

func TestTrackingPauseSideClosesSessionAndReassignmentPreservesHistory(t *testing.T) {
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
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, nil)
	work := domain.FacetAssignment{
		ID: "a1", DeviceID: "d1", Facet: 1, TaskID: "task-1",
		TaskLabelSnapshot: "Coding", TaskIconSnapshot: "code", TaskColorSnapshot: "#2255AA",
		EffectiveFrom: time.Now().UTC(),
	}
	if err := st.SaveFacetAssignment(ctx, work); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveFacetAssignment(ctx, domain.FacetAssignment{ID: "pause", DeviceID: "d1", Facet: 2, IsPauseAssignment: true, EffectiveFrom: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	pauseAssignment, err := st.GetFacetAssignment(ctx, "d1", 2)
	if err != nil {
		t.Fatalf("get pause assignment: %v", err)
	}
	if !pauseAssignment.IsPauseAssignment {
		t.Fatalf("pause assignment was not persisted as pause: %#v", pauseAssignment)
	}
	start := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "facet", Facet: 1, OccurredAt: start, Source: "test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetOpenTaskSession(ctx, "d1"); err != nil {
		t.Fatalf("expected open session after work facet: %v", err)
	}
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "facet", Facet: 2, OccurredAt: start.Add(30 * time.Minute), Source: "test"}); err != nil {
		t.Fatal(err)
	}
	sessions, err := st.ListTaskSessions(ctx, domain.TaskSessionFilter{DeviceID: "d1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected one session, got %d", len(sessions))
	}
	if sessions[0].TaskLabelSnapshot != "Coding" || sessions[0].DurationSeconds != 1800 {
		t.Fatalf("unexpected session: %#v", sessions[0])
	}
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }
