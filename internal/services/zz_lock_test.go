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
		state: domain.DeviceState{DeviceID: "d1", ConnectionState: domain.ConnectionConnected, CurrentFacet: 4, CurrentFacetKnown: true, Paused: true},
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
	if !state.Locked || !state.Paused || state.ConnectionState != domain.ConnectionConnected || state.CurrentFacet != 4 {
		t.Fatalf("unexpected device state: %#v", state)
	}
	if len(client.locks) != 1 || !client.locks[0] {
		t.Fatalf("expected one lock command, got %#v", client.locks)
	}
	if !hasPublishedEvent(bus.Events, "device.state") {
		t.Fatalf("expected device.state event, got %#v", bus.Events)
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
