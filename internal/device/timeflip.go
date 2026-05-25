package device

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	timeflip "github.com/mitchellrj/timeflip-go"
)

type TimeflipDeviceClient struct {
	client *timeflip.Client
}

type TimeflipHandle struct {
	deviceID string
	session  *timeflip.Session
}

func (h *TimeflipHandle) DeviceID() string {
	if h == nil {
		return ""
	}
	return h.deviceID
}

func NewTimeflipDeviceClient(transport timeflip.Transport, timeout time.Duration) (*TimeflipDeviceClient, error) {
	if timeout == 0 {
		timeout = domain.DefaultTimeout
	}
	client, err := timeflip.NewClient(transport, timeflip.Config{CommunicationTimeout: timeout})
	if err != nil {
		return nil, MapDeviceError(err)
	}
	return &TimeflipDeviceClient{client: client}, nil
}

func (c *TimeflipDeviceClient) Scan(ctx context.Context, includeUnsupported bool) ([]domain.DiscoveredDevice, error) {
	devices, err := c.client.ListDevices(ctx, timeflip.ScanFilter{IncludeUnsupported: includeUnsupported})
	if err != nil {
		return nil, MapDeviceError(err)
	}
	out := make([]domain.DiscoveredDevice, 0, len(devices))
	for _, d := range devices {
		out = append(out, domain.DiscoveredDevice{
			ID:        string(d.ID),
			Name:      d.Name,
			RSSI:      d.RSSI,
			Supported: d.Supported,
			Metadata:  d.Metadata,
		})
	}
	return out, nil
}

func (c *TimeflipDeviceClient) Pair(ctx context.Context, req PairRequest) (domain.PairingWorkflow, error) {
	result, err := c.client.Pair(ctx, timeflip.PairRequest{
		DeviceID:       timeflip.DeviceID(req.DeviceID),
		Password:       req.Password,
		NewPassword:    req.NewPassword,
		AllowOSPairing: req.AllowOSPairing,
		Timeout:        req.Timeout,
	})
	workflow := domain.PairingWorkflow{
		DeviceID:     string(result.DeviceID),
		CurrentStage: string(result.Stage),
		Completed:    result.Completed,
		Stages:       mapStages(result.Stages),
		ManualAction: mapManualAction(result.ManualAction),
	}
	return workflow, MapDeviceError(err)
}

func (c *TimeflipDeviceClient) Unpair(ctx context.Context, req UnpairRequest) (domain.UnpairingWorkflow, error) {
	result, err := c.client.Unpair(ctx, timeflip.UnpairRequest{
		DeviceID:         timeflip.DeviceID(req.DeviceID),
		Password:         req.Password,
		FactoryReset:     req.FactoryReset,
		AllowOSUnpairing: req.AllowOSUnpairing,
		Timeout:          req.Timeout,
	})
	workflow := domain.UnpairingWorkflow{
		DeviceID:     string(result.DeviceID),
		CurrentStage: string(result.Stage),
		Completed:    result.Completed,
		Stages:       mapStages(result.Stages),
		ManualAction: mapManualAction(result.ManualAction),
	}
	return workflow, MapDeviceError(err)
}

func (c *TimeflipDeviceClient) Connect(ctx context.Context, req ConnectRequest) (Handle, error) {
	session, err := c.client.Connect(ctx, timeflip.ConnectRequest{
		DeviceID:        timeflip.DeviceID(req.DeviceID),
		AdvertisedName:  req.AdvertisedName,
		Timeout:         req.Timeout,
		ProtocolVersion: protocolVersion(req.ProtocolVersion),
	})
	if err != nil {
		return nil, MapDeviceError(err)
	}
	return &TimeflipHandle{deviceID: req.DeviceID, session: session}, nil
}

func (c *TimeflipDeviceClient) Authorize(ctx context.Context, handle Handle, password string) error {
	h, err := asHandle(handle)
	if err != nil {
		return err
	}
	_, err = h.session.Authorize(ctx, password)
	return MapDeviceError(err)
}

