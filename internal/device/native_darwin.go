//go:build darwin

package device

import (
	"io"
	"time"

	timeflip "github.com/mitchellrj/timeflip-go"
	"github.com/mitchellrj/timeflip-go/macos"
)

type NativeClientOptions struct {
	TraceBLE io.Writer
}

func NewNativeDeviceClient(timeout time.Duration) (Client, error) {
	return NewNativeDeviceClientWithOptions(timeout, NativeClientOptions{})
}

func NewNativeDeviceClientWithOptions(timeout time.Duration, opts NativeClientOptions) (Client, error) {
	return newAdapterRecoveryClient(func() (Client, error) {
		return newNativeDeviceClient(timeout, opts)
	})
}

func newNativeDeviceClient(timeout time.Duration, opts NativeClientOptions) (Client, error) {
	transport := timeflip.Transport(macos.NewTransport())
	if opts.TraceBLE != nil {
		transport = NewTracingTransport(transport, opts.TraceBLE)
	}
	client, err := NewTimeflipDeviceClient(transport, timeout)
	if err != nil {
		return nil, err
	}
	return newPreflightClient(client, defaultBluetoothAvailabilityCheck), nil
}
