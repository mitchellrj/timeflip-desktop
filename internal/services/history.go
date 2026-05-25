package services

import (
	"context"
	"sort"
	"strconv"

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

func (s *HistoryService) ImportDeviceHistory(ctx context.Context, handle device.Handle) (int, error) {
	if s.client == nil || handle == nil {
		return 0, nil
	}
	events, err := s.client.ReadHistory(ctx, handle, device.HistoryRequest{All: true})
	if err != nil {
		return 0, err
	}
	return s.ReconcileEventsToSessions(ctx, handle.DeviceID(), events)
}

func (s *HistoryService) ReconcileEventsToSessions(ctx context.Context, deviceID string, events []domain.DeviceEventRecord) (int, error) {
	seen, err := s.existingEventKeys(ctx, deviceID)
	if err != nil {
		return 0, err
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].EventNumber == events[j].EventNumber {
			return events[i].OccurredAt.Before(events[j].OccurredAt)
		}
		return events[i].EventNumber < events[j].EventNumber
	})
	imported := 0
	for _, event := range events {
		if event.DeviceID == "" {
			event.DeviceID = deviceID
		}
		key := eventKey(event)
		if seen[key] {
			continue
		}
		if err := s.tracking.ApplyDeviceEvent(ctx, event); err != nil {
			return imported, domain.TrackingError{AppError: domain.NewAppError(domain.ErrHistoryReconciliationFailed, "Could not reconcile device history.", err.Error(), err)}
		}
		seen[key] = true
		imported++
	}
	return imported, nil
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

func (s *HistoryService) existingEventKeys(ctx context.Context, deviceID string) (map[string]bool, error) {
	events, err := s.store.ListDeviceEvents(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(events))
	for _, event := range events {
		out[eventKey(event)] = true
	}
	return out, nil
}

func eventKey(event domain.DeviceEventRecord) string {
	if event.EventNumber > 0 {
		return "number:" + strconv.FormatUint(uint64(event.EventNumber), 10)
	}
	return "event:" + event.Kind + ":" + strconv.Itoa(int(event.Facet)) + ":" + strconv.FormatBool(event.Pause) + ":" + event.OccurredAt.UTC().Format("2006-01-02T15:04:05.000000000Z07:00")
}
