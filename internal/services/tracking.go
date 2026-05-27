package services

import (
	"context"
	"errors"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/store"
)

type TrackingService struct {
	store store.Store
	clock Clock
	bus   EventBus
}

func NewTrackingService(store store.Store, clock Clock, bus EventBus) *TrackingService {
	if clock == nil {
		clock = SystemClock{}
	}
	if bus == nil {
		bus = NoopEventBus{}
	}
	return &TrackingService{store: store, clock: clock, bus: bus}
}

func (s *TrackingService) ApplyDeviceSnapshot(ctx context.Context, snapshot domain.DeviceSnapshot) error {
	state := snapshot.State
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = s.clock.Now()
	}
	if err := s.store.SaveDeviceState(ctx, state); err != nil {
		return err
	}
	if !state.CurrentFacetKnown || state.CurrentFacetUndefined || state.Paused {
		return s.PauseTracking(ctx, state.DeviceID, "snapshot")
	}
	assignment, err := s.ResolveActiveAssignment(ctx, state.DeviceID, state.CurrentFacet)
	if err != nil {
		if isStoreNotFound(err) {
			return s.PauseTrackingAtWithState(ctx, state.DeviceID, "unassigned_facet", state.UpdatedAt)
		}
		return err
	}
	return s.StartSessionForAssignment(ctx, state.DeviceID, assignment, domain.DeviceEventRecord{
		DeviceID:   state.DeviceID,
		Kind:       "snapshot",
		Facet:      state.CurrentFacet,
		OccurredAt: state.UpdatedAt,
		Source:     "snapshot",
	})
}

func (s *TrackingService) ApplyDeviceEvent(ctx context.Context, event domain.DeviceEventRecord) error {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = s.clock.Now()
	}
	if err := s.store.InsertDeviceEvent(ctx, event); err != nil {
		return err
	}
	if !isTrackingEvent(event) {
		return nil
	}
	if event.Pause {
		if err := s.setPausedState(ctx, event.DeviceID, true, event.OccurredAt); err != nil {
			return err
		}
		return s.PauseTrackingAt(ctx, event.DeviceID, "device_pause", event.OccurredAt)
	}
	if event.Facet == 0 {
		if err := s.setPausedState(ctx, event.DeviceID, false, event.OccurredAt); err != nil {
			return err
		}
		return s.ResumeTrackingAt(ctx, event.DeviceID, "device_resume", event.OccurredAt)
	}
	if event.Facet > 0 {
		current, err := s.store.GetDeviceState(ctx, event.DeviceID)
		if err != nil && !isStoreNotFound(err) {
			return err
		}
		if current.Locked {
			current.ConnectionState = domain.ConnectionConnected
			current.UpdatedAt = event.OccurredAt
			if current.DeviceID == "" {
				current.DeviceID = event.DeviceID
			}
			if err := s.store.SaveDeviceState(ctx, current); err != nil {
				return err
			}
			s.bus.Publish(ctx, "device.state", current)
			return nil
		}
	}
	_, err := s.updateDeviceStateFromEvent(ctx, event)
	if err != nil {
		return err
	}
	assignment, err := s.ResolveActiveAssignment(ctx, event.DeviceID, event.Facet)
	if err != nil {
		if isStoreNotFound(err) {
			return s.PauseTrackingAtWithState(ctx, event.DeviceID, "unassigned_facet", event.OccurredAt)
		}
		return err
	}
	return s.StartSessionForAssignment(ctx, event.DeviceID, assignment, event)
}

func isTrackingEvent(event domain.DeviceEventRecord) bool {
	switch event.Kind {
	case "facet", "double_tap", "history", "pause", "resume", "pause_state":
		return true
	default:
		return event.Facet > 0 || event.Pause
	}
}

func (s *TrackingService) updateDeviceStateFromEvent(ctx context.Context, event domain.DeviceEventRecord) (domain.DeviceState, error) {
	state, err := s.store.GetDeviceState(ctx, event.DeviceID)
	if err != nil {
		if !isStoreNotFound(err) {
			return domain.DeviceState{}, err
		}
		state = domain.DeviceState{DeviceID: event.DeviceID}
	}
	state.ConnectionState = domain.ConnectionConnected
	if event.Facet > 0 {
		state.CurrentFacet = event.Facet
		state.CurrentFacetKnown = true
		state.CurrentFacetUndefined = false
		if state.Paused && !state.Locked {
			state.Paused = false
		}
	}
	state.UpdatedAt = event.OccurredAt
	if err := s.store.SaveDeviceState(ctx, state); err != nil {
		return domain.DeviceState{}, err
	}
	s.bus.Publish(ctx, "device.state", state)
	return state, nil
}

func (s *TrackingService) ResolveActiveAssignment(ctx context.Context, deviceID string, facet uint8) (domain.FacetAssignment, error) {
	return s.store.GetFacetAssignment(ctx, deviceID, facet)
}

