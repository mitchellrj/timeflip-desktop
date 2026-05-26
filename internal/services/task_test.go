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

func TestTaskServiceRejectsDuplicateTaskLabels(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	tasks := NewTaskService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)})

	if _, err := tasks.CreateTask(ctx, "TimeFlip Desktop", "tag", "#69d2a5"); err != nil {
		t.Fatal(err)
	}
	if _, err := tasks.CreateTask(ctx, " timeflip   desktop ", "tag", "#69d2a5"); err == nil {
		t.Fatal("expected duplicate task label error")
	}
}

func TestAssignFacetReusesExistingTaskByLabel(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	tasks := NewTaskService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)})

	existing, err := tasks.CreateTask(ctx, "TimeFlip Desktop", "tag", "#69d2a5")
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := tasks.AssignFacet(ctx, domain.FacetConfigurationRequest{
		DeviceID: "d1",
		Facet:    1,
		Label:    "timeflip desktop",
		Icon:     "tag",
		Color:    "#69d2a5",
	})
	if err != nil {
		t.Fatal(err)
	}
	if assignment.TaskID != existing.ID {
		t.Fatalf("expected existing task to be reused, got %q want %q", assignment.TaskID, existing.ID)
	}
	allTasks, err := st.ListTasks(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(allTasks) != 1 {
		t.Fatalf("expected one task, got %#v", allTasks)
	}
}

func TestClearFacetConfigurationUnassignsOneFacet(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	tasks := NewTaskService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)})

	if _, err := tasks.AssignFacet(ctx, domain.FacetConfigurationRequest{
		DeviceID: "d1",
		Facet:    1,
		Label:    "Coding",
		Icon:     "code-xml",
		Color:    "#153f8a",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tasks.AssignFacet(ctx, domain.FacetConfigurationRequest{
		DeviceID:          "d1",
		Facet:             2,
		Label:             "Break",
		Icon:              "coffee",
		Color:             "#69d2a5",
		IsPauseAssignment: true,
	}); err != nil {
		t.Fatal(err)
	}

	cleared, err := tasks.ClearFacetConfiguration(ctx, "d1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Facet != 1 || cleared.Label != "" || cleared.TaskID != "" || cleared.IsPauseAssignment {
		t.Fatalf("expected blank facet view after clear, got %#v", cleared)
	}
	views, err := tasks.ListFacetConfiguration(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if views[0].Label != "" || views[0].TaskID != "" {
		t.Fatalf("expected facet 1 to be unassigned, got %#v", views[0])
	}
	if !views[1].IsPauseAssignment || views[1].Label != "Paused" {
		t.Fatalf("expected facet 2 to remain assigned, got %#v", views[1])
	}
}

func newTestStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	st := store.NewSQLiteStore(db)
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return st
}
