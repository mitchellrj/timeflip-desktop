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

func TestTrackingPauseSideAccumulatesPausedTimeAndReassignmentPreservesHistory(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
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
	state, err := st.GetDeviceState(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if !state.Paused {
		t.Fatalf("expected pause side to mark device state paused: %#v", state)
	}
	open, err := st.GetOpenTaskSession(ctx, "d1")
	if err != nil {
		t.Fatalf("expected paused open session: %v", err)
	}
	if open.PauseStartedAt == nil {
		t.Fatalf("expected pause start on open session: %#v", open)
	}
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "facet", Facet: 1, OccurredAt: start.Add(45 * time.Minute), Source: "test"}); err != nil {
		t.Fatal(err)
	}
	state, err = st.GetDeviceState(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if state.Paused {
		t.Fatalf("expected work side to mark device state unpaused: %#v", state)
	}
	open, err = st.GetOpenTaskSession(ctx, "d1")
	if err != nil {
		t.Fatalf("expected resumed open session: %v", err)
	}
	if open.PauseStartedAt != nil || open.PausedSeconds != 900 {
		t.Fatalf("unexpected resumed session: %#v", open)
	}
	if err := st.SaveFacetAssignment(ctx, domain.FacetAssignment{
		ID: "a3", DeviceID: "d1", Facet: 3, TaskID: "task-2",
		TaskLabelSnapshot: "Review", TaskIconSnapshot: "search", TaskColorSnapshot: "#33AA55",
		EffectiveFrom: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "facet", Facet: 3, OccurredAt: start.Add(60 * time.Minute), Source: "test"}); err != nil {
		t.Fatal(err)
	}
	sessions, err := st.ListTaskSessions(ctx, domain.TaskSessionFilter{DeviceID: "d1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected one session, got %d", len(sessions))
	}
	var coding domain.TaskSession
	for _, session := range sessions {
		if session.TaskLabelSnapshot == "Coding" {
			coding = session
		}
	}
	if coding.ID == "" || coding.DurationSeconds != 3600 || coding.PausedSeconds != 900 || coding.EndedAt == nil {
		t.Fatalf("unexpected coding session: %#v", coding)
	}
}

func TestPauseOffEventResumesPausedSessionWhenCurrentFacetIsWork(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
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
	start := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "facet", Facet: 1, OccurredAt: start, Source: "test"}); err != nil {
		t.Fatal(err)
	}
	if err := tracking.PauseTrackingAt(ctx, "d1", "test_pause", start.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "pause", OccurredAt: start.Add(3 * time.Minute), Source: "test"}); err != nil {
		t.Fatal(err)
	}
	open, err := st.GetOpenTaskSession(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if open.PauseStartedAt != nil || open.PausedSeconds != 60 {
		t.Fatalf("expected pause-off event to resume session, got %#v", open)
	}
	state, err := st.GetDeviceState(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if state.Paused {
		t.Fatalf("expected pause-off event to mark device unpaused: %#v", state)
	}
}

func TestUnassignedFacetActsLikePause(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 5, 25, 11, 0, 0, 0, time.UTC)
	store := &trackingMemoryStore{
		assignments: map[uint8]domain.FacetAssignment{
			1: {
				ID: "a1", DeviceID: "d1", Facet: 1, TaskID: "task-1",
				TaskLabelSnapshot: "Coding", TaskIconSnapshot: "code", TaskColorSnapshot: "#2255AA",
				EffectiveFrom: start,
			},
		},
	}
	tracking := NewTrackingService(store, fixedClock{t: start}, nil)
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "facet", Facet: 1, OccurredAt: start, Source: "test"}); err != nil {
		t.Fatal(err)
	}
	unassignedAt := start.Add(7 * time.Minute)
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "facet", Facet: 4, OccurredAt: unassignedAt, Source: "test"}); err != nil {
		t.Fatal(err)
	}
	if !store.state.Paused || store.state.CurrentFacet != 4 || !store.state.CurrentFacetKnown {
		t.Fatalf("expected unassigned facet to be active pause state, got %#v", store.state)
	}
	if store.session.PauseStartedAt == nil || !store.session.PauseStartedAt.Equal(unassignedAt) {
		t.Fatalf("expected session pause at unassigned flip time, got %#v", store.session)
	}
}

