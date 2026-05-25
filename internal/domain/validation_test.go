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
