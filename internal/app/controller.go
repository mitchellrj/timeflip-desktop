package app

import (
	"context"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/services"
	"github.com/mitchellrj/timeflip-desktop/internal/store"
)

type Controller struct {
	ctx        context.Context
	store      store.Store
	config     *services.ConfigService
	devices    *services.DeviceService
	tasks      *services.TaskService
	history    *services.HistoryService
	connection *services.ConnectionManager
}

func NewController(st store.Store, config *services.ConfigService, devices *services.DeviceService, tasks *services.TaskService, history *services.HistoryService, connection *services.ConnectionManager) *Controller {
	return &Controller{store: st, config: config, devices: devices, tasks: tasks, history: history, connection: connection}
}

func (c *Controller) Startup(ctx context.Context) {
	c.ctx = ctx
	if c.connection != nil {
		_ = c.connection.Start(ctx)
	}
}

func (c *Controller) Shutdown(ctx context.Context) {
	if c.connection != nil {
		c.connection.Stop(ctx)
	}
	if c.store != nil {
		_ = c.store.Close()
	}
}

func (c *Controller) GetAppState() (services.AppState, error) {
	ctx := c.context()
	cfg, err := c.config.Load(ctx)
	if err != nil {
		return services.AppState{}, err
	}
	profiles, err := c.store.ListDeviceProfiles(ctx)
	if err != nil {
		return services.AppState{}, err
	}
	devices := make([]domain.DeviceProfileView, 0, len(profiles))
	var states []domain.DeviceState
	var tapSettings []domain.DeviceTapSettings
	var ledSettings []domain.DeviceLEDSettings
	var facets []domain.FacetConfigurationView
	for _, profile := range profiles {
		devices = append(devices, profile.View())
		if state, err := c.store.GetDeviceState(ctx, profile.ID); err == nil {
			states = append(states, state)
		}
		if settings, err := c.store.GetDeviceTapSettings(ctx, profile.ID); err == nil {
			tapSettings = append(tapSettings, settings)
		}
		if settings, err := c.store.GetDeviceLEDSettings(ctx, profile.ID); err == nil {
			ledSettings = append(ledSettings, settings)
		}
		if views, err := c.tasks.ListFacetConfiguration(ctx, profile.ID); err == nil {
			facets = append(facets, views...)
		}
	}
	tasks, err := c.store.ListTasks(ctx, false)
	if err != nil {
		return services.AppState{}, err
	}
	sessions, err := c.history.ListTaskSessions(ctx, domain.TaskSessionFilter{})
	if err != nil {
		return services.AppState{}, err
	}
	var current *domain.TaskSession
	if len(profiles) > 0 {
		current, _ = c.history.GetCurrentSession(ctx, profiles[0].ID)
	}
	var tapTuningStates []domain.TapTuningState
	if c.devices != nil {
		tapTuningStates = c.devices.TapTuningStates()
	}
	return services.AppState{Config: cfg, Devices: devices, States: states, TapSettings: tapSettings, TapTuningStates: tapTuningStates, LEDSettings: ledSettings, Tasks: tasks, Sessions: sessions, FacetConfigs: facets, CurrentSession: current}, nil
}

func (c *Controller) ScanDevices() ([]domain.DiscoveredDevice, error) {
	return c.devices.ListDevices(c.context())
}

type PairDeviceRequest struct {
	DeviceID       string `json:"deviceID"`
	Password       string `json:"password"`
	NewPassword    string `json:"newPassword"`
	AllowOSPairing bool   `json:"allowOSPairing"`
}

func (c *Controller) PairDevice(req PairDeviceRequest) (domain.PairingWorkflow, error) {
	return c.devices.PairDevice(c.context(), req.DeviceID, req.Password, req.NewPassword, req.AllowOSPairing)
}

type UnpairDeviceRequest struct {
	DeviceID         string `json:"deviceID"`
	FactoryReset     bool   `json:"factoryReset"`
	AllowOSUnpairing bool   `json:"allowOSUnpairing"`
}

