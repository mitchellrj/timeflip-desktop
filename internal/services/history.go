package services

import (
	"context"
	"sort"

	"github.com/mitchellrj/timeflip-desktop/internal/device"
	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/store"
)

type HistoryService struct {
	store    store.Store
	client   device.Client
	tracking *TrackingService
}

func NewHistoryService(store store.Store, client device.Client, tracking *TrackingService) *HistoryService {
	return &HistoryService{store: store, client: client, tracking: tracking}
}

func (s *HistoryService) ImportDeviceHistory(ctx context.Context, handle device.Handle) error {
	if s.client == nil || handle == nil {
		return nil
	}
	events, err := s.client.ReadHistory(ctx, handle, device.HistoryRequest{All: true})
	if err != nil {
		return err
	}
	return s.ReconcileEventsToSessions(ctx, handle.DeviceID(), events)
}

func (s *HistoryService) ReconcileEventsToSessions(ctx context.Context, deviceID string, events []domain.DeviceEventRecord) error {
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].EventNumber == events[j].EventNumber {
			return events[i].OccurredAt.Before(events[j].OccurredAt)
		}
		return events[i].EventNumber < events[j].EventNumber
	})
	for _, event := range events {
		if event.DeviceID == "" {
			event.DeviceID = deviceID
		}
		if err := s.tracking.ApplyDeviceEvent(ctx, event); err != nil {
			return domain.TrackingError{AppError: domain.NewAppError(domain.ErrHistoryReconciliationFailed, "Could not reconcile device history.", err.Error(), err)}
		}
	}
	return nil
}

func (s *HistoryService) ListTaskSessions(ctx context.Context, filter domain.TaskSessionFilter) ([]domain.TaskSession, error) {
	return s.store.ListTaskSessions(ctx, filter)
}

func (s *HistoryService) GetCurrentSession(ctx context.Context, deviceID string) (*domain.TaskSession, error) {
	session, err := s.store.GetOpenTaskSession(ctx, deviceID)
	if err != nil {
		if isStoreNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}
