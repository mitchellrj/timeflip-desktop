package domain

import (
	"testing"
	"time"
)

func TestValidateFacetAssignmentRejectsPauseTaskMix(t *testing.T) {
	err := ValidateFacetAssignment(FacetAssignment{
		DeviceID:          "device-1",
		Facet:             1,
		TaskID:            "task-1",
		IsPauseAssignment: true,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateFacetAssignmentRequiresExplicitPomodoroDuration(t *testing.T) {
	err := ValidateFacetAssignment(FacetAssignment{
		DeviceID:             "device-1",
		Facet:                1,
		TaskID:               "task-1",
		IsPomodoroAssignment: true,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateFacetAssignmentRejectsTaskDurationWithoutPomodoro(t *testing.T) {
	err := ValidateFacetAssignment(FacetAssignment{
		DeviceID:             "device-1",
		Facet:                1,
		TaskID:               "task-1",
		PomodoroLimitSeconds: 1500,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateDeviceNameRequiresPrintableASCIIWithinDeviceLimit(t *testing.T) {
	valid := "TimeFlip Desk"
	if err := ValidateDeviceName(valid); err != nil {
		t.Fatalf("expected %q to be valid: %v", valid, err)
	}
	for _, name := range []string{"", "1234567890123456789", "TimeFlip π", "TimeFlip\nDesk"} {
		if err := ValidateDeviceName(name); err == nil {
			t.Fatalf("expected %q to be invalid", name)
		}
	}
}

func TestStartTaskSessionSkipsPauseAssignment(t *testing.T) {
	session, created, err := StartTaskSession("device-1", FacetAssignment{
		DeviceID:          "device-1",
		Facet:             1,
		IsPauseAssignment: true,
	}, DeviceEventRecord{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Fatalf("expected no session, got %#v", session)
	}
}

func TestEndTaskSessionRejectsNegativeDuration(t *testing.T) {
	start := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	_, _, err := EndTaskSession(TaskSession{StartedAt: start}, start.Add(-time.Second), 1)
	if err == nil {
		t.Fatal("expected negative duration error")
	}
}

func TestEndTaskSessionAccumulatesOpenPause(t *testing.T) {
	start := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	pauseStarted := start.Add(30 * time.Minute)
	closed, meaningful, err := EndTaskSession(TaskSession{StartedAt: start, PauseStartedAt: &pauseStarted}, start.Add(45*time.Minute), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !meaningful {
		t.Fatal("expected meaningful session")
	}
	if closed.DurationSeconds != 2700 || closed.PausedSeconds != 900 || closed.PauseStartedAt != nil {
		t.Fatalf("unexpected closed session: %#v", closed)
	}
}
