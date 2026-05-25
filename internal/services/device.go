package services

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/device"
	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/store"
)

type DeviceService struct {
	client    device.Client
	store     store.Store
	tasks     *TaskService
	tracking  *TrackingService
	history   *HistoryService
	bus       EventBus
	clock     Clock
	mu        sync.Mutex
	connectMu sync.Mutex
	handles   map[string]device.Handle
	streams   map[string]context.CancelFunc
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
		streams:  map[string]context.CancelFunc{},
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
	s.stopEventStream(deviceID)
	if handle, ok := s.takeHandle(deviceID); ok {
		_ = s.closeHandle(ctx, handle)
		_ = s.resetDeviceState(ctx, deviceID)
	}
	profile, _ := s.store.GetDeviceProfile(ctx, deviceID)
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
	_ = s.resetDeviceState(ctx, deviceID)
	if result.Completed || result.ManualAction != nil {
		if stored, err := s.store.GetDeviceProfile(ctx, deviceID); err == nil {
			profile = stored
		}
		storedPassword := password
		if newPassword != "" {
			storedPassword = newPassword
		}
		if storedPassword == "" {
			storedPassword = "000000"
		}
		profile.ID = deviceID
		if profile.DisplayName == "" {
			profile.DisplayName = deviceID
		}
		profile.StoredPassword = storedPassword
		profile.PairingState = result.CurrentStage
		if result.Completed {
			profile.PairingState = "paired"
		}
		profile.LastSeenAt = s.clock.Now()
		_ = s.store.SaveDeviceProfile(ctx, profile)
	}
	s.bus.Publish(ctx, "device.pairing", result)
	return result, err
}

func (s *DeviceService) UnpairDevice(ctx context.Context, deviceID string, factoryReset bool, allowOSUnpairing bool) (domain.UnpairingWorkflow, error) {
	s.stopEventStream(deviceID)
	if handle, ok := s.takeHandle(deviceID); ok {
		s.closeHandleAsync(handle)
	}
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
		if s.tracking != nil {
			_ = s.tracking.CloseOpenSession(ctx, deviceID, domain.DeviceEventRecord{
				DeviceID:   deviceID,
				Kind:       "unpair",
				OccurredAt: s.clock.Now(),
				Source:     "user_unpair",
			})
		}
		_ = s.resetDeviceState(ctx, deviceID)
	}
	s.bus.Publish(ctx, "device.unpairing", result)
	return result, err
}

func (s *DeviceService) RefreshDeviceState(ctx context.Context, deviceID string) (domain.DeviceSnapshot, error) {
	handle, err := s.ensureHandle(ctx, deviceID)
	if err != nil {
		return domain.DeviceSnapshot{}, err
	}
	return s.readAndSaveDeviceSnapshot(ctx, deviceID, handle)
}