func (s *TrackingService) StartSessionForAssignment(ctx context.Context, deviceID string, assignment domain.FacetAssignment, event domain.DeviceEventRecord) error {
	if assignment.IsPauseAssignment {
		if err := s.setPausedState(ctx, deviceID, true, event.OccurredAt); err != nil {
			return err
		}
		return s.PauseTrackingAt(ctx, deviceID, "pause_facet", event.OccurredAt)
	}
	if err := s.setPausedState(ctx, deviceID, false, event.OccurredAt); err != nil {
		return err
	}
	open, err := s.store.GetOpenTaskSession(ctx, deviceID)
	if err == nil {
		if open.TaskID == assignment.TaskID && open.Facet == assignment.Facet {
			return s.resumeOpenSession(ctx, open, event.OccurredAt)
		}
		if err := s.closeSession(ctx, open, event); err != nil {
			return err
		}
	} else if !isStoreNotFound(err) {
		return err
	}
	session, created, err := domain.StartTaskSession(deviceID, assignment, event)
	if err != nil || !created {
		return err
	}
	if err := s.store.SaveTaskSession(ctx, session); err != nil {
		return err
	}
	s.bus.Publish(ctx, "tracking.session.started", session)
	return nil
}

func (s *TrackingService) PauseTracking(ctx context.Context, deviceID string, source string) error {
	return s.PauseTrackingAt(ctx, deviceID, source, s.clock.Now())
}

func (s *TrackingService) PauseTrackingAt(ctx context.Context, deviceID string, source string, pausedAt time.Time) error {
	open, err := s.store.GetOpenTaskSession(ctx, deviceID)
	if err != nil {
		if isStoreNotFound(err) {
			return nil
		}
		return err
	}
	if open.PauseStartedAt != nil {
		return nil
	}
	if pausedAt.IsZero() {
		pausedAt = s.clock.Now()
	}
	pausedAt = pausedAt.UTC()
	open.PauseStartedAt = &pausedAt
	if err := s.store.SaveTaskSession(ctx, open); err != nil {
		return err
	}
	s.bus.Publish(ctx, "tracking.session.updated", open)
	return nil
}

func (s *TrackingService) PauseTrackingAtWithState(ctx context.Context, deviceID string, source string, pausedAt time.Time) error {
	if err := s.setPausedState(ctx, deviceID, true, pausedAt); err != nil {
		return err
	}
	return s.PauseTrackingAt(ctx, deviceID, source, pausedAt)
}

func (s *TrackingService) ResumeTracking(ctx context.Context, deviceID string, source string) error {
	return s.ResumeTrackingAt(ctx, deviceID, source, s.clock.Now())
}

func (s *TrackingService) ResumeTrackingAt(ctx context.Context, deviceID string, source string, resumedAt time.Time) error {
	state, err := s.store.GetDeviceState(ctx, deviceID)
	if err != nil {
		return err
	}
	if !state.CurrentFacetKnown || state.CurrentFacetUndefined {
		return nil
	}
	assignment, err := s.ResolveActiveAssignment(ctx, deviceID, state.CurrentFacet)
	if err != nil {
		return nil
	}
	return s.StartSessionForAssignment(ctx, deviceID, assignment, domain.DeviceEventRecord{DeviceID: deviceID, Kind: "resume", Facet: state.CurrentFacet, OccurredAt: resumedAt, Source: source})
}

func (s *TrackingService) CloseOpenSession(ctx context.Context, deviceID string, event domain.DeviceEventRecord) error {
	open, err := s.store.GetOpenTaskSession(ctx, deviceID)
	if err != nil {
		if isStoreNotFound(err) {
			return nil
		}
		return err
	}
	return s.closeSession(ctx, open, event)
}

func (s *TrackingService) closeSession(ctx context.Context, open domain.TaskSession, event domain.DeviceEventRecord) error {
	endedAt := event.OccurredAt
	if endedAt.IsZero() {
		endedAt = s.clock.Now()
	}
	closed, meaningful, err := domain.EndTaskSession(open, endedAt, event.EventNumber)
	if err != nil || !meaningful {
		return err
	}
	if err := s.store.SaveTaskSession(ctx, closed); err != nil {
		return err
	}
	s.bus.Publish(ctx, "tracking.session.ended", closed)
	return nil
}

func (s *TrackingService) resumeOpenSession(ctx context.Context, open domain.TaskSession, resumedAt time.Time) error {
	if open.PauseStartedAt == nil {
		return nil
	}
	if resumedAt.IsZero() {
		resumedAt = s.clock.Now()
	}
	resumedAt = resumedAt.UTC()
	pauseStarted := open.PauseStartedAt.UTC()
	if resumedAt.After(pauseStarted) {
		open.PausedSeconds += uint32(resumedAt.Sub(pauseStarted).Seconds())
	}
	open.PauseStartedAt = nil
	if err := s.store.SaveTaskSession(ctx, open); err != nil {
		return err
	}
	s.bus.Publish(ctx, "tracking.session.updated", open)
	return nil
}

func (s *TrackingService) setPausedState(ctx context.Context, deviceID string, paused bool, updatedAt time.Time) error {
	state, err := s.store.GetDeviceState(ctx, deviceID)
	if err != nil {
		if !isStoreNotFound(err) {
			return err
		}
		state = domain.DeviceState{DeviceID: deviceID}
	}
	if updatedAt.IsZero() {
		updatedAt = s.clock.Now()
	}
	state.Paused = paused
	state.UpdatedAt = updatedAt.UTC()
	if state.ConnectionState == "" {
		state.ConnectionState = domain.ConnectionConnected
	}
	if err := s.store.SaveDeviceState(ctx, state); err != nil {
		return err
	}
	s.bus.Publish(ctx, "device.state", state)
	return nil
}

func isStoreNotFound(err error) bool {
	var appErr *domain.AppError
	return errors.Is(err, domain.ErrNotFound) || (errors.As(err, &appErr) && appErr.Code == domain.ErrDeviceNotFound)
}