func (c *TimeflipDeviceClient) ReadDeviceSnapshot(ctx context.Context, handle Handle) (domain.DeviceSnapshot, error) {
	h, err := asHandle(handle)
	if err != nil {
		return domain.DeviceSnapshot{}, err
	}
	info, _ := h.session.ReadDeviceInfo(ctx)
	battery, _ := h.session.ReadBattery(ctx)
	status, err := h.session.ReadTrackerStatus(ctx, timeflip.CommandOptions{})
	if err != nil {
		return domain.DeviceSnapshot{}, MapDeviceError(err)
	}
	system, _ := h.session.ReadSystemState(ctx)
	tap, tapErr := h.session.ReadTapSettings(ctx, timeflip.CommandOptions{})
	configs := make([]domain.FacetConfigurationView, 0, domain.FacetCount)
	for facet := uint8(1); facet <= domain.FacetCount; facet++ {
		params, err := h.session.ReadTaskParameters(ctx, timeflip.FacetID(facet), timeflip.CommandOptions{})
		view := domain.FacetConfigurationView{DeviceID: h.deviceID, Facet: facet}
		if err == nil {
			view.IsPomodoroAssignment = params.Mode == 1
			view.PomodoroLimitSeconds = params.PomodoroLimitSeconds
			view.AssignedOnDevice = params.Assigned
		}
		configs = append(configs, view)
	}
	snapshot := domain.DeviceSnapshot{
		Profile: domain.DeviceProfile{
			ID:              h.deviceID,
			DisplayName:     firstNonEmpty(info.Name, h.deviceID),
			AdvertisedName:  info.Name,
			ProtocolVersion: string(info.ProtocolVersion),
			LastSeenAt:      time.Now().UTC(),
		},
		State: domain.DeviceState{
			DeviceID:              h.deviceID,
			ConnectionState:       domain.ConnectionConnected,
			CurrentFacet:          uint8(status.CurrentFacet),
			CurrentFacetKnown:     status.CurrentFacetKnown,
			CurrentFacetUndefined: status.CurrentFacetUndefined,
			Paused:                status.PauseEnabled,
			Locked:                status.LockEnabled,
			BatteryPercent:        battery.Percentage,
			SystemStatus:          firstNonEmpty(system.StatusDescription, "unknown"),
			UpdatedAt:             time.Now().UTC(),
		},
		FacetConfigs:  configs,
		TapConfigured: tapErr == nil && tap.Configured,
	}
	if tapErr == nil && tap.Configured {
		snapshot.TapSettings = domain.DeviceTapSettings{
			DeviceID:          h.deviceID,
			Threshold:         tap.Threshold,
			Limit:             tap.Limit,
			Latency:           tap.Latency,
			Window:            tap.Window,
			ConfirmedOnDevice: true,
			UpdatedAt:         time.Now().UTC(),
		}
	}
	return snapshot, nil
}

func (c *TimeflipDeviceClient) WriteFacetConfiguration(ctx context.Context, handle Handle, assignment domain.FacetAssignment) (domain.FacetConfigurationView, error) {
	h, err := asHandle(handle)
	if err != nil {
		return domain.FacetConfigurationView{}, err
	}
	if assignment.TaskColorSnapshot != "" {
		r, g, b := parseHexColor(assignment.TaskColorSnapshot)
		if _, err := h.session.SetFacetColor(ctx, timeflip.FacetID(assignment.Facet), timeflip.RGB{R: r, G: g, B: b}, timeflip.CommandOptions{}); err != nil {
			return domain.FacetConfigurationView{
				DeviceID:             assignment.DeviceID,
				Facet:                assignment.Facet,
				TaskID:               assignment.TaskID,
				Label:                assignment.TaskLabelSnapshot,
				Icon:                 assignment.TaskIconSnapshot,
				Color:                assignment.TaskColorSnapshot,
				IsPauseAssignment:    assignment.IsPauseAssignment,
				IsPomodoroAssignment: assignment.IsPomodoroAssignment,
				PomodoroLimitSeconds: assignment.PomodoroLimitSeconds,
			}, MapDeviceError(err)
		}
	}
	params := timeflip.TaskParameters{
		Facet:                timeflip.FacetID(assignment.Facet),
		Assigned:             !assignment.IsPauseAssignment,
		Mode:                 taskMode(assignment),
		PomodoroLimitSeconds: assignment.PomodoroLimitSeconds,
	}
	if _, err := h.session.SetTaskParameters(ctx, params, timeflip.CommandOptions{}); err != nil {
		return domain.FacetConfigurationView{}, MapDeviceError(err)
	}
	readback, err := h.session.ReadTaskParameters(ctx, timeflip.FacetID(assignment.Facet), timeflip.CommandOptions{})
	view := domain.FacetConfigurationView{
		DeviceID:             assignment.DeviceID,
		Facet:                assignment.Facet,
		TaskID:               assignment.TaskID,
		Label:                assignment.TaskLabelSnapshot,
		Icon:                 assignment.TaskIconSnapshot,
		Color:                assignment.TaskColorSnapshot,
		IsPauseAssignment:    assignment.IsPauseAssignment,
		IsPomodoroAssignment: assignment.IsPomodoroAssignment,
		PomodoroLimitSeconds: assignment.PomodoroLimitSeconds,
		AssignedOnDevice:     err == nil && readback.Assigned == !assignment.IsPauseAssignment && readback.Mode == taskMode(assignment),
	}
	if err != nil {
		return view, MapDeviceError(err)
	}
	return view, nil
}

