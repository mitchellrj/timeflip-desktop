package services

import (
	"context"
	"errors"

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
	if !state.CurrentFacetKnown || state.CurrentFacetUndefined || state.Locked || state.Paused {
		return s.PauseTracking(ctx, state.DeviceID, "snapshot")
	}
	assignment, err := s.ResolveActiveAssignment(ctx, state.DeviceID, state.CurrentFacet)
	if err != nil {
		return s.PauseTracking(ctx, state.DeviceID, "unassigned_facet")
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
	if event.Pause {
		return s.CloseOpenSession(ctx, event.DeviceID, event)
	}
	if event.Facet == 0 {
		return nil
	}
	assignment, err := s.ResolveActiveAssignment(ctx, event.DeviceID, event.Facet)
	if err != nil {
		return s.PauseTracking(ctx, event.DeviceID, "unassigned_facet")
	}
	return s.StartSessionForAssignment(ctx, event.DeviceID, assignment, event)
}

func (s *TrackingService) ResolveActiveAssignment(ctx context.Context, deviceID string, facet uint8) (domain.FacetAssignment, error) {
	return s.store.GetFacetAssignment(ctx, deviceID, facet)
}

func (s *TrackingService) StartSessionForAssignment(ctx context.Context, deviceID string, assignment domain.FacetAssignment, event domain.DeviceEventRecord) error {
	if assignment.IsPauseAssignment {
		return s.CloseOpenSession(ctx, deviceID, event)
	}
	open, err := s.store.GetOpenTaskSession(ctx, deviceID)
	if err == nil {
		if open.TaskID == assignment.TaskID && open.Facet == assignment.Facet {
			return nil
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
	open, err := s.store.GetOpenTaskSession(ctx, deviceID)
	if err != nil {
		if isStoreNotFound(err) {
			return nil
		}
		return err
	}
	return s.closeSession(ctx, open, domain.DeviceEventRecord{DeviceID: deviceID, Source: source, OccurredAt: s.clock.Now()})
}

func (s *TrackingService) ResumeTracking(ctx context.Context, deviceID string, source string) error {
	state, err := s.store.GetDeviceState(ctx, deviceID)
	if err != nil {
		return err
	}
	if !state.CurrentFacetKnown || state.CurrentFacetUndefined || state.Locked {
		return nil
	}
	assignment, err := s.ResolveActiveAssignment(ctx, deviceID, state.CurrentFacet)
	if err != nil {
		return nil
	}
	return s.StartSessionForAssignment(ctx, deviceID, assignment, domain.DeviceEventRecord{DeviceID: deviceID, Kind: "resume", Facet: state.CurrentFacet, OccurredAt: s.clock.Now(), Source: source})
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

func isStoreNotFound(err error) bool {
	var appErr *domain.AppError
	return errors.Is(err, domain.ErrNotFound) || (errors.As(err, &appErr) && appErr.Code == domain.ErrDeviceNotFound)
}
