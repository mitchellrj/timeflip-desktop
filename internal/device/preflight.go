package device

import (
	"context"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
)

type availabilityCheck func(context.Context) error

type preflightClient struct {
	inner Client
	check availabilityCheck
}

func newPreflightClient(inner Client, check availabilityCheck) Client {
	if check == nil {
		return inner
	}
	return preflightClient{inner: inner, check: check}
}

func (c preflightClient) Scan(ctx context.Context, includeUnsupported bool) ([]domain.DiscoveredDevice, error) {
	if err := c.check(ctx); err != nil {
		return nil, err
	}
	return c.inner.Scan(ctx, includeUnsupported)
}

func (c preflightClient) Pair(ctx context.Context, req PairRequest) (domain.PairingWorkflow, error) {
	if err := c.check(ctx); err != nil {
		return domain.PairingWorkflow{}, err
	}
	return c.inner.Pair(ctx, req)
}

func (c preflightClient) Unpair(ctx context.Context, req UnpairRequest) (domain.UnpairingWorkflow, error) {
	return c.inner.Unpair(ctx, req)
}

func (c preflightClient) Connect(ctx context.Context, req ConnectRequest) (Handle, error) {
	if err := c.check(ctx); err != nil {
		return nil, err
	}
	return c.inner.Connect(ctx, req)
}

func (c preflightClient) Authorize(ctx context.Context, handle Handle, password string) error {
	return c.inner.Authorize(ctx, handle, password)
}

func (c preflightClient) ReadDeviceSnapshot(ctx context.Context, handle Handle) (domain.DeviceSnapshot, error) {
	return c.inner.ReadDeviceSnapshot(ctx, handle)
}

func (c preflightClient) WriteFacetConfiguration(ctx context.Context, handle Handle, assignment domain.FacetAssignment) (domain.FacetConfigurationView, error) {
	return c.inner.WriteFacetConfiguration(ctx, handle, assignment)
}

func (c preflightClient) SetPause(ctx context.Context, handle Handle, enabled bool) error {
	return c.inner.SetPause(ctx, handle, enabled)
}

func (c preflightClient) SetLock(ctx context.Context, handle Handle, enabled bool) error {
	return c.inner.SetLock(ctx, handle, enabled)
}

func (c preflightClient) SetAutoPause(ctx context.Context, handle Handle, minutes uint16) error {
	return c.inner.SetAutoPause(ctx, handle, minutes)
}

func (c preflightClient) SetTapSettings(ctx context.Context, handle Handle, settings TapSettings) error {
	return c.inner.SetTapSettings(ctx, handle, settings)
}

func (c preflightClient) SetLEDSettings(ctx context.Context, handle Handle, settings LEDSettings) error {
	return c.inner.SetLEDSettings(ctx, handle, settings)
}

func (c preflightClient) SetDeviceName(ctx context.Context, handle Handle, name string) error {
	return c.inner.SetDeviceName(ctx, handle, name)
}

func (c preflightClient) ReadHistory(ctx context.Context, handle Handle, req HistoryRequest) ([]domain.DeviceEventRecord, error) {
	return c.inner.ReadHistory(ctx, handle, req)
}

func (c preflightClient) Events(ctx context.Context, handle Handle) (<-chan domain.DeviceEventRecord, <-chan error, error) {
	return c.inner.Events(ctx, handle)
}

func (c preflightClient) Close(ctx context.Context, handle Handle) error {
	return c.inner.Close(ctx, handle)
}