func taskMode(assignment domain.FacetAssignment) uint8 {
	if assignment.IsPomodoroAssignment {
		return 1
	}
	return 0
}

func (c *TimeflipDeviceClient) SetPause(ctx context.Context, handle Handle, enabled bool) error {
	h, err := asHandle(handle)
	if err != nil {
		return err
	}
	_, err = h.session.SetPause(ctx, enabled, timeflip.CommandOptions{})
	return MapDeviceError(err)
}

func (c *TimeflipDeviceClient) SetLock(ctx context.Context, handle Handle, enabled bool) error {
	h, err := asHandle(handle)
	if err != nil {
		return err
	}
	_, err = h.session.SetLock(ctx, enabled, timeflip.CommandOptions{})
	return MapDeviceError(err)
}

func (c *TimeflipDeviceClient) SetAutoPause(ctx context.Context, handle Handle, minutes uint16) error {
	h, err := asHandle(handle)
	if err != nil {
		return err
	}
	_, err = h.session.SetAutoPause(ctx, minutes, timeflip.CommandOptions{})
	return MapDeviceError(err)
}

func (c *TimeflipDeviceClient) SetTapSettings(ctx context.Context, handle Handle, settings TapSettings) error {
	h, err := asHandle(handle)
	if err != nil {
		return err
	}
	_, err = h.session.SetTapSettings(ctx, timeflip.TapSettings{
		Configured: settings.Configured,
		Threshold:  settings.Threshold,
		Limit:      settings.Limit,
		Latency:    settings.Latency,
		Window:     settings.Window,
	}, timeflip.CommandOptions{})
	return MapDeviceError(err)
}

func (c *TimeflipDeviceClient) SetLEDSettings(ctx context.Context, handle Handle, settings LEDSettings) error {
	h, err := asHandle(handle)
	if err != nil {
		return err
	}
	_, err = h.session.SetLED(ctx, settings.BrightnessPercent, settings.BlinkSeconds, timeflip.CommandOptions{})
	return MapDeviceError(err)
}

func (c *TimeflipDeviceClient) SetDeviceName(ctx context.Context, handle Handle, name string) error {
	h, err := asHandle(handle)
	if err != nil {
		return err
	}
	_, err = h.session.SetName(ctx, name, timeflip.CommandOptions{})
	return MapDeviceError(err)
}

func (c *TimeflipDeviceClient) ReadHistory(ctx context.Context, handle Handle, req HistoryRequest) ([]domain.DeviceEventRecord, error) {
	h, err := asHandle(handle)
	if err != nil {
		return nil, err
	}
	entries, err := h.session.ReadHistory(ctx, timeflip.HistoryRequest{StartEvent: req.StartEvent, All: req.All})
	if err != nil {
		return nil, MapDeviceError(err)
	}
	out := make([]domain.DeviceEventRecord, 0, len(entries))
	for _, entry := range entries {
		out = append(out, domain.DeviceEventRecord{
			ID:          domain.NewID("event"),
			DeviceID:    h.deviceID,
			Kind:        "history",
			Facet:       uint8(entry.Facet),
			Pause:       entry.Pause,
			EventNumber: entry.EventNumber,
			OccurredAt:  time.Unix(int64(entry.MomentUnix), 0).UTC(),
			Source:      "device_history",
			RawSummary:  fmt.Sprintf("duration=%d previous=%d", entry.DurationSeconds, entry.PreviousEventNumber),
		})
	}
	return out, nil
}

func (c *TimeflipDeviceClient) Events(ctx context.Context, handle Handle) (<-chan domain.DeviceEventRecord, <-chan error, error) {
	h, err := asHandle(handle)
	if err != nil {
		return nil, nil, err
	}
	events, errs, err := h.session.Events(ctx, timeflip.EventOptions{Buffer: 16, IncludeHistory: false})
	if err != nil {
		return nil, nil, MapDeviceError(err)
	}
	out := make(chan domain.DeviceEventRecord, 16)
	go func() {
		defer close(out)
		for event := range events {
			out <- mapEvent(h.deviceID, event)
		}
	}()
	return out, errs, nil
}