func (s *DeviceService) readAndSaveDeviceSnapshot(ctx context.Context, deviceID string, handle device.Handle) (domain.DeviceSnapshot, error) {
	snapshot, err := s.client.ReadDeviceSnapshot(ctx, handle)
	if err != nil {
		s.removeHandle(deviceID)
		_ = s.saveConnectionState(ctx, deviceID, domain.ConnectionReconnecting, err.Error())
		s.closeHandleAsync(handle)
		return domain.DeviceSnapshot{}, err
	}
	if profile, err := s.store.GetDeviceProfile(ctx, deviceID); err == nil {
		snapshot.Profile.StoredPassword = profile.StoredPassword
		if profile.DisplayName != "" && profile.DisplayName != profile.AdvertisedName {
			snapshot.Profile.DisplayName = profile.DisplayName
		}
	}
	if err := s.store.SaveDeviceProfile(ctx, snapshot.Profile); err != nil {
		return domain.DeviceSnapshot{}, err
	}
	if err := s.tracking.ApplyDeviceSnapshot(ctx, snapshot); err != nil {
		return domain.DeviceSnapshot{}, err
	}
	if snapshot.TapSettings.DeviceID != "" {
		_ = s.store.SaveDeviceTapSettings(ctx, snapshot.TapSettings)
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
		DeviceID:             req.DeviceID,
		Facet:                assignment.Facet,
		TaskID:               assignment.TaskID,
		Label:                assignment.TaskLabelSnapshot,
		Icon:                 assignment.TaskIconSnapshot,
		Color:                assignment.TaskColorSnapshot,
		IsPauseAssignment:    assignment.IsPauseAssignment,
		IsPomodoroAssignment: assignment.IsPomodoroAssignment,
		PomodoroLimitSeconds: assignment.PomodoroLimitSeconds,
	}
	handle, ok := s.currentHandle(req.DeviceID)
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

func (s *DeviceService) ConfigureTapSettings(ctx context.Context, settings domain.DeviceTapSettings) (domain.DeviceTapSettings, error) {
	if settings.UpdatedAt.IsZero() {
		settings.UpdatedAt = s.clock.Now()
	}
	settings.ConfirmedOnDevice = false
	if err := domain.ValidateDeviceTapSettings(settings); err != nil {
		return domain.DeviceTapSettings{}, err
	}
	if err := s.store.SaveDeviceTapSettings(ctx, settings); err != nil {
		return domain.DeviceTapSettings{}, err
	}
	handle, ok := s.currentHandle(settings.DeviceID)
	if !ok {
		s.bus.Publish(ctx, "device.tap.saved", settings)
		return settings, nil
	}
	if err := s.writeTapSettings(ctx, handle, settings); err != nil {
		s.bus.Publish(ctx, "device.configuration.unconfirmed", settings)
		return settings, err
	}
	settings.ConfirmedOnDevice = true
	settings.UpdatedAt = s.clock.Now()
	if err := s.store.SaveDeviceTapSettings(ctx, settings); err != nil {
		return domain.DeviceTapSettings{}, err
	}
	s.bus.Publish(ctx, "device.tap.saved", settings)
	return settings, nil
}

func (s *DeviceService) ConfigureLEDSettings(ctx context.Context, settings domain.DeviceLEDSettings) (domain.DeviceLEDSettings, error) {
	if settings.UpdatedAt.IsZero() {
		settings.UpdatedAt = s.clock.Now()
	}
	settings.ConfirmedOnDevice = false
	if err := domain.ValidateDeviceLEDSettings(settings); err != nil {
		return domain.DeviceLEDSettings{}, err
	}
	if err := s.store.SaveDeviceLEDSettings(ctx, settings); err != nil {
		return domain.DeviceLEDSettings{}, err
	}
	handle, ok := s.currentHandle(settings.DeviceID)
	if !ok {
		s.bus.Publish(ctx, "device.led.saved", settings)
		return settings, nil
	}
	if err := s.writeLEDSettings(ctx, handle, settings); err != nil {
		s.bus.Publish(ctx, "device.configuration.unconfirmed", settings)
		return settings, err
	}
	settings.ConfirmedOnDevice = true
	settings.UpdatedAt = s.clock.Now()
	if err := s.store.SaveDeviceLEDSettings(ctx, settings); err != nil {
		return domain.DeviceLEDSettings{}, err
	}
	s.bus.Publish(ctx, "device.led.saved", settings)
	return settings, nil
}

func (s *DeviceService) ConfigureDeviceName(ctx context.Context, deviceID string, name string) (domain.DeviceProfileView, error) {
	deviceID = strings.TrimSpace(deviceID)
	name = strings.TrimSpace(name)
	if deviceID == "" {
		return domain.DeviceProfileView{}, domain.ValidationError{AppError: domain.NewAppError(domain.ErrValidation, "Device ID is required.", "device id is empty", nil)}
	}
	if err := domain.ValidateDeviceName(name); err != nil {
		return domain.DeviceProfileView{}, err
	}
	profile, err := s.store.GetDeviceProfile(ctx, deviceID)
	if err != nil && !isStoreNotFound(err) {
		return domain.DeviceProfileView{}, err
	}
	if profile.ID == "" {
		profile.ID = deviceID
	}
	profile.DisplayName = name
	if err := s.store.SaveDeviceProfile(ctx, profile); err != nil {
		return domain.DeviceProfileView{}, err
	}
	view := profile.View()
	handle, ok := s.currentHandle(deviceID)
	if !ok {
		s.bus.Publish(ctx, "device.profile.saved", view)
		return view, nil
	}
	if err := s.client.SetDeviceName(ctx, handle, name); err != nil {
		s.bus.Publish(ctx, "device.configuration.unconfirmed", map[string]any{"deviceID": deviceID, "setting": "deviceName", "error": err.Error()})
		s.bus.Publish(ctx, "device.profile.saved", view)
		return view, err
	}
	profile.AdvertisedName = name
	profile.LastSeenAt = s.clock.Now()
	if err := s.store.SaveDeviceProfile(ctx, profile); err != nil {
		return domain.DeviceProfileView{}, err
	}
	view = profile.View()
	s.bus.Publish(ctx, "device.profile.saved", view)
	return view, nil
}

func (s *DeviceService) SetPaused(ctx context.Context, deviceID string, paused bool) error {
	handle, err := s.ensureHandle(ctx, deviceID)
	if err != nil {
		return err
	}
	if err := s.client.SetPause(ctx, handle, paused); err != nil {
		return err
	}
	if err := s.savePausedState(ctx, deviceID, paused); err != nil {
		return err
	}
	if paused {
		return s.tracking.PauseTracking(ctx, deviceID, "user_pause")
	}
	return s.tracking.ResumeTracking(ctx, deviceID, "user_resume")
}

func (s *DeviceService) SetLocked(ctx context.Context, deviceID string, locked bool) error {
	handle, err := s.ensureHandle(ctx, deviceID)
	if err != nil {
		return err
	}
	if err := s.client.SetLock(ctx, handle, locked); err != nil {
		return err
	}
	return s.saveLockedState(ctx, deviceID, locked)
}

func (s *DeviceService) ConnectDevice(ctx context.Context, deviceID string) error {
	if handle, ok := s.currentHandle(deviceID); ok {
		_ = s.saveConnectionState(ctx, deviceID, domain.ConnectionConnected, "")
		s.startEventStream(ctx, deviceID, handle)
		return nil
	}
	handle, connectedNow, err := s.ensureHandleState(ctx, deviceID)
	if err != nil {
		return err
	}
	if !connectedNow {
		_ = s.saveConnectionState(ctx, deviceID, domain.ConnectionConnected, "")
		s.startEventStream(ctx, deviceID, handle)
		return nil
	}
	if s.history != nil {
		imported, err := s.history.ImportDeviceHistory(ctx, handle)
		if err != nil {
			s.bus.Publish(ctx, "history.import_failed", map[string]any{"deviceID": deviceID, "error": err.Error()})
		} else {
			s.bus.Publish(ctx, "history.imported", map[string]any{"deviceID": deviceID, "events": imported})
		}
	}
	if _, err := s.readAndSaveDeviceSnapshot(ctx, deviceID, handle); err != nil {
		return err
	}
	s.startEventStream(ctx, deviceID, handle)
	return nil
}

func (s *DeviceService) DisconnectDevice(ctx context.Context, deviceID string) error {
	s.stopEventStream(deviceID)
	s.mu.Lock()
	handle, ok := s.handles[deviceID]
	if !ok {
		s.mu.Unlock()
		_ = s.saveConnectionState(ctx, deviceID, domain.ConnectionDisconnected, "")
		return nil
	}
	delete(s.handles, deviceID)
	s.mu.Unlock()
	_ = s.saveConnectionState(ctx, deviceID, domain.ConnectionDisconnected, "")
	return s.client.Close(ctx, handle)
}

func (s *DeviceService) ensureHandle(ctx context.Context, deviceID string) (device.Handle, error) {
	handle, _, err := s.ensureHandleState(ctx, deviceID)
	return handle, err
}

func (s *DeviceService) ensureHandleState(ctx context.Context, deviceID string) (device.Handle, bool, error) {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	s.mu.Lock()
	if handle, ok := s.handles[deviceID]; ok {
		s.mu.Unlock()
		return handle, false, nil
	}
	s.mu.Unlock()
	profile, err := s.store.GetDeviceProfile(ctx, deviceID)
	if err != nil {
		return nil, false, err
	}
	_ = s.saveConnectionState(ctx, deviceID, domain.ConnectionConnecting, "")
	handle, err := s.client.Connect(ctx, device.ConnectRequest{
		DeviceID:        deviceID,
		AdvertisedName:  profile.AdvertisedName,
		ProtocolVersion: connectProtocolVersion(profile),
		Timeout:         10 * time.Second,
	})
	if err != nil {
		_ = s.saveConnectionState(ctx, deviceID, domain.ConnectionError, err.Error())
		return nil, false, err
	}
	_ = s.saveConnectionState(ctx, deviceID, domain.ConnectionAuthorizing, "")
	if err := s.client.Authorize(ctx, handle, profile.StoredPassword); err != nil {
		_ = s.client.Close(ctx, handle)
		_ = s.saveConnectionState(ctx, deviceID, domain.ConnectionError, err.Error())
		return nil, false, err
	}
	profile.LastConnectedAt = s.clock.Now()
	profile.PairingState = "paired"
	_ = s.store.SaveDeviceProfile(ctx, profile)
	_ = s.saveConnectionState(ctx, deviceID, domain.ConnectionConnected, "")
	s.mu.Lock()
	s.handles[deviceID] = handle
	s.mu.Unlock()
	if err := s.applyStoredTapSettings(ctx, deviceID, handle); err != nil {
		s.bus.Publish(ctx, "device.configuration.unconfirmed", domain.DefaultDeviceTapSettings(deviceID))
	}
	if err := s.applyStoredLEDSettings(ctx, deviceID, handle); err != nil {
		s.bus.Publish(ctx, "device.configuration.unconfirmed", domain.DefaultDeviceLEDSettings(deviceID))
	}
	if err := s.applyStoredDeviceName(ctx, deviceID, handle); err != nil {
		s.bus.Publish(ctx, "device.configuration.unconfirmed", map[string]any{"deviceID": deviceID, "setting": "deviceName", "error": err.Error()})
	}
	if err := s.applyStoredFacetAssignments(ctx, deviceID, handle); err != nil {
		s.bus.Publish(ctx, "device.configuration.unconfirmed", map[string]any{"deviceID": deviceID, "error": err.Error()})
	}
	return handle, true, nil
}

func connectProtocolVersion(profile domain.DeviceProfile) string {
	if profile.ProtocolVersion == "v3" {
		return profile.ProtocolVersion
	}
	return ""
}

func (s *DeviceService) currentHandle(deviceID string) (device.Handle, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	handle, ok := s.handles[deviceID]
	return handle, ok
}

func (s *DeviceService) applyStoredTapSettings(ctx context.Context, deviceID string, handle device.Handle) error {
	settings, err := s.store.GetDeviceTapSettings(ctx, deviceID)
	if err != nil {
		if isStoreNotFound(err) {
			return nil
		}
		return err
	}
	if settings.ConfirmedOnDevice {
		return nil
	}
	if err := s.writeTapSettings(ctx, handle, settings); err != nil {
		return err
	}
	settings.ConfirmedOnDevice = true
	settings.UpdatedAt = s.clock.Now()
	return s.store.SaveDeviceTapSettings(ctx, settings)
}

func (s *DeviceService) writeTapSettings(ctx context.Context, handle device.Handle, settings domain.DeviceTapSettings) error {
	return s.client.SetTapSettings(ctx, handle, device.TapSettings{
		Configured: true,
		Threshold:  settings.Threshold,
		Limit:      settings.Limit,
		Latency:    settings.Latency,
		Window:     settings.Window,
	})
}

func (s *DeviceService) applyStoredLEDSettings(ctx context.Context, deviceID string, handle device.Handle) error {
	settings, err := s.store.GetDeviceLEDSettings(ctx, deviceID)
	if err != nil {
		if isStoreNotFound(err) {
			return nil
		}
		return err
	}
	if settings.ConfirmedOnDevice {
		return nil
	}
	if err := s.writeLEDSettings(ctx, handle, settings); err != nil {
		return err
	}
	settings.ConfirmedOnDevice = true
	settings.UpdatedAt = s.clock.Now()
	return s.store.SaveDeviceLEDSettings(ctx, settings)
}

func (s *DeviceService) writeLEDSettings(ctx context.Context, handle device.Handle, settings domain.DeviceLEDSettings) error {
	return s.client.SetLEDSettings(ctx, handle, device.LEDSettings{
		BrightnessPercent: settings.BrightnessPercent,
		BlinkSeconds:      settings.BlinkSeconds,
	})
}

func (s *DeviceService) applyStoredDeviceName(ctx context.Context, deviceID string, handle device.Handle) error {
	profile, err := s.store.GetDeviceProfile(ctx, deviceID)
	if err != nil {
		if isStoreNotFound(err) {
			return nil
		}
		return err
	}
	name := strings.TrimSpace(profile.DisplayName)
	if name == "" || name == profile.AdvertisedName {
		return nil
	}
	if err := domain.ValidateDeviceName(name); err != nil {
		return err
	}
	if err := s.client.SetDeviceName(ctx, handle, name); err != nil {
		return err
	}
	profile.AdvertisedName = name
	profile.LastSeenAt = s.clock.Now()
	return s.store.SaveDeviceProfile(ctx, profile)
}

func (s *DeviceService) applyStoredFacetAssignments(ctx context.Context, deviceID string, handle device.Handle) error {
	assignments, err := s.store.ListFacetAssignments(ctx, deviceID)
	if err != nil {
		return err
	}
	for _, assignment := range assignments {
		if assignment.ConfirmedOnDevice {
			continue
		}
		deviceView, err := s.client.WriteFacetConfiguration(ctx, handle, assignment)
		if err != nil {
			return err
		}
		assignment.ConfirmedOnDevice = deviceView.AssignedOnDevice
		if err := s.store.SaveFacetAssignment(ctx, assignment); err != nil {
			return err
		}
	}
	return nil
}

func (s *DeviceService) saveConnectionState(ctx context.Context, deviceID string, connection domain.ConnectionState, status string) error {
	state, err := s.store.GetDeviceState(ctx, deviceID)
	if err != nil && !isStoreNotFound(err) {
		return err
	}
	if state.DeviceID == "" {
		state.DeviceID = deviceID
	}
	state.ConnectionState = connection
	state.SystemStatus = status
	state.UpdatedAt = s.clock.Now()
	if err := s.store.SaveDeviceState(ctx, state); err != nil {
		return err
	}
	s.bus.Publish(ctx, "device.connection", state)
	return nil
}

func (s *DeviceService) savePausedState(ctx context.Context, deviceID string, paused bool) error {
	state, err := s.store.GetDeviceState(ctx, deviceID)
	if err != nil && !isStoreNotFound(err) {
		return err
	}
	if state.DeviceID == "" {
		state.DeviceID = deviceID
	}
	state.ConnectionState = domain.ConnectionConnected
	state.Paused = paused
	state.UpdatedAt = s.clock.Now()
	if err := s.store.SaveDeviceState(ctx, state); err != nil {
		return err
	}
	s.bus.Publish(ctx, "device.state", state)
	return nil
}

func (s *DeviceService) saveLockedState(ctx context.Context, deviceID string, locked bool) error {
	state, err := s.store.GetDeviceState(ctx, deviceID)
	if err != nil && !isStoreNotFound(err) {
		return err
	}
	if state.DeviceID == "" {
		state.DeviceID = deviceID
	}
	state.ConnectionState = domain.ConnectionConnected
	state.Locked = locked
	state.UpdatedAt = s.clock.Now()
	if err := s.store.SaveDeviceState(ctx, state); err != nil {
		return err
	}
	s.bus.Publish(ctx, "device.state", state)
	return nil
}

func (s *DeviceService) startEventStream(ctx context.Context, deviceID string, handle device.Handle) {
	s.mu.Lock()
	if _, ok := s.streams[deviceID]; ok {
		s.mu.Unlock()
		return
	}
	streamCtx, cancel := context.WithCancel(ctx)
	s.streams[deviceID] = cancel
	s.mu.Unlock()

	events, errs, err := s.client.Events(streamCtx, handle)
	if err != nil {
		cancel()
		s.mu.Lock()
		delete(s.streams, deviceID)
		s.mu.Unlock()
		_ = s.saveConnectionState(ctx, deviceID, domain.ConnectionConnected, "event stream unavailable")
		return
	}

	go func() {
		defer func() {
			cancel()
			s.mu.Lock()
			delete(s.streams, deviceID)
			s.mu.Unlock()
		}()
		for events != nil || errs != nil {
			select {
			case <-streamCtx.Done():
				return
			case event, ok := <-events:
				if !ok {
					events = nil
					continue
				}
				if shouldIgnoreDeviceEvent(event) {
					continue
				}
				if err := s.tracking.ApplyDeviceEvent(streamCtx, event); err != nil {
					s.bus.Publish(streamCtx, "device.error", err.Error())
				}
			case err, ok := <-errs:
				if !ok {
					errs = nil
					continue
				}
				if err != nil {
					if device.IsEventDecodeError(err) {
						s.bus.Publish(streamCtx, "device.error", err.Error())
						continue
					}
					s.removeHandle(deviceID)
					_ = s.saveConnectionState(streamCtx, deviceID, domain.ConnectionReconnecting, err.Error())
					s.closeHandleAsync(handle)
					s.bus.Publish(streamCtx, "device.error", err.Error())
					return
				}
			}
		}
		s.removeHandle(deviceID)
		_ = s.saveConnectionState(context.Background(), deviceID, domain.ConnectionReconnecting, "event stream closed")
		s.closeHandleAsync(handle)
	}()
}

const timeFlipEventsCharacteristic = "F1196F51-71A4-11E6-BDF4-0800200C9A66"

func shouldIgnoreDeviceEvent(event domain.DeviceEventRecord) bool {
	return event.Kind == "facet" && strings.EqualFold(event.RawSummary, timeFlipEventsCharacteristic)
}

func (s *DeviceService) stopEventStream(deviceID string) {
	s.mu.Lock()
	cancel := s.streams[deviceID]
	delete(s.streams, deviceID)
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *DeviceService) removeHandle(deviceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.handles, deviceID)
}

func (s *DeviceService) takeHandle(deviceID string) (device.Handle, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	handle, ok := s.handles[deviceID]
	if ok {
		delete(s.handles, deviceID)
	}
	return handle, ok
}

func (s *DeviceService) closeHandle(ctx context.Context, handle device.Handle) error {
	closeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.client.Close(closeCtx, handle)
}

func (s *DeviceService) resetDeviceState(ctx context.Context, deviceID string) error {
	state := domain.DeviceState{
		DeviceID:        deviceID,
		ConnectionState: domain.ConnectionDisconnected,
		UpdatedAt:       s.clock.Now(),
	}
	if err := s.store.SaveDeviceState(ctx, state); err != nil {
		return err
	}
	s.bus.Publish(ctx, "device.state", state)
	s.bus.Publish(ctx, "device.connection", state)
	return nil
}

func (s *DeviceService) closeHandleAsync(handle device.Handle) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.client.Close(ctx, handle)
	}()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
