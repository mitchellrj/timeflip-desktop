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
