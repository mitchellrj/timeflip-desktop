package device

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
)

type clientFactory func() (Client, error)

type adapterRecoveryClient struct {
	mu      sync.Mutex
	inner   Client
	factory clientFactory
}

func newAdapterRecoveryClient(factory clientFactory) (Client, error) {
	inner, err := factory()
	if err != nil {
		return nil, err
	}
	return &adapterRecoveryClient{inner: inner, factory: factory}, nil
}

func (c *adapterRecoveryClient) Scan(ctx context.Context, includeUnsupported bool) ([]domain.DiscoveredDevice, error) {
	return withAdapterRecovery(c, func(inner Client) ([]domain.DiscoveredDevice, error) {
		return inner.Scan(ctx, includeUnsupported)
	})
}

func (c *adapterRecoveryClient) Pair(ctx context.Context, req PairRequest) (domain.PairingWorkflow, error) {
	return withAdapterRecovery(c, func(inner Client) (domain.PairingWorkflow, error) {
		return inner.Pair(ctx, req)
	})
}

func (c *adapterRecoveryClient) Unpair(ctx context.Context, req UnpairRequest) (domain.UnpairingWorkflow, error) {
	return c.current().Unpair(ctx, req)
}

func (c *adapterRecoveryClient) Connect(ctx context.Context, req ConnectRequest) (Handle, error) {
	return withAdapterRecovery(c, func(inner Client) (Handle, error) {
		return inner.Connect(ctx, req)
	})
}

func (c *adapterRecoveryClient) Authorize(ctx context.Context, handle Handle, password string) error {
	return c.current().Authorize(ctx, handle, password)
}

func (c *adapterRecoveryClient) ReadDeviceSnapshot(ctx context.Context, handle Handle) (domain.DeviceSnapshot, error) {
	return c.current().ReadDeviceSnapshot(ctx, handle)
}

func (c *adapterRecoveryClient) WriteFacetConfiguration(ctx context.Context, handle Handle, assignment domain.FacetAssignment) (domain.FacetConfigurationView, error) {
	return c.current().WriteFacetConfiguration(ctx, handle, assignment)
}

func (c *adapterRecoveryClient) SetPause(ctx context.Context, handle Handle, enabled bool) error {
	return c.current().SetPause(ctx, handle, enabled)
}

func (c *adapterRecoveryClient) SetLock(ctx context.Context, handle Handle, enabled bool) error {
	return c.current().SetLock(ctx, handle, enabled)
}

func (c *adapterRecoveryClient) SetAutoPause(ctx context.Context, handle Handle, minutes uint16) error {
	return c.current().SetAutoPause(ctx, handle, minutes)
}

func (c *adapterRecoveryClient) SetTapSettings(ctx context.Context, handle Handle, settings TapSettings) error {
	return c.current().SetTapSettings(ctx, handle, settings)
}

func (c *adapterRecoveryClient) SetLEDSettings(ctx context.Context, handle Handle, settings LEDSettings) error {
	return c.current().SetLEDSettings(ctx, handle, settings)
}

func (c *adapterRecoveryClient) SetDeviceName(ctx context.Context, handle Handle, name string) error {
	return c.current().SetDeviceName(ctx, handle, name)
}

func (c *adapterRecoveryClient) ReadHistory(ctx context.Context, handle Handle, req HistoryRequest) ([]domain.DeviceEventRecord, error) {
	return c.current().ReadHistory(ctx, handle, req)
}

func (c *adapterRecoveryClient) Events(ctx context.Context, handle Handle) (<-chan domain.DeviceEventRecord, <-chan error, error) {
	return c.current().Events(ctx, handle)
}

func (c *adapterRecoveryClient) Close(ctx context.Context, handle Handle) error {
	return c.current().Close(ctx, handle)
}

func (c *adapterRecoveryClient) current() Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inner
}

func (c *adapterRecoveryClient) rebuild() error {
	inner, err := c.factory()
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.inner = inner
	c.mu.Unlock()
	return nil
}

func withAdapterRecovery[T any](c *adapterRecoveryClient, op func(Client) (T, error)) (T, error) {
	result, err := op(c.current())
	if err == nil {
		return result, nil
	}
	if !adapterEnableFailed(err) {
		return result, err
	}
	if rebuildErr := c.rebuild(); rebuildErr != nil {
		return result, errors.Join(err, rebuildErr)
	}
	if !adapterAlreadyCallingEnable(err) {
		return result, err
	}
	return op(c.current())
}

func adapterEnableFailed(err error) bool {
	return adapterAlreadyCallingEnable(err) || adapterEnableTimedOut(err)
}

func adapterAlreadyCallingEnable(err error) bool {
	return errorTextContains(err, "already calling enable function")
}

func adapterEnableTimedOut(err error) bool {
	return errorTextContains(err, "timeout enabling centralmanager")
}

func errorTextContains(err error, needle string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), needle)
}
