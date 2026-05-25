package device

import (
	"context"
	"errors"
	"testing"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
)

func TestPreflightClientChecksAvailabilityBeforeConnect(t *testing.T) {
	checkErr := domain.DeviceWorkflowError{AppError: domain.NewAppError(domain.ErrBluetoothUnavailable, "Bluetooth is turned off.", "test", nil)}
	client := newPreflightClient(&recordingClient{}, func(context.Context) error { return checkErr })

	if _, err := client.Connect(context.Background(), ConnectRequest{DeviceID: "d1"}); !errors.Is(err, checkErr) {
		t.Fatalf("expected preflight error, got %v", err)
	}
	inner := client.(preflightClient).inner.(*recordingClient)
	if inner.connects != 0 {
		t.Fatalf("connect should not be attempted when preflight fails")
	}
}

func TestPreflightClientAllowsConnectWhenAvailable(t *testing.T) {
	inner := &recordingClient{}
	client := newPreflightClient(inner, func(context.Context) error { return nil })

	if _, err := client.Connect(context.Background(), ConnectRequest{DeviceID: "d1"}); err != nil {
		t.Fatal(err)
	}
	if inner.connects != 1 {
		t.Fatalf("expected one connect attempt, got %d", inner.connects)
	}
}

type recordingClient struct {
	connects   int
	connectErr error
}

type recordingHandle string

func (h recordingHandle) DeviceID() string { return string(h) }

func (c *recordingClient) Scan(context.Context, bool) ([]domain.DiscoveredDevice, error) {
	return nil, nil
}

func (c *recordingClient) Pair(context.Context, PairRequest) (domain.PairingWorkflow, error) {
	return domain.PairingWorkflow{}, nil
}

func (c *recordingClient) Unpair(context.Context, UnpairRequest) (domain.UnpairingWorkflow, error) {
	return domain.UnpairingWorkflow{}, nil
}

func (c *recordingClient) Connect(context.Context, ConnectRequest) (Handle, error) {
	c.connects++
	if c.connectErr != nil {
		return nil, c.connectErr
	}
	return recordingHandle("d1"), nil
}

func (c *recordingClient) Authorize(context.Context, Handle, string) error {
	return nil
}

func (c *recordingClient) ReadDeviceSnapshot(context.Context, Handle) (domain.DeviceSnapshot, error) {
	return domain.DeviceSnapshot{}, nil
}

func (c *recordingClient) WriteFacetConfiguration(context.Context, Handle, domain.FacetAssignment) (domain.FacetConfigurationView, error) {
	return domain.FacetConfigurationView{}, nil
}

func (c *recordingClient) SetPause(context.Context, Handle, bool) error {
	return nil
}

func (c *recordingClient) SetLock(context.Context, Handle, bool) error {
	return nil
}

func (c *recordingClient) SetAutoPause(context.Context, Handle, uint16) error {
	return nil
}

func (c *recordingClient) SetTapSettings(context.Context, Handle, TapSettings) error {
	return nil
}

func (c *recordingClient) SetLEDSettings(context.Context, Handle, LEDSettings) error {
	return nil
}

func (c *recordingClient) SetDeviceName(context.Context, Handle, string) error {
	return nil
}

func (c *recordingClient) ReadHistory(context.Context, Handle, HistoryRequest) ([]domain.DeviceEventRecord, error) {
	return nil, nil
}

func (c *recordingClient) Events(context.Context, Handle) (<-chan domain.DeviceEventRecord, <-chan error, error) {
	return nil, nil, nil
}

func (c *recordingClient) Close(context.Context, Handle) error {
	return nil
}
