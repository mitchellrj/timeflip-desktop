//go:build !darwin

package device

import (
	"context"
	"io"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
)

type UnsupportedClient struct{}

type NativeClientOptions struct {
	TraceBLE io.Writer
}

func NewNativeDeviceClient(time.Duration) (Client, error) {
	return UnsupportedClient{}, nil
}

func NewNativeDeviceClientWithOptions(time.Duration, NativeClientOptions) (Client, error) {
	return UnsupportedClient{}, nil
}

func (UnsupportedClient) Scan(context.Context, bool) ([]domain.DiscoveredDevice, error) {
	return nil, unsupported()
}
func (UnsupportedClient) Pair(context.Context, PairRequest) (domain.PairingWorkflow, error) {
	return domain.PairingWorkflow{}, unsupported()
}
func (UnsupportedClient) Unpair(context.Context, UnpairRequest) (domain.UnpairingWorkflow, error) {
	return domain.UnpairingWorkflow{}, unsupported()
}
func (UnsupportedClient) Connect(context.Context, ConnectRequest) (Handle, error) {
	return nil, unsupported()
}
func (UnsupportedClient) Authorize(context.Context, Handle, string) error { return unsupported() }
func (UnsupportedClient) ReadDeviceSnapshot(context.Context, Handle) (domain.DeviceSnapshot, error) {
	return domain.DeviceSnapshot{}, unsupported()
}
func (UnsupportedClient) WriteFacetConfiguration(context.Context, Handle, domain.FacetAssignment) (domain.FacetConfigurationView, error) {
	return domain.FacetConfigurationView{}, unsupported()
}
func (UnsupportedClient) SetPause(context.Context, Handle, bool) error       { return unsupported() }
func (UnsupportedClient) SetLock(context.Context, Handle, bool) error        { return unsupported() }
func (UnsupportedClient) SetAutoPause(context.Context, Handle, uint16) error { return unsupported() }
func (UnsupportedClient) SetTapSettings(context.Context, Handle, TapSettings) error {
	return unsupported()
}
func (UnsupportedClient) SetLEDSettings(context.Context, Handle, LEDSettings) error {
	return unsupported()
}
func (UnsupportedClient) SetDeviceName(context.Context, Handle, string) error { return unsupported() }
func (UnsupportedClient) ReadHistory(context.Context, Handle, HistoryRequest) ([]domain.DeviceEventRecord, error) {
	return nil, unsupported()
}
func (UnsupportedClient) Events(context.Context, Handle) (<-chan domain.DeviceEventRecord, <-chan error, error) {
	return nil, nil, unsupported()
}
func (UnsupportedClient) Close(context.Context, Handle) error { return nil }

func unsupported() error {
	return domain.DeviceWorkflowError{AppError: domain.NewAppError(domain.ErrUnsupportedOperation, "BLE runtime support is macOS-first in this build.", "native device client unsupported on this OS", nil)}
}