func (c *TimeflipDeviceClient) Close(ctx context.Context, handle Handle) error {
	h, err := asHandle(handle)
	if err != nil {
		return err
	}
	return MapDeviceError(h.session.Close(ctx))
}

func IsEventDecodeError(err error) bool {
	var opErr *timeflip.OperationError
	return errors.As(err, &opErr) && opErr.Operation == "events" && errors.Is(err, timeflip.ErrProtocol)
}

func MapDeviceError(err error) error {
	if err == nil {
		return nil
	}
	code := domain.ErrDeviceTimeout
	message := "Device operation failed."
	lower := strings.ToLower(err.Error())
	if errors.Is(err, context.DeadlineExceeded) {
		message = "Device operation timed out."
	}
	if strings.Contains(lower, "timeout enabling centralmanager") {
		code = domain.ErrBluetoothUnavailable
		message = "Bluetooth is unavailable or turned off."
	}
	if strings.Contains(lower, "already calling enable function") {
		code = domain.ErrBluetoothUnavailable
		message = "Bluetooth is still starting. Try connecting again in a moment."
	}
	if errors.Is(err, timeflip.ErrAuthorizationFailed) {
		code = domain.ErrAuthorizationFailed
		message = "TimeFlip authorization failed."
	}
	if errors.Is(err, timeflip.ErrUnsupportedOperation) {
		code = domain.ErrUnsupportedOperation
		message = "This device operation is not supported here."
	}
	return domain.DeviceWorkflowError{AppError: domain.NewAppError(code, message, err.Error(), err)}
}

func asHandle(handle Handle) (*TimeflipHandle, error) {
	h, ok := handle.(*TimeflipHandle)
	if !ok || h == nil || h.session == nil {
		return nil, domain.DeviceWorkflowError{AppError: domain.NewAppError(domain.ErrDeviceNotFound, "Device session is not active.", "invalid device handle", nil)}
	}
	return h, nil
}

func mapStages(stages []timeflip.StageResult) []domain.StageResult {
	out := make([]domain.StageResult, 0, len(stages))
	for _, stage := range stages {
		item := domain.StageResult{
			Stage:        stage.Stage,
			Completed:    stage.Completed,
			ManualAction: mapManualAction(stage.ManualAction),
		}
		if stage.Err != nil {
			item.Error = domain.RedactSensitive(stage.Err.Error())
		}
		out = append(out, item)
	}
	return out
}

func mapManualAction(action *timeflip.ManualAction) *domain.ManualAction {
	if action == nil {
		return nil
	}
	return &domain.ManualAction{
		Kind:        string(action.Kind),
		Description: action.Description,
		Inputs:      action.Inputs,
	}
}

func protocolVersion(value string) timeflip.ProtocolVersion {
	switch value {
	case string(timeflip.ProtocolV3):
		return timeflip.ProtocolV3
	case string(timeflip.ProtocolV4):
		return timeflip.ProtocolV4
	default:
		return timeflip.ProtocolAuto
	}
}

func mapEvent(deviceID string, event timeflip.Event) domain.DeviceEventRecord {
	record := domain.DeviceEventRecord{
		ID:         domain.NewID("event"),
		DeviceID:   deviceID,
		Kind:       string(event.Kind),
		OccurredAt: event.ReceivedAt,
		Source:     "device_stream",
		RawSummary: string(event.Source),
	}
	if record.OccurredAt.IsZero() {
		record.OccurredAt = time.Now().UTC()
	}
	switch payload := event.Payload.(type) {
	case timeflip.FacetEvent:
		record.Facet = uint8(payload.Facet)
	case timeflip.DoubleTapEvent:
		record.Facet = uint8(payload.Facet)
		record.Pause = payload.Pause
	case timeflip.PauseStateEvent:
		record.Pause = payload.Paused
		if payload.Paused {
			record.Kind = "pause"
		} else {
			record.Kind = "resume"
		}
	case timeflip.HistoryEntry:
		record.Facet = uint8(payload.Facet)
		record.Pause = payload.Pause
		record.EventNumber = payload.EventNumber
		if payload.MomentUnix > 0 {
			record.OccurredAt = time.Unix(int64(payload.MomentUnix), 0).UTC()
		}
	}
	return record
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parseHexColor(color string) (uint16, uint16, uint16) {
	if len(color) != 7 || color[0] != '#' {
		return 0, 0, 0
	}
	var r, g, b uint16
	_, _ = fmt.Sscanf(color, "#%02x%02x%02x", &r, &g, &b)
	return r * 0x101, g * 0x101, b * 0x101
}
