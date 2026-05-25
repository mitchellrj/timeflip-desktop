package services

import (
	"context"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/device"
	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/store"
)

type DeviceService struct {
	client   device.Client
	store    store.Store
	tasks    *TaskService
	tracking *TrackingService
	history  *HistoryService
	bus      EventBus
	clock    Clock
	handles  map[string]device.Handle
}

func NewDeviceService(client device.Client, store store.Store, tasks *TaskService, tracking *TrackingService, history *HistoryService, bus EventBus, clock Clock) *DeviceService {
	if bus == nil {
		bus = NoopEventBus{}
	}
	if clock == nil {
		clock = SystemClock{}
	}
	return &DeviceService{
		client:   client,
		store:    store,
		tasks:    tasks,
		tracking: tracking,
		history:  history,
		bus:      bus,
		clock:    clock,
		handles:  map[string]device.Handle{},
	}
}

func (s *DeviceService) ListDevices(ctx context.Context) ([]domain.DiscoveredDevice, error) {
	devices, err := s.client.Scan(ctx, false)
	if err != nil {
		return nil, err
	}
	for _, found := range devices {
		if !found.Supported {
			continue
		}
		_ = s.store.SaveDeviceProfile(ctx, domain.DeviceProfile{
			ID:             found.ID,
			DisplayName:    firstNonEmpty(found.Name, found.ID),
			AdvertisedName: found.Name,
			PairingState:   "seen",
			LastSeenAt:     s.clock.Now(),
		})
	}
	s.bus.Publish(ctx, "devices.scanned", devices)
	return devices, nil
}

func (s *DeviceService) PairDevice(ctx context.Context, deviceID string, password string, newPassword string, allowOSPairing bool) (domain.PairingWorkflow, error) {
	result, err := s.client.Pair(ctx, device.PairRequest{
		DeviceID:       deviceID,
		Password:       password,
		NewPassword:    newPassword,
		AllowOSPairing: allowOSPairing,
		Timeout:        5 * time.Second,
	})
	if result.DeviceID == "" {
		result.DeviceID = deviceID
	}
	if result.Completed || result.ManualAction != nil {
		storedPassword := password
		if newPassword != "" {
			storedPassword = newPassword
		}
		if storedPassword == "" {
			storedPassword = "000000"
		}
		_ = s.store.SaveDeviceProfile(ctx, domain.DeviceProfile{
			ID:             deviceID,
			DisplayName:    deviceID,
			StoredPassword: storedPassword,
			PairingState:   result.CurrentStage,
			LastSeenAt:     s.clock.Now(),
		})
	}
	s.bus.Publish(ctx, "device.pairing", result)
	return result, err
}

func (s *DeviceService) UnpairDevice(ctx context.Context, deviceID string, factoryReset bool, allowOSUnpairing bool) (domain.UnpairingWorkflow, error) {
	profile, _ := s.store.GetDeviceProfile(ctx, deviceID)
	result, err := s.client.Unpair(ctx, device.UnpairRequest{
		DeviceID:         deviceID,
		Password:         profile.StoredPassword,
		FactoryReset:     factoryReset,
		AllowOSUnpairing: allowOSUnpairing,
		Timeout:          5 * time.Second,
	})
	if result.DeviceID == "" {
		result.DeviceID = deviceID
	}
	if result.Completed {
		profile.PairingState = "unpaired"
		_ = s.store.SaveDeviceProfile(ctx, profile)
	}
	s.bus.Publish(ctx, "device.unpairing", result)
	return result, err
}

func (s *DeviceService) RefreshDeviceState(ctx context.Context, deviceID string) (domain.DeviceSnapshot, error) {
	handle, err := s.ensureHandle(ctx, deviceID)
	if err != nil {
		return domain.DeviceSnapshot{}, err
	}
	snapshot, err := s.client.ReadDeviceSnapshot(ctx, handle)
	if err != nil {
		return domain.DeviceSnapshot{}, err
	}
	if profile, err := s.store.GetDeviceProfile(ctx, deviceID); err == nil {
		snapshot.Profile.StoredPassword = profile.StoredPassword
	}
	if err := s.store.SaveDeviceProfile(ctx, snapshot.Profile); err != nil {
		return domain.DeviceSnapshot{}, err
	}
	if err := s.tracking.ApplyDeviceSnapshot(ctx, snapshot); err != nil {
		return domain.DeviceSnapshot{}, err
	}
	s.bus.Publish(ctx, "device.state", snapshot)
	return snapshot, nil
}