func (c *Controller) UnpairDevice(req UnpairDeviceRequest) (domain.UnpairingWorkflow, error) {
	return c.devices.UnpairDevice(c.context(), req.DeviceID, req.FactoryReset, req.AllowOSUnpairing)
}

func (c *Controller) ConnectDevice(deviceID string) error {
	if c.connection != nil {
		return c.connection.ConnectDevice(c.context(), deviceID)
	}
	return c.devices.ConnectDevice(c.context(), deviceID)
}

func (c *Controller) DisconnectDevice(deviceID string) error {
	return c.devices.DisconnectDevice(c.context(), deviceID)
}

type SaveTaskRequest struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

func (c *Controller) SaveTask(req SaveTaskRequest) (domain.Task, error) {
	if req.ID == "" {
		return c.tasks.CreateTask(c.context(), req.Label, req.Icon, req.Color)
	}
	return c.tasks.UpdateTask(c.context(), domain.Task{ID: req.ID, Label: req.Label, Icon: req.Icon, Color: req.Color})
}

func (c *Controller) SaveFacetAssignment(req domain.FacetConfigurationRequest) (domain.FacetConfigurationView, error) {
	return c.devices.ConfigureFacet(c.context(), req)
}

func (c *Controller) SetPaused(deviceID string, paused bool) error {
	return c.devices.SetPaused(c.context(), deviceID, paused)
}

func (c *Controller) SetLocked(deviceID string, locked bool) error {
	return c.devices.SetLocked(c.context(), deviceID, locked)
}

type TapPauseSettingsRequest struct {
	DeviceID string `json:"deviceID"`
	Paused   bool   `json:"paused"`
}

func (c *Controller) SaveTapPauseSettings(req TapPauseSettingsRequest) error {
	return c.devices.ConfigureTapPause(c.context(), req.DeviceID, req.Paused)
}

func (c *Controller) SaveTapSettings(settings domain.DeviceTapSettings) (domain.DeviceTapSettings, error) {
	return c.devices.ConfigureTapSettings(c.context(), settings)
}

func (c *Controller) BeginTapTuning(deviceID string) (domain.TapTuningState, error) {
	return c.devices.BeginTapTuning(c.context(), deviceID)
}

func (c *Controller) PreviewTapTuningSettings(settings domain.DeviceTapSettings) (domain.TapTuningState, error) {
	return c.devices.PreviewTapTuningSettings(c.context(), settings)
}

func (c *Controller) ConfirmTapTuningSettings(settings domain.DeviceTapSettings) (domain.DeviceTapSettings, error) {
	return c.devices.ConfirmTapTuningSettings(c.context(), settings)
}

func (c *Controller) CancelTapTuning(deviceID string) (domain.TapTuningState, error) {
	return c.devices.CancelTapTuning(c.context(), deviceID)
}

func (c *Controller) ListTapTuningPresets(deviceID string) []domain.TapTuningPreset {
	return c.devices.ListTapTuningPresets(deviceID)
}

func (c *Controller) SaveLEDSettings(settings domain.DeviceLEDSettings) (domain.DeviceLEDSettings, error) {
	return c.devices.ConfigureLEDSettings(c.context(), settings)
}

type SaveDeviceNameRequest struct {
	DeviceID string `json:"deviceID"`
	Name     string `json:"name"`
}

func (c *Controller) SaveDeviceName(req SaveDeviceNameRequest) (domain.DeviceProfileView, error) {
	return c.devices.ConfigureDeviceName(c.context(), req.DeviceID, req.Name)
}

func (c *Controller) ListTaskSessions(filter domain.TaskSessionFilter) ([]domain.TaskSession, error) {
	return c.history.ListTaskSessions(c.context(), filter)
}

func (c *Controller) SaveSettings(config domain.AppConfig) error {
	return c.config.Save(c.context(), config)
}

func (c *Controller) context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}
