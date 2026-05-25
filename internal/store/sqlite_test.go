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

	assignment := domain.FacetAssignment{
		ID:                   "assignment-1",
		DeviceID:             "d1",
		Facet:                1,
		TaskID:               "task-1",
		TaskLabelSnapshot:    "Coding",
		TaskIconSnapshot:     "code",
		TaskColorSnapshot:    "#2B6CB0",
		IsPomodoroAssignment: true,
		PomodoroLimitSeconds: 1500,
		ConfirmedOnDevice:    true,
	}
	if err := s.SaveFacetAssignment(ctx, assignment); err != nil {
		t.Fatalf("save facet assignment: %v", err)
	}
	loadedAssignment, err := s.GetFacetAssignment(ctx, "d1", 1)
	if err != nil {
		t.Fatalf("get facet assignment: %v", err)
	}
	if !loadedAssignment.IsPomodoroAssignment || loadedAssignment.PomodoroLimitSeconds != 1500 || !loadedAssignment.ConfirmedOnDevice {
		t.Fatalf("unexpected facet assignment: %#v", loadedAssignment)
	}

	settings := domain.DeviceTapSettings{DeviceID: "d1", Threshold: 20, Limit: 10, Latency: 5, Window: 30, ConfirmedOnDevice: true}
	if err := s.SaveDeviceTapSettings(ctx, settings); err != nil {
		t.Fatalf("save tap settings: %v", err)
	}
	loaded, err := s.GetDeviceTapSettings(ctx, "d1")
	if err != nil {
		t.Fatalf("get tap settings: %v", err)
	}
	if loaded.Threshold != 20 || loaded.Limit != 10 || loaded.Latency != 5 || loaded.Window != 30 || !loaded.ConfirmedOnDevice {
		t.Fatalf("unexpected tap settings: %#v", loaded)
	}

	led := domain.DeviceLEDSettings{DeviceID: "d1", BrightnessPercent: 55, BlinkSeconds: 12, ConfirmedOnDevice: true}
	if err := s.SaveDeviceLEDSettings(ctx, led); err != nil {
		t.Fatalf("save LED settings: %v", err)
	}
	loadedLED, err := s.GetDeviceLEDSettings(ctx, "d1")
	if err != nil {
		t.Fatalf("get LED settings: %v", err)
	}
	if loadedLED.BrightnessPercent != 55 || loadedLED.BlinkSeconds != 12 || !loadedLED.ConfirmedOnDevice {
		t.Fatalf("unexpected LED settings: %#v", loadedLED)
	}
}
