package services

import (
	"context"
	"testing"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/device"
	"github.com/mitchellrj/timeflip-desktop/internal/domain"
)

func TestSetLockedPersistsDeviceState(t *testing.T) {
	ctx := context.Background()
	st := &trackingMemoryStore{
		state: domain.DeviceState{DeviceID: "d1", ConnectionState: domain.ConnectionConnected, CurrentFacet: 4, CurrentFacetKnown: true},
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &lockClient{}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)
	svc.handles["d1"] = pauseHandle("d1")
	if err := svc.SetLocked(ctx, "d1", true); err != nil {
		t.Fatal(err)
	}

	state, err := st.GetDeviceState(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if !state.Locked || state.Paused || state.ConnectionState != domain.ConnectionConnected || state.CurrentFacet != 4 {
		t.Fatalf("unexpected device state: %#v", state)
	}
	if len(client.locks) != 1 || !client.locks[0] {
		t.Fatalf("expected one lock command, got %#v", client.locks)
	}
	if !hasPublishedEvent(bus.Events, "device.state") {
		t.Fatalf("expected device.state event, got %#v", bus.Events)
	}
}

func TestLockDoesNotPauseAndIgnoresFacetChangesUntilUnlocked(t *testing.T) {
	ctx := context.Background()
	st := &trackingMemoryStore{}

	now := time.Date(2026, 5, 25, 10, 15, 0, 0, time.UTC)
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: now}, bus)
	client := &lockClient{}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)
	svc.handles["d1"] = pauseHandle("d1")

	start := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	if err := st.SaveFacetAssignment(ctx, domain.FacetAssignment{
		ID: "a1", DeviceID: "d1", Facet: 1, TaskID: "task-1",
		TaskLabelSnapshot: "Coding", TaskIconSnapshot: "code", TaskColorSnapshot: "#2255AA",
		EffectiveFrom: start,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveFacetAssignment(ctx, domain.FacetAssignment{
		ID: "a2", DeviceID: "d1", Facet: 2, TaskID: "task-2",
		TaskLabelSnapshot: "Review", TaskIconSnapshot: "search", TaskColorSnapshot: "#33AA55",
		EffectiveFrom: start,
	}); err != nil {
		t.Fatal(err)
	}
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "facet", Facet: 1, OccurredAt: start, Source: "test"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetLocked(ctx, "d1", true); err != nil {
		t.Fatal(err)
	}
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "facet", Facet: 2, OccurredAt: start.Add(10 * time.Minute), Source: "test"}); err != nil {
		t.Fatal(err)
	}

	state, err := st.GetDeviceState(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if !state.Locked || state.Paused || state.CurrentFacet != 1 {
		t.Fatalf("expected lock to keep tracking original facet without pausing, got %#v", state)
	}
	if len(client.locks) != 1 || !client.locks[0] {
		t.Fatalf("expected one lock command, got %#v", client.locks)
	}
	sessions, err := st.ListTaskSessions(ctx, domain.TaskSessionFilter{DeviceID: "d1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected locked facet change to keep one current session, got %d: %#v", len(sessions), sessions)
	}
	if sessions[0].TaskID != "task-1" || sessions[0].EndedAt != nil || sessions[0].PauseStartedAt != nil {
		t.Fatalf("expected original task to keep tracking while locked, got %#v", sessions[0])
	}
	if err := svc.SetLocked(ctx, "d1", false); err != nil {
		t.Fatal(err)
	}
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "facet", Facet: 2, OccurredAt: start.Add(20 * time.Minute), Source: "test"}); err != nil {
		t.Fatal(err)
	}
	state, err = st.GetDeviceState(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if state.Locked || state.Paused || state.CurrentFacet != 2 {
		t.Fatalf("expected unlocked facet event to switch tracking, got %#v", state)
	}
}

type lockClient struct {
	pauseClient
	locks []bool
}

func (c *lockClient) SetLock(_ context.Context, _ device.Handle, locked bool) error {
	c.locks = append(c.locks, locked)
	return nil
}
