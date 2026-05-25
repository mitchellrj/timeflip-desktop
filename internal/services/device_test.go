package services

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/device"
	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/store"
	_ "modernc.org/sqlite"
)

func TestSetPausedPersistsDeviceState(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveDeviceState(ctx, domain.DeviceState{DeviceID: "d1", ConnectionState: domain.ConnectionConnected, CurrentFacet: 4, CurrentFacetKnown: true}); err != nil {
		t.Fatal(err)
	}

	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	svc := NewDeviceService(&pauseClient{}, st, nil, tracking, nil, bus, tracking.clock)
	if err := svc.SetPaused(ctx, "d1", true); err != nil {
		t.Fatal(err)
	}

	state, err := st.GetDeviceState(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if !state.Paused || state.ConnectionState != domain.ConnectionConnected || state.CurrentFacet != 4 {
		t.Fatalf("unexpected device state: %#v", state)
	}
	if !hasPublishedEvent(bus.Events, "device.state") {
		t.Fatalf("expected device.state event, got %#v", bus.Events)
	}
}

func TestConfigureTapSettingsPersistsAndWritesDevice(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}

	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &pauseClient{}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)
	if _, err := svc.ensureHandle(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	saved, err := svc.ConfigureTapSettings(ctx, domain.DeviceTapSettings{DeviceID: "d1", Threshold: 21, Limit: 9, Latency: 4, Window: 31})
	if err != nil {
		t.Fatal(err)
	}

	if !saved.ConfirmedOnDevice {
		t.Fatalf("expected settings confirmed on device: %#v", saved)
	}
	if len(client.tapSettings) != 1 {
		t.Fatalf("expected one tap settings write, got %#v", client.tapSettings)
	}
	written := client.tapSettings[0]
	if written.Threshold != 21 || written.Limit != 9 || written.Latency != 4 || written.Window != 31 {
		t.Fatalf("unexpected tap settings write: %#v", written)
	}
	loaded, err := st.GetDeviceTapSettings(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.ConfirmedOnDevice || loaded.Threshold != 21 || loaded.Window != 31 {
		t.Fatalf("unexpected stored tap settings: %#v", loaded)
	}
}

func TestBeginTapTuningUsesStoredSettingsOrDefaults(t *testing.T) {
	ctx := context.Background()
	st := &trackingMemoryStore{}
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	stored := domain.DeviceTapSettings{DeviceID: "d1", Threshold: 21, Limit: 9, Latency: 4, Window: 31, ConfirmedOnDevice: true}
	if err := st.SaveDeviceTapSettings(ctx, stored); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	svc := NewDeviceService(&noStreamTapClient{}, st, nil, tracking, nil, bus, tracking.clock)

	state, err := svc.BeginTapTuning(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if !state.Active || state.DraftSettings.Threshold != 21 || state.OriginalSettings.Window != 31 {
		t.Fatalf("expected stored settings in tuning state, got %#v", state)
	}
	if !hasPublishedEvent(bus.Events, "device.tap.tuning.state") {
		t.Fatalf("expected tuning state event, got %#v", bus.Events)
	}

	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d2", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	state, err = svc.BeginTapTuning(ctx, "d2")
	if err != nil {
		t.Fatal(err)
	}
	if state.DraftSettings.Threshold != 20 || state.DraftSettings.Limit != 10 || state.DraftSettings.Window != 30 {
		t.Fatalf("expected default settings for unsaved device, got %#v", state)
	}
}

func TestPreviewTapTuningWritesDeviceWithoutPersisting(t *testing.T) {
	ctx := context.Background()
	st := &trackingMemoryStore{}
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	original := domain.DeviceTapSettings{DeviceID: "d1", Threshold: 20, Limit: 10, Latency: 5, Window: 30, ConfirmedOnDevice: true}
	if err := st.SaveDeviceTapSettings(ctx, original); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &noStreamTapClient{}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)
	if _, err := svc.BeginTapTuning(ctx, "d1"); err != nil {
		t.Fatal(err)
	}

	state, err := svc.PreviewTapTuningSettings(ctx, domain.DeviceTapSettings{DeviceID: "d1", Threshold: 24, Limit: 12, Latency: 6, Window: 36})
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "temporary" || !state.AppliedSettings.ConfirmedOnDevice {
		t.Fatalf("expected temporary applied state, got %#v", state)
	}
	if len(client.tapSettings) != 1 || client.tapSettings[0].Threshold != 24 || client.tapSettings[0].Window != 36 {
		t.Fatalf("expected one temporary device write, got %#v", client.tapSettings)
	}
	loaded, err := st.GetDeviceTapSettings(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Threshold != original.Threshold || loaded.Window != original.Window {
		t.Fatalf("preview persisted settings unexpectedly: %#v", loaded)
	}
}

func TestConfirmTapTuningPersistsAndEndsSession(t *testing.T) {
	ctx := context.Background()
	st := &trackingMemoryStore{}
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &noStreamTapClient{}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)
	if _, err := svc.BeginTapTuning(ctx, "d1"); err != nil {
		t.Fatal(err)
	}

	saved, err := svc.ConfirmTapTuningSettings(ctx, domain.DeviceTapSettings{DeviceID: "d1", Threshold: 23, Limit: 11, Latency: 6, Window: 35})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.ConfirmedOnDevice || saved.Threshold != 23 {
		t.Fatalf("expected confirmed saved settings, got %#v", saved)
	}
	if states := svc.TapTuningStates(); len(states) != 0 {
		t.Fatalf("expected session to end, got %#v", states)
	}
	loaded, err := st.GetDeviceTapSettings(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.ConfirmedOnDevice || loaded.Window != 35 {
		t.Fatalf("unexpected stored settings: %#v", loaded)
	}
}

func TestCancelTapTuningRestoresOriginalWithoutChangingStore(t *testing.T) {
	ctx := context.Background()
	st := &trackingMemoryStore{}
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	original := domain.DeviceTapSettings{DeviceID: "d1", Threshold: 20, Limit: 10, Latency: 5, Window: 30, ConfirmedOnDevice: true}
	if err := st.SaveDeviceTapSettings(ctx, original); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &noStreamTapClient{}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)
	if _, err := svc.BeginTapTuning(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PreviewTapTuningSettings(ctx, domain.DeviceTapSettings{DeviceID: "d1", Threshold: 24, Limit: 12, Latency: 6, Window: 36}); err != nil {
		t.Fatal(err)
	}
	state, err := svc.CancelTapTuning(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if state.Active || state.Status != "cancelled" {
		t.Fatalf("expected inactive cancelled state, got %#v", state)
	}
	if len(client.tapSettings) != 2 || client.tapSettings[1].Threshold != original.Threshold || client.tapSettings[1].Window != original.Window {
		t.Fatalf("expected preview then restore writes, got %#v", client.tapSettings)
	}
	loaded, err := st.GetDeviceTapSettings(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Threshold != original.Threshold || loaded.Window != original.Window {
		t.Fatalf("cancel changed stored settings: %#v", loaded)
	}
}

func TestDoubleTapEventPublishesTuningObservation(t *testing.T) {
	ctx := context.Background()
	st := &trackingMemoryStore{}
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	svc := NewDeviceService(&noStreamTapClient{}, st, nil, tracking, nil, bus, tracking.clock)
	if _, err := svc.BeginTapTuning(ctx, "d1"); err != nil {
		t.Fatal(err)
	}

	svc.publishTapTuningObservation(ctx, domain.DeviceEventRecord{
		DeviceID:   "d1",
		Kind:       "double_tap",
		Facet:      4,
		OccurredAt: time.Date(2026, 5, 25, 10, 1, 0, 0, time.UTC),
		Source:     "test",
	})

	states := svc.TapTuningStates()
	if len(states) != 1 || states[0].DetectedCount != 1 || states[0].LastObservation == nil || states[0].LastObservation.Facet != 4 {
		t.Fatalf("expected detected tap in tuning state, got %#v", states)
	}
	if !hasPublishedEvent(bus.Events, "device.tap.tuning.detected") {
		t.Fatalf("expected tuning detected event, got %#v", bus.Events)
	}
}

func TestPreviewTapTuningRequiresActiveSession(t *testing.T) {
	ctx := context.Background()
	st := &trackingMemoryStore{}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	svc := NewDeviceService(&pauseClient{}, st, nil, tracking, nil, bus, tracking.clock)

	if _, err := svc.PreviewTapTuningSettings(ctx, domain.DeviceTapSettings{DeviceID: "d1", Threshold: 24, Limit: 12, Latency: 6, Window: 36}); err == nil {
		t.Fatal("expected active session validation error")
	}
}

func TestConfigureLEDSettingsPersistsAndWritesDevice(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}

	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &pauseClient{}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)
	if _, err := svc.ensureHandle(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	saved, err := svc.ConfigureLEDSettings(ctx, domain.DeviceLEDSettings{DeviceID: "d1", BrightnessPercent: 60, BlinkSeconds: 15})
	if err != nil {
		t.Fatal(err)
	}

	if !saved.ConfirmedOnDevice {
		t.Fatalf("expected settings confirmed on device: %#v", saved)
	}
	if len(client.ledSettings) != 1 {
		t.Fatalf("expected one LED settings write, got %#v", client.ledSettings)
	}
	written := client.ledSettings[0]
	if written.BrightnessPercent != 60 || written.BlinkSeconds != 15 {
		t.Fatalf("unexpected LED settings write: %#v", written)
	}
	loaded, err := st.GetDeviceLEDSettings(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.ConfirmedOnDevice || loaded.BrightnessPercent != 60 || loaded.BlinkSeconds != 15 {
		t.Fatalf("unexpected stored LED settings: %#v", loaded)
	}
}

func TestConfigureDeviceNamePersistsAndWritesDevice(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000", DisplayName: "Old", AdvertisedName: "Old"}); err != nil {
		t.Fatal(err)
	}

	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &pauseClient{}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)
	if _, err := svc.ensureHandle(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	saved, err := svc.ConfigureDeviceName(ctx, "d1", "Desk Flip")
	if err != nil {
		t.Fatal(err)
	}

	if saved.DisplayName != "Desk Flip" || saved.AdvertisedName != "Desk Flip" {
		t.Fatalf("unexpected saved profile: %#v", saved)
	}
	if len(client.names) != 1 || client.names[0] != "Desk Flip" {
		t.Fatalf("expected one device name write, got %#v", client.names)
	}
	loaded, err := st.GetDeviceProfile(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DisplayName != "Desk Flip" || loaded.AdvertisedName != "Desk Flip" || loaded.StoredPassword != "000000" {
		t.Fatalf("unexpected stored profile: %#v", loaded)
	}
	if !hasPublishedEvent(bus.Events, "device.profile.saved") {
		t.Fatalf("expected profile saved event, got %#v", bus.Events)
	}
}

func TestConfigureDeviceNameSavesPendingNameWhenDisconnected(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000", DisplayName: "Old", AdvertisedName: "Old"}); err != nil {
		t.Fatal(err)
	}

	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &pauseClient{}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)
	saved, err := svc.ConfigureDeviceName(ctx, "d1", "Later Flip")
	if err != nil {
		t.Fatal(err)
	}

	if saved.DisplayName != "Later Flip" || saved.AdvertisedName != "Old" {
		t.Fatalf("unexpected pending profile: %#v", saved)
	}
	if len(client.names) != 0 {
		t.Fatalf("disconnected device should not be written immediately: %#v", client.names)
	}
	if _, err := svc.ensureHandle(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	if len(client.names) != 1 || client.names[0] != "Later Flip" {
		t.Fatalf("expected pending name write on connect, got %#v", client.names)
	}
	loaded, err := st.GetDeviceProfile(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AdvertisedName != "Later Flip" {
		t.Fatalf("expected advertised name confirmed after connect: %#v", loaded)
	}
}

func TestConfigureFacetWritesDeviceColour(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}

	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &countingClient{facetView: domain.FacetConfigurationView{AssignedOnDevice: true}}
	tasks := NewTaskService(st, tracking.clock)
	svc := NewDeviceService(client, st, tasks, tracking, nil, bus, tracking.clock)
	if _, err := svc.ensureHandle(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	saved, err := svc.ConfigureFacet(ctx, domain.FacetConfigurationRequest{
		DeviceID: "d1",
		Facet:    3,
		TaskID:   "task-1",
		Label:    "Coding",
		Icon:     "code",
		Color:    "#2255AA",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !saved.AssignedOnDevice {
		t.Fatalf("expected facet confirmed on device: %#v", saved)
	}
	if len(client.facetAssignments) != 1 {
		t.Fatalf("expected one facet write, got %#v", client.facetAssignments)
	}
	written := client.facetAssignments[0]
	if written.Facet != 3 || written.TaskColorSnapshot != "#2255AA" {
		t.Fatalf("unexpected facet write: %#v", written)
	}
	loaded, err := st.GetFacetAssignment(ctx, "d1", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.ConfirmedOnDevice {
		t.Fatalf("expected stored facet to be confirmed: %#v", loaded)
	}
}

func TestConfigureFacetWritesExplicitPomodoroMode(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}

	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &countingClient{facetView: domain.FacetConfigurationView{AssignedOnDevice: true}}
	tasks := NewTaskService(st, tracking.clock)
	svc := NewDeviceService(client, st, tasks, tracking, nil, bus, tracking.clock)
	if _, err := svc.ensureHandle(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	saved, err := svc.ConfigureFacet(ctx, domain.FacetConfigurationRequest{
		DeviceID:             "d1",
		Facet:                5,
		TaskID:               "task-1",
		Label:                "Focus",
		Icon:                 "timer",
		Color:                "#2255AA",
		IsPomodoroAssignment: true,
		PomodoroLimitSeconds: 1500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.IsPomodoroAssignment || saved.PomodoroLimitSeconds != 1500 {
		t.Fatalf("expected pomodoro facet view: %#v", saved)
	}
	if len(client.facetAssignments) != 1 {
		t.Fatalf("expected one facet write, got %#v", client.facetAssignments)
	}
	written := client.facetAssignments[0]
	if !written.IsPomodoroAssignment || written.PomodoroLimitSeconds != 1500 {
		t.Fatalf("unexpected facet write: %#v", written)
	}
	loaded, err := st.GetFacetAssignment(ctx, "d1", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.IsPomodoroAssignment || loaded.PomodoroLimitSeconds != 1500 {
		t.Fatalf("unexpected stored facet: %#v", loaded)
	}
}

func TestConnectDeviceAppliesUnconfirmedFacetAssignments(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveFacetAssignment(ctx, domain.FacetAssignment{
		ID:                "assignment-1",
		DeviceID:          "d1",
		Facet:             4,
		TaskID:            "task-1",
		TaskLabelSnapshot: "Review",
		TaskIconSnapshot:  "search",
		TaskColorSnapshot: "#33AA55",
		EffectiveFrom:     time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &countingClient{facetView: domain.FacetConfigurationView{AssignedOnDevice: true}}
	history := NewHistoryService(st, client, tracking)
	svc := NewDeviceService(client, st, nil, tracking, history, bus, tracking.clock)
	if err := svc.ConnectDevice(ctx, "d1"); err != nil {
		t.Fatal(err)
	}

	if len(client.facetAssignments) != 1 {
		t.Fatalf("expected stored facet write on connect, got %#v", client.facetAssignments)
	}
	written := client.facetAssignments[0]
	if written.Facet != 4 || written.TaskColorSnapshot != "#33AA55" {
		t.Fatalf("unexpected facet write: %#v", written)
	}
	loaded, err := st.GetFacetAssignment(ctx, "d1", 4)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.ConfirmedOnDevice {
		t.Fatalf("expected stored facet to be confirmed: %#v", loaded)
	}
}

func TestShouldIgnoreTextFacetEvents(t *testing.T) {
	event := domain.DeviceEventRecord{Kind: "facet", Facet: 3, RawSummary: "F1196F51-71A4-11E6-BDF4-0800200C9A66"}
	if !shouldIgnoreDeviceEvent(event) {
		t.Fatalf("expected promoted text facet event to be ignored")
	}
	event.RawSummary = "F1196F52-71A4-11E6-BDF4-0800200C9A66"
	if shouldIgnoreDeviceEvent(event) {
		t.Fatalf("expected authoritative facet characteristic to be accepted")
	}
}

func TestRefreshDeviceStatePublishesReconnectBeforeCloseReturns(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	client := &blockingCloseClient{snapshotErr: errors.New("snapshot timeout"), release: make(chan struct{}), done: make(chan struct{})}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)

	start := time.Now()
	if _, err := svc.RefreshDeviceState(ctx, "d1"); err == nil {
		t.Fatal("expected snapshot error")
	}
	if time.Since(start) > time.Second {
		t.Fatalf("refresh waited for close to return")
	}
	if !hasConnectionState(bus.Events, domain.ConnectionReconnecting) {
		t.Fatalf("expected reconnecting state before close completed, got %#v", bus.Events)
	}
	close(client.release)
	select {
	case <-client.done:
	case <-time.After(time.Second):
		t.Fatalf("async close did not finish")
	}
}

func TestConnectDeviceIsNoopWhenHandleAlreadyActive(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &countingClient{}
	history := NewHistoryService(st, client, tracking)
	svc := NewDeviceService(client, st, nil, tracking, history, bus, tracking.clock)

	if err := svc.ConnectDevice(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConnectDevice(ctx, "d1"); err != nil {
		t.Fatal(err)
	}

	if got := client.count("connect"); got != 1 {
		t.Fatalf("expected one physical connect, got %d", got)
	}
	if got := client.count("history"); got != 1 {
		t.Fatalf("expected one startup history import, got %d", got)
	}
}

func TestConnectDeviceSerializesConcurrentAttempts(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &countingClient{connectStarted: make(chan struct{}), releaseConnect: make(chan struct{})}
	history := NewHistoryService(st, client, tracking)
	svc := NewDeviceService(client, st, nil, tracking, history, bus, tracking.clock)

	errs := make(chan error, 2)
	go func() { errs <- svc.ConnectDevice(ctx, "d1") }()
	<-client.connectStarted
	go func() { errs <- svc.ConnectDevice(ctx, "d1") }()
	close(client.releaseConnect)

	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if got := client.count("connect"); got != 1 {
		t.Fatalf("expected one physical connect, got %d", got)
	}
	if got := client.count("history"); got != 1 {
		t.Fatalf("expected one startup history import, got %d", got)
	}
}

func TestConnectDeviceDoesNotPublishDeviceErrorWhenHistoryImportFails(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000"}); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &countingClient{historyErr: errors.New("history protocol mismatch")}
	history := NewHistoryService(st, client, tracking)
	svc := NewDeviceService(client, st, nil, tracking, history, bus, tracking.clock)

	if err := svc.ConnectDevice(ctx, "d1"); err != nil {
		t.Fatal(err)
	}

	if hasPublishedEvent(bus.Events, "device.error") {
		t.Fatalf("expected history import failure to stay out of device.error, got %#v", bus.Events)
	}
	if !hasPublishedEvent(bus.Events, "history.import_failed") {
		t.Fatalf("expected history.import_failed diagnostic event, got %#v", bus.Events)
	}
	if got := client.count("snapshot"); got != 1 {
		t.Fatalf("expected connect to continue through snapshot, got %d snapshots", got)
	}
}

func TestConnectDeviceUsesAutoProtocolForStoredV4Profile(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000", ProtocolVersion: "v4"}); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &countingClient{}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)

	if err := svc.ConnectDevice(ctx, "d1"); err != nil {
		t.Fatal(err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.connectRequests) != 1 {
		t.Fatalf("expected one connect request, got %#v", client.connectRequests)
	}
	if client.connectRequests[0].ProtocolVersion != "" {
		t.Fatalf("expected auto protocol for stored v4 profile, got %q", client.connectRequests[0].ProtocolVersion)
	}
}

func TestUnpairDeviceResetsCurrentStateAndClosesSession(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "000000", PairingState: "paired"}); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 5, 25, 9, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	if err := st.SaveDeviceState(ctx, domain.DeviceState{
		DeviceID:          "d1",
		ConnectionState:   domain.ConnectionConnected,
		CurrentFacet:      4,
		CurrentFacetKnown: true,
		Paused:            true,
		Locked:            true,
		BatteryPercent:    87,
		UpdatedAt:         start,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveTaskSession(ctx, domain.TaskSession{
		ID:                "session-1",
		DeviceID:          "d1",
		TaskID:            "task-1",
		TaskLabelSnapshot: "Coding",
		Facet:             4,
		StartedAt:         start,
		Source:            "test",
	}); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: now}, bus)
	client := &countingClient{unpairResult: domain.UnpairingWorkflow{DeviceID: "d1", Completed: true}}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)
	svc.handles["d1"] = pauseHandle("d1")

	result, err := svc.UnpairDevice(ctx, "d1", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Completed {
		t.Fatalf("expected completed unpair result: %#v", result)
	}

	state, err := st.GetDeviceState(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if state.ConnectionState != domain.ConnectionDisconnected || state.CurrentFacetKnown || state.CurrentFacet != 0 || state.Paused || state.Locked || state.BatteryPercent != 0 {
		t.Fatalf("expected reset device state, got %#v", state)
	}
	profile, err := st.GetDeviceProfile(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if profile.PairingState != "unpaired" {
		t.Fatalf("expected unpaired profile, got %#v", profile)
	}
	if _, err := st.GetOpenTaskSession(ctx, "d1"); err == nil {
		t.Fatal("expected active session to be closed")
	}
	if !hasPublishedEvent(bus.Events, "device.state") || !hasPublishedEvent(bus.Events, "device.connection") || !hasPublishedEvent(bus.Events, "tracking.session.ended") {
		t.Fatalf("expected reset events to be published, got %#v", bus.Events)
	}
	if _, ok := svc.currentHandle("d1"); ok {
		t.Fatal("expected active handle to be removed")
	}
}

func TestPairDeviceResetsClosedSessionState(t *testing.T) {
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
	if err := st.SaveDeviceProfile(ctx, domain.DeviceProfile{ID: "d1", StoredPassword: "123456", PairingState: "paired", DisplayName: "TimeFlip"}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveDeviceState(ctx, domain.DeviceState{
		DeviceID:          "d1",
		ConnectionState:   domain.ConnectionConnected,
		CurrentFacet:      4,
		CurrentFacetKnown: true,
		Paused:            true,
		Locked:            true,
		BatteryPercent:    87,
		UpdatedAt:         time.Date(2026, 5, 25, 9, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	client := &countingClient{pairResult: domain.PairingWorkflow{DeviceID: "d1", Completed: true, CurrentStage: "complete"}}
	svc := NewDeviceService(client, st, nil, tracking, nil, bus, tracking.clock)
	svc.handles["d1"] = pauseHandle("d1")

	result, err := svc.PairDevice(ctx, "d1", "123456", "654321", true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Completed {
		t.Fatalf("expected completed pair result: %#v", result)
	}
	if _, ok := svc.currentHandle("d1"); ok {
		t.Fatal("expected stale active handle to be removed")
	}
	if got := client.count("close"); got != 1 {
		t.Fatalf("expected stale handle close, got %d", got)
	}
	state, err := st.GetDeviceState(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if state.ConnectionState != domain.ConnectionDisconnected || state.CurrentFacetKnown || state.CurrentFacet != 0 || state.Paused || state.Locked || state.BatteryPercent != 0 {
		t.Fatalf("expected disconnected reset state after closed pair session, got %#v", state)
	}
	profile, err := st.GetDeviceProfile(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if profile.PairingState != "paired" || profile.StoredPassword != "654321" || profile.DisplayName != "TimeFlip" {
		t.Fatalf("unexpected profile after pair: %#v", profile)
	}
	if !hasPublishedEvent(bus.Events, "device.state") || !hasPublishedEvent(bus.Events, "device.connection") || !hasPublishedEvent(bus.Events, "device.pairing") {
		t.Fatalf("expected reset and pairing events, got %#v", bus.Events)
	}
}

func newDeviceTestStore(t *testing.T, ctx context.Context) (*store.SQLiteStore, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	st := store.NewSQLiteStore(db)
	if err := st.Migrate(ctx); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	return st, func() {}
}

func hasPublishedEvent(events []PublishedEvent, name string) bool {
	for _, event := range events {
		if event.Name == name {
			return true
		}
	}
	return false
}

func hasConnectionState(events []PublishedEvent, state domain.ConnectionState) bool {
	for _, event := range events {
		deviceState, ok := event.Payload.(domain.DeviceState)
		if ok && event.Name == "device.connection" && deviceState.ConnectionState == state {
			return true
		}
	}
	return false
}

type pauseClient struct {
	tapSettings []device.TapSettings
	ledSettings []device.LEDSettings
	names       []string
}

type countingClient struct {
	mu               sync.Mutex
	counts           map[string]int
	connectRequests  []device.ConnectRequest
	deviceNames      []string
	historyErr       error
	unpairResult     domain.UnpairingWorkflow
	pairResult       domain.PairingWorkflow
	facetView        domain.FacetConfigurationView
	facetAssignments []domain.FacetAssignment
	connectStarted   chan struct{}
	releaseConnect   chan struct{}
	connectOnce      sync.Once
}

func (c *countingClient) bump(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.counts == nil {
		c.counts = map[string]int{}
	}
	c.counts[name]++
}

func (c *countingClient) count(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[name]
}

func (c *countingClient) Scan(context.Context, bool) ([]domain.DiscoveredDevice, error) {
	return nil, nil
}

func (c *countingClient) Pair(context.Context, device.PairRequest) (domain.PairingWorkflow, error) {
	if c.pairResult.DeviceID != "" || c.pairResult.Completed {
		return c.pairResult, nil
	}
	return domain.PairingWorkflow{}, nil
}

func (c *countingClient) Unpair(context.Context, device.UnpairRequest) (domain.UnpairingWorkflow, error) {
	if c.unpairResult.DeviceID != "" || c.unpairResult.Completed {
		return c.unpairResult, nil
	}
	return domain.UnpairingWorkflow{}, nil
}

func (c *countingClient) Connect(_ context.Context, req device.ConnectRequest) (device.Handle, error) {
	c.bump("connect")
	c.mu.Lock()
	c.connectRequests = append(c.connectRequests, req)
	c.mu.Unlock()
	if c.connectStarted != nil {
		c.connectOnce.Do(func() {
			close(c.connectStarted)
		})
	}
	if c.releaseConnect != nil {
		<-c.releaseConnect
	}
	return pauseHandle(req.DeviceID), nil
}

func (c *countingClient) Authorize(context.Context, device.Handle, string) error {
	return nil
}

func (c *countingClient) ReadDeviceSnapshot(context.Context, device.Handle) (domain.DeviceSnapshot, error) {
	c.bump("snapshot")
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	return domain.DeviceSnapshot{
		Profile: domain.DeviceProfile{ID: "d1", DisplayName: "d1", LastSeenAt: now},
		State: domain.DeviceState{
			DeviceID:        "d1",
			ConnectionState: domain.ConnectionConnected,
			UpdatedAt:       now,
		},
	}, nil
}

func (c *countingClient) WriteFacetConfiguration(_ context.Context, _ device.Handle, assignment domain.FacetAssignment) (domain.FacetConfigurationView, error) {
	c.mu.Lock()
	c.facetAssignments = append(c.facetAssignments, assignment)
	c.mu.Unlock()
	return c.facetView, nil
}

func (c *countingClient) SetPause(context.Context, device.Handle, bool) error {
	return nil
}

func (c *countingClient) SetLock(context.Context, device.Handle, bool) error {
	return nil
}

func (c *countingClient) SetAutoPause(context.Context, device.Handle, uint16) error {
	return nil
}

func (c *countingClient) SetTapSettings(context.Context, device.Handle, device.TapSettings) error {
	return nil
}

func (c *countingClient) SetLEDSettings(context.Context, device.Handle, device.LEDSettings) error {
	return nil
}

func (c *countingClient) SetDeviceName(_ context.Context, _ device.Handle, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deviceNames = append(c.deviceNames, name)
	return nil
}

func (c *countingClient) ReadHistory(context.Context, device.Handle, device.HistoryRequest) ([]domain.DeviceEventRecord, error) {
	c.bump("history")
	return nil, c.historyErr
}

func (c *countingClient) Events(context.Context, device.Handle) (<-chan domain.DeviceEventRecord, <-chan error, error) {
	c.bump("events")
	return nil, nil, errors.New("stream unavailable")
}

func (c *countingClient) Close(context.Context, device.Handle) error {
	c.bump("close")
	return nil
}

type blockingCloseClient struct {
	pauseClient
	snapshotErr error
	release     chan struct{}
	done        chan struct{}
}

func (c *blockingCloseClient) ReadDeviceSnapshot(context.Context, device.Handle) (domain.DeviceSnapshot, error) {
	return domain.DeviceSnapshot{}, c.snapshotErr
}

func (c *blockingCloseClient) Close(ctx context.Context, _ device.Handle) error {
	defer close(c.done)
	select {
	case <-c.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type noStreamTapClient struct {
	pauseClient
}

func (c *noStreamTapClient) Events(context.Context, device.Handle) (<-chan domain.DeviceEventRecord, <-chan error, error) {
	return nil, nil, errors.New("stream unavailable")
}

type pauseHandle string

func (h pauseHandle) DeviceID() string { return string(h) }

func (pauseClient) Scan(context.Context, bool) ([]domain.DiscoveredDevice, error) {
	return nil, nil
}

func (pauseClient) Pair(context.Context, device.PairRequest) (domain.PairingWorkflow, error) {
	return domain.PairingWorkflow{}, nil
}

func (pauseClient) Unpair(context.Context, device.UnpairRequest) (domain.UnpairingWorkflow, error) {
	return domain.UnpairingWorkflow{}, nil
}

func (pauseClient) Connect(_ context.Context, req device.ConnectRequest) (device.Handle, error) {
	return pauseHandle(req.DeviceID), nil
}

func (pauseClient) Authorize(context.Context, device.Handle, string) error {
	return nil
}

func (pauseClient) ReadDeviceSnapshot(context.Context, device.Handle) (domain.DeviceSnapshot, error) {
	return domain.DeviceSnapshot{}, nil
}

func (pauseClient) WriteFacetConfiguration(context.Context, device.Handle, domain.FacetAssignment) (domain.FacetConfigurationView, error) {
	return domain.FacetConfigurationView{}, nil
}

func (pauseClient) SetPause(context.Context, device.Handle, bool) error {
	return nil
}

func (pauseClient) SetLock(context.Context, device.Handle, bool) error {
	return nil
}

func (pauseClient) SetAutoPause(context.Context, device.Handle, uint16) error {
	return nil
}

func (c *pauseClient) SetTapSettings(_ context.Context, _ device.Handle, settings device.TapSettings) error {
	c.tapSettings = append(c.tapSettings, settings)
	return nil
}

func (c *pauseClient) SetLEDSettings(_ context.Context, _ device.Handle, settings device.LEDSettings) error {
	c.ledSettings = append(c.ledSettings, settings)
	return nil
}

func (c *pauseClient) SetDeviceName(_ context.Context, _ device.Handle, name string) error {
	c.names = append(c.names, name)
	return nil
}

func (pauseClient) ReadHistory(context.Context, device.Handle, device.HistoryRequest) ([]domain.DeviceEventRecord, error) {
	return nil, nil
}

func (pauseClient) Events(context.Context, device.Handle) (<-chan domain.DeviceEventRecord, <-chan error, error) {
	events := make(chan domain.DeviceEventRecord)
	errs := make(chan error)
	return events, errs, nil
}

func (pauseClient) Close(context.Context, device.Handle) error {
	return nil
}
