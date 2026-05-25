package store

import (
	"context"
	"database/sql"
	"testing"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	_ "modernc.org/sqlite"
)

func TestSQLiteStoreMigrateTwiceAndPersistTask(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := NewSQLiteStore(db)
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	task := domain.Task{ID: "task-1", Label: "Coding", Icon: "code", Color: "#2B6CB0"}
	if err := s.SaveTask(ctx, task); err != nil {
		t.Fatalf("save task: %v", err)
	}
	tasks, err := s.ListTasks(ctx, false)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Label != "Coding" {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}
}