func (s *DeviceService) ConfigureFacet(ctx context.Context, req domain.FacetConfigurationRequest) (domain.FacetConfigurationView, error) {
	assignment, err := s.tasks.AssignFacet(ctx, req)
	if err != nil {
		return domain.FacetConfigurationView{}, err
	}
	view := domain.FacetConfigurationView{
		Facet:                assignment.Facet,
		TaskID:               assignment.TaskID,
		Label:                assignment.TaskLabelSnapshot,
		Icon:                 assignment.TaskIconSnapshot,
		Color:                assignment.TaskColorSnapshot,
		IsPauseAssignment:    assignment.IsPauseAssignment,
		PomodoroLimitSeconds: assignment.PomodoroLimitSeconds,
	}
	handle, ok := s.handles[req.DeviceID]
	if ok {
		deviceView, err := s.client.WriteFacetConfiguration(ctx, handle, assignment)
		if err != nil {
			s.bus.Publish(ctx, "device.configuration.unconfirmed", view)
			return view, err
		}
		assignment.ConfirmedOnDevice = deviceView.AssignedOnDevice
		_ = s.store.SaveFacetAssignment(ctx, assignment)
		view.AssignedOnDevice = deviceView.AssignedOnDevice
	}
	s.bus.Publish(ctx, "device.facet.saved", view)
	return view, nil
}

func (s *DeviceService) ConfigureTapPause(ctx context.Context, deviceID string, paused bool) error {
	return s.SetPaused(ctx, deviceID, paused)
}

func (s *DeviceService) SetPaused(ctx context.Context, deviceID string, paused bool) error {
	handle, err := s.ensureHandle(ctx, deviceID)
	if err != nil {
		return err
	}
	if err := s.client.SetPause(ctx, handle, paused); err != nil {
		return err
	}
	if paused {
		return s.tracking.PauseTracking(ctx, deviceID, "user_pause")
	}
	return s.tracking.ResumeTracking(ctx, deviceID, "user_resume")
}

func (s *DeviceService) ConnectDevice(ctx context.Context, deviceID string) error {
	_, err := s.ensureHandle(ctx, deviceID)
	return err
}

func (s *DeviceService) DisconnectDevice(ctx context.Context, deviceID string) error {
	handle, ok := s.handles[deviceID]
	if !ok {
		return nil
	}
	delete(s.handles, deviceID)
	return s.client.Close(ctx, handle)
}

func (s *DeviceService) ensureHandle(ctx context.Context, deviceID string) (device.Handle, error) {
	if handle, ok := s.handles[deviceID]; ok {
		return handle, nil
	}
	profile, err := s.store.GetDeviceProfile(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "device.connection", domain.DeviceState{DeviceID: deviceID, ConnectionState: domain.ConnectionConnecting, UpdatedAt: s.clock.Now()})
	handle, err := s.client.Connect(ctx, device.ConnectRequest{
		DeviceID:        deviceID,
		AdvertisedName:  profile.AdvertisedName,
		ProtocolVersion: profile.ProtocolVersion,
		Timeout:         10 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "device.connection", domain.DeviceState{DeviceID: deviceID, ConnectionState: domain.ConnectionAuthorizing, UpdatedAt: s.clock.Now()})
	if err := s.client.Authorize(ctx, handle, profile.StoredPassword); err != nil {
		_ = s.client.Close(ctx, handle)
		return nil, err
	}
	profile.LastConnectedAt = s.clock.Now()
	profile.PairingState = "paired"
	_ = s.store.SaveDeviceProfile(ctx, profile)
	s.handles[deviceID] = handle
	return handle, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