func TestApplyDeviceEventIgnoresNonTrackingEvents(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	st := store.NewSQLiteStore(db)
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, nil)
	start := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	if err := st.SaveDeviceState(ctx, domain.DeviceState{DeviceID: "d1", ConnectionState: domain.ConnectionConnected, CurrentFacet: 4, CurrentFacetKnown: true, Paused: true, UpdatedAt: start}); err != nil {
		t.Fatal(err)
	}
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "battery", OccurredAt: start.Add(time.Minute), Source: "test"}); err != nil {
		t.Fatal(err)
	}
	state, err := st.GetDeviceState(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentFacet != 4 || !state.Paused || !state.UpdatedAt.Equal(start) {
		t.Fatalf("non-tracking event changed tracking state: %#v", state)
	}
}

func TestApplyDeviceEventUpdatesStoredDeviceState(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	st := store.NewSQLiteStore(db)
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	bus := &MemoryEventBus{}
	tracking := NewTrackingService(st, fixedClock{t: time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)}, bus)
	if err := st.SaveFacetAssignment(ctx, domain.FacetAssignment{
		ID: "a1", DeviceID: "d1", Facet: 3, TaskID: "task-1",
		TaskLabelSnapshot: "Coding", TaskIconSnapshot: "code", TaskColorSnapshot: "#2255AA",
		EffectiveFrom: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	occurred := time.Date(2026, 5, 25, 11, 0, 0, 0, time.UTC)
	if err := tracking.ApplyDeviceEvent(ctx, domain.DeviceEventRecord{DeviceID: "d1", Kind: "facet", Facet: 3, OccurredAt: occurred, Source: "test"}); err != nil {
		t.Fatal(err)
	}
	state, err := st.GetDeviceState(ctx, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if state.ConnectionState != domain.ConnectionConnected || !state.CurrentFacetKnown || state.CurrentFacet != 3 || state.Paused {
		t.Fatalf("unexpected device state: %#v", state)
	}
	if len(bus.Events) == 0 || bus.Events[0].Name != "device.state" {
		t.Fatalf("expected device.state event, got %#v", bus.Events)
	}
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

type trackingMemoryStore struct {
	assignments map[uint8]domain.FacetAssignment
	profiles    map[string]domain.DeviceProfile
	tapSettings map[string]domain.DeviceTapSettings
	state       domain.DeviceState
	session     domain.TaskSession
}

func (s *trackingMemoryStore) Migrate(context.Context) error { return nil }
func (s *trackingMemoryStore) Close() error                  { return nil }
func (s *trackingMemoryStore) SaveDeviceProfile(_ context.Context, profile domain.DeviceProfile) error {
	if s.profiles == nil {
		s.profiles = map[string]domain.DeviceProfile{}
	}
	s.profiles[profile.ID] = profile
	return nil
}
func (s *trackingMemoryStore) GetDeviceProfile(_ context.Context, deviceID string) (domain.DeviceProfile, error) {
	profile, ok := s.profiles[deviceID]
	if !ok {
		return domain.DeviceProfile{}, domain.ErrNotFound
	}
	return profile, nil
}
func (s *trackingMemoryStore) ListDeviceProfiles(context.Context) ([]domain.DeviceProfile, error) {
	profiles := make([]domain.DeviceProfile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		profiles = append(profiles, profile)
	}
	return profiles, nil
}
func (s *trackingMemoryStore) SaveTask(context.Context, domain.Task) error { return nil }
func (s *trackingMemoryStore) ListTasks(context.Context, bool) ([]domain.Task, error) {
	return nil, nil
}
func (s *trackingMemoryStore) ArchiveTask(context.Context, string) error { return nil }
func (s *trackingMemoryStore) SaveFacetAssignment(_ context.Context, a domain.FacetAssignment) error {
	if s.assignments == nil {
		s.assignments = map[uint8]domain.FacetAssignment{}
	}
	s.assignments[a.Facet] = a
	return nil
}
func (s *trackingMemoryStore) ListFacetAssignments(context.Context, string) ([]domain.FacetAssignment, error) {
	return nil, nil
}
func (s *trackingMemoryStore) GetFacetAssignment(_ context.Context, _ string, facet uint8) (domain.FacetAssignment, error) {
	assignment, ok := s.assignments[facet]
	if !ok {
		return domain.FacetAssignment{}, domain.ErrNotFound
	}
	return assignment, nil
}
func (s *trackingMemoryStore) SaveDeviceState(_ context.Context, state domain.DeviceState) error {
	s.state = state
	return nil
}
func (s *trackingMemoryStore) GetDeviceState(context.Context, string) (domain.DeviceState, error) {
	if s.state.DeviceID == "" {
		return domain.DeviceState{}, domain.ErrNotFound
	}
	return s.state, nil
}
func (s *trackingMemoryStore) SaveDeviceTapSettings(_ context.Context, settings domain.DeviceTapSettings) error {
	if s.tapSettings == nil {
		s.tapSettings = map[string]domain.DeviceTapSettings{}
	}
	s.tapSettings[settings.DeviceID] = settings
	return nil
}
func (s *trackingMemoryStore) GetDeviceTapSettings(_ context.Context, deviceID string) (domain.DeviceTapSettings, error) {
	settings, ok := s.tapSettings[deviceID]
	if !ok {
		return domain.DeviceTapSettings{}, domain.ErrNotFound
	}
	return settings, nil
}
func (s *trackingMemoryStore) ListDeviceTapSettings(context.Context) ([]domain.DeviceTapSettings, error) {
	settings := make([]domain.DeviceTapSettings, 0, len(s.tapSettings))
	for _, item := range s.tapSettings {
		settings = append(settings, item)
	}
	return settings, nil
}
func (s *trackingMemoryStore) SaveDeviceLEDSettings(context.Context, domain.DeviceLEDSettings) error {
	return nil
}
func (s *trackingMemoryStore) GetDeviceLEDSettings(context.Context, string) (domain.DeviceLEDSettings, error) {
	return domain.DeviceLEDSettings{}, domain.ErrNotFound
}
func (s *trackingMemoryStore) ListDeviceLEDSettings(context.Context) ([]domain.DeviceLEDSettings, error) {
	return nil, nil
}
func (s *trackingMemoryStore) InsertDeviceEvent(context.Context, domain.DeviceEventRecord) error {
	return nil
}
func (s *trackingMemoryStore) ListDeviceEvents(context.Context, string) ([]domain.DeviceEventRecord, error) {
	return nil, nil
}
func (s *trackingMemoryStore) SaveTaskSession(_ context.Context, session domain.TaskSession) error {
	s.session = session
	return nil
}
func (s *trackingMemoryStore) ListTaskSessions(context.Context, domain.TaskSessionFilter) ([]domain.TaskSession, error) {
	return nil, nil
}
func (s *trackingMemoryStore) GetOpenTaskSession(context.Context, string) (domain.TaskSession, error) {
	if s.session.ID == "" || s.session.EndedAt != nil {
		return domain.TaskSession{}, domain.ErrNotFound
	}
	return s.session, nil
}
func (s *trackingMemoryStore) SaveConfig(context.Context, domain.AppConfig) error { return nil }
func (s *trackingMemoryStore) LoadConfig(context.Context) (domain.AppConfig, error) {
	return domain.DefaultAppConfig(), nil
}
